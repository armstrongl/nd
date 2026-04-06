package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// statusScreen shows all deployed assets grouped by type with health indicators.
// It is a read-only list that loads data asynchronously on init.
// Long lists are windowed to the terminal height; j/k scroll the view and
// ↑/↓ indicators appear when rows are hidden above or below the viewport.
type statusScreen struct {
	svc     Services
	styles  Styles
	isDark  bool
	entries []deploy.StatusEntry
	issues  int
	err     error
	loaded  bool

	filter filterInput

	// generation is incremented on every ScopeSwitchedMsg so that stale
	// statusLoadedMsg results from a previous scope's goroutine are discarded.
	generation uint64

	renderedLines []string  // content lines cached after data loads
	height        int       // terminal height, updated by tea.WindowSizeMsg
	scroll        listScroll
}

// statusLoadedMsg carries the results of loading status data.
type statusLoadedMsg struct {
	entries    []deploy.StatusEntry
	err        error
	generation uint64
}

func newStatusScreen(svc Services, styles Styles, isDark bool) *statusScreen {
	return &statusScreen{svc: svc, styles: styles, isDark: isDark}
}

func (s *statusScreen) Title() string     { return "Status" }
func (s *statusScreen) InputActive() bool { return s.filter.active }

// HelpItems returns context-sensitive help items for the status screen.
func (s *statusScreen) HelpItems() []HelpItem {
	if s.filter.active {
		return []HelpItem{
			{"esc", "clear filter"},
		}
	}
	items := []HelpItem{
		{"j/k", "scroll"},
		{"/", "filter"},
		{"d", "deploy"},
		{"r", "remove"},
		{"f", "fix"},
	}
	return items
}

func (s *statusScreen) Init() tea.Cmd {
	// M10: capture svc in local to avoid capturing mutable receiver in goroutine
	svc := s.svc
	gen := s.generation
	return func() tea.Msg {
		return loadStatus(svc, gen)
	}
}

func loadStatus(svc Services, gen uint64) statusLoadedMsg {
	eng, err := svc.DeployEngine()
	if err != nil {
		return statusLoadedMsg{err: err, generation: gen}
	}
	if eng == nil {
		return statusLoadedMsg{err: fmt.Errorf("deploy engine not available"), generation: gen}
	}
	entries, err := eng.Status()
	return statusLoadedMsg{entries: entries, err: err, generation: gen}
}

func (s *statusScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ScopeSwitchedMsg:
		s.generation++
		s.loaded = false
		s.entries = nil
		s.renderedLines = nil
		s.scroll = listScroll{}
		s.filter = filterInput{}
		return s, s.Init()

	case statusLoadedMsg:
		if msg.generation != s.generation {
			return s, nil // stale result from a previous scope; discard
		}
		s.loaded = true
		s.entries = msg.entries
		s.err = msg.err
		s.issues = 0
		for _, e := range s.entries {
			if e.Health != state.HealthOK {
				s.issues++
			}
		}
		s.renderedLines = splitLines(s.buildContent())
		return s, nil

	case tea.WindowSizeMsg:
		s.height = msg.Height
		return s, nil

	// M11: Handle shortcut keys shown in help bar
	case tea.KeyPressMsg:
		if s.filter.active {
			if s.filter.HandleKey(msg) {
				s.rebuildRendered()
			}
			return s, nil
		}
		switch msg.String() {
		case "j", "down":
			s.scroll.ScrollDown(len(s.renderedLines), s.contentHeight())
		case "k", "up":
			s.scroll.ScrollUp()
		case "/":
			s.filter.active = true
			return s, nil
		case "d":
			screen := newDeployScreen(s.svc, s.styles, s.isDark)
			return s, func() tea.Msg { return NavigateMsg{Screen: screen} }
		case "r":
			screen := newRemoveScreen(s.svc, s.styles, s.isDark)
			return s, func() tea.Msg { return NavigateMsg{Screen: screen} }
		case "f":
			screen := newDoctorScreen(s.svc, s.styles, s.isDark)
			return s, func() tea.Msg { return NavigateMsg{Screen: screen} }
		}
	}
	return s, nil
}

