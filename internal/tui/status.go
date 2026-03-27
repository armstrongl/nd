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
// It is a read-only viewport that loads data asynchronously on init.
type statusScreen struct {
	svc     Services
	styles  Styles
	isDark  bool
	entries []deploy.StatusEntry
	issues  int
	err     error
	loaded  bool
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
		return s, nil

	// M11: Handle shortcut keys shown in help bar
	case tea.KeyPressMsg:
		switch msg.String() {
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

	// Group entries by asset type.
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
		b.WriteString(fmt.Sprintf("\n  %s (%d)\n", s.styles.Bold.Render(string(t)), len(entries)))
		for _, e := range entries {
			glyph := healthGlyph(e.Health)
			styled := s.styleGlyph(glyph, e.Health)
			scope := string(e.Deployment.Scope)
			b.WriteString(fmt.Sprintf("    %s  %-20s  %s  %s\n",
				styled, e.Deployment.AssetName, s.styles.Subtle.Render(scope), s.styles.Subtle.Render(e.Deployment.SourceID)))
		}
	}

	// Summary line.
	b.WriteString(fmt.Sprintf("\n  %d deployed", len(s.entries)))
	if s.issues > 0 {
		b.WriteString(fmt.Sprintf("  %s", s.styles.Danger.Render(fmt.Sprintf("%d issues", s.issues))))
	}
	b.WriteString("\n")

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
