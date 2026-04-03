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

	renderedLines []string  // content lines cached after data loads
	height        int       // terminal height, updated by tea.WindowSizeMsg
	scroll        listScroll
}

// statusLoadedMsg carries the results of loading status data.
type statusLoadedMsg struct {
	entries []deploy.StatusEntry
	err     error
}

func newStatusScreen(svc Services, styles Styles, isDark bool) *statusScreen {
	return &statusScreen{svc: svc, styles: styles, isDark: isDark}
}

func (s *statusScreen) Title() string     { return "Status" }
func (s *statusScreen) InputActive() bool { return false }

// HelpItems returns context-sensitive help items for the status screen.
func (s *statusScreen) HelpItems() []HelpItem {
	return []HelpItem{
		{"j/k", "scroll"},
		{"d", "deploy"},
		{"r", "remove"},
		{"f", "fix"},
	}
}

func (s *statusScreen) Init() tea.Cmd {
	// M10: capture svc in local to avoid capturing mutable receiver in goroutine
	svc := s.svc
	return func() tea.Msg {
		return loadStatus(svc)
	}
}

func loadStatus(svc Services) statusLoadedMsg {
	eng, err := svc.DeployEngine()
	if err != nil {
		return statusLoadedMsg{err: err}
	}
	if eng == nil {
		return statusLoadedMsg{err: fmt.Errorf("deploy engine not available")}
	}
	entries, err := eng.Status()
	return statusLoadedMsg{entries: entries, err: err}
}

func (s *statusScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusLoadedMsg:
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
		switch msg.String() {
		case "j", "down":
			s.scroll.ScrollDown(len(s.renderedLines), s.contentHeight())
		case "k", "up":
			s.scroll.ScrollUp()
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

// contentHeight returns the number of content rows that fit in the terminal.
// Returns listScrollUnlimited when height is unknown, disabling windowing.
func (s *statusScreen) contentHeight() int {
	if s.height == 0 {
		return listScrollUnlimited
	}
	h := s.height - 4 // root chrome: header + 2 blank separators + helpbar
	if h < 3 {
		h = 3
	}
	return h
}

// buildContent renders the full status list as a string (no windowing).
// Called once after data loads to populate renderedLines.
func (s *statusScreen) buildContent() string {
	grouped := make(map[nd.AssetType][]deploy.StatusEntry)
	var types []nd.AssetType
	for _, e := range s.entries {
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
	fmt.Fprintf(&b, "\n  %d deployed", len(s.entries))
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

	lines := s.renderedLines
	pageSize := s.contentHeight()
	// Reserve rows for scroll indicators so they don't push content past the
	// terminal height budget.  MoreAbove depends only on the offset (known
	// before windowing); MoreBelow is checked after we've already reduced the
	// budget, so the indicator row itself is accounted for.
	if s.scroll.MoreAbove() > 0 {
		pageSize--
	}
	if s.scroll.MoreBelow(len(lines), pageSize) > 0 {
		pageSize--
	}
	if pageSize < 1 {
		pageSize = 1
	}
	start, end := s.scroll.Window(len(lines), pageSize)

	var b strings.Builder
	if above := s.scroll.MoreAbove(); above > 0 {
		fmt.Fprintf(&b, "%s\n", scrollIndicatorLine(s.styles, "↑", above))
	}
	b.WriteString(strings.Join(lines[start:end], "\n"))
	if below := s.scroll.MoreBelow(len(lines), pageSize); below > 0 {
		fmt.Fprintf(&b, "\n%s", scrollIndicatorLine(s.styles, "↓", below))
	}

	return tea.NewView(b.String())
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