func (s *statusScreen) contentHeight() int {
	return ContentHeight(s.height, 4) // root chrome: header + 2 blank separators + helpbar
}

// filteredEntries returns entries matching the current filter.
func (s *statusScreen) filteredEntries() []deploy.StatusEntry {
	if s.filter.text == "" {
		return s.entries
	}
	var result []deploy.StatusEntry
	for _, e := range s.entries {
		if s.filter.MatchesAny(e.Deployment.AssetName, e.Deployment.SourceID) {
			result = append(result, e)
		}
	}
	return result
}

// rebuildRendered re-renders the displayed lines from filtered entries.
func (s *statusScreen) rebuildRendered() {
	s.renderedLines = splitLines(s.buildContent())
	s.scroll = listScroll{} // reset scroll position
}

// buildContent renders the status list as a string (no windowing).
// Called after data loads and on filter changes to populate renderedLines.
func (s *statusScreen) buildContent() string {
	filtered := s.filteredEntries()
	grouped := make(map[nd.AssetType][]deploy.StatusEntry)
	var types []nd.AssetType
	for _, e := range filtered {
		t := e.Deployment.AssetType
		if _, ok := grouped[t]; !ok {
			types = append(types, t)
		}
		grouped[t] = append(grouped[t], e)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	var b strings.Builder
	for _, t := range types {
		entries := grouped[t]
		fmt.Fprintf(&b, "\n  %s (%d)\n", s.styles.Bold.Render(string(t)), len(entries))
		for _, e := range entries {
			glyph := healthGlyph(e.Health)
			styled := s.styleGlyph(glyph, e.Health)
			scope := string(e.Deployment.Scope)
			fmt.Fprintf(&b, "    %s  %-20s  %s  %s\n",
				styled, e.Deployment.AssetName, s.styles.Subtle.Render(scope), s.styles.Subtle.Render(e.Deployment.SourceID))
		}
	}
	if s.filter.text != "" {
		fmt.Fprintf(&b, "\n  %d/%d matching %q", len(filtered), len(s.entries), s.filter.text)
	} else {
		fmt.Fprintf(&b, "\n  %d deployed", len(s.entries))
	}
	if s.issues > 0 {
		fmt.Fprintf(&b, "  %s", s.styles.Danger.Render(fmt.Sprintf("%d issues", s.issues)))
	}
	b.WriteString("\n")
	return b.String()
}

func (s *statusScreen) View() tea.View {
	if !s.loaded {
		return tea.NewView("  Loading...")
	}
	if s.err != nil {
		return tea.NewView(fmt.Sprintf("  Error: %s\n\n  %s",
			s.err, s.styles.Subtle.Render("Press esc to go back.")))
	}
	if len(s.entries) == 0 {
		return tea.NewView("  " + NothingDeployed())
	}

	filterLine := s.filter.Render(s.styles)
	content := RenderScrolledLines(s.styles, &s.scroll, s.renderedLines, s.contentHeight())
	return tea.NewView(filterLine + content)
}

// healthGlyph returns the text glyph for the given health status.
func healthGlyph(h state.HealthStatus) string {
	switch h {
	case state.HealthOK:
		return GlyphOK
	case state.HealthBroken:
		return GlyphBroken
	case state.HealthDrifted:
		return GlyphDrifted
	case state.HealthOrphaned:
		return GlyphOrphan
	case state.HealthMissing:
		return GlyphMissing
	default:
		return "??"
	}
}

// styleGlyph applies the appropriate color style to a health glyph.
func (s *statusScreen) styleGlyph(glyph string, h state.HealthStatus) string {
	return styleGlyphWith(s.styles, glyph, h)
}
