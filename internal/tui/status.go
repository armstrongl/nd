package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
)

// statusScreen shows all deployed assets grouped by type with health indicators.
// It uses a viewport for scrollable content and tracks a cursor over selectable items.
type statusScreen struct {
	svc     Services
	styles  Styles
	isDark  bool
	entries []deploy.StatusEntry
	issues  int
	err     error
	loaded  bool

	// Viewport for scrollable content.
	vp            *viewport.Model
	cursor        int   // index into selectableLines
	selectableLines []int // line numbers (0-based) that are asset rows

	// Two-phase init: store dimensions until viewport is created.
	pendingWidth  int
	pendingHeight int

	// Custom filter (/ to activate, esc to clear).
	filter    string
	filtering bool
}

// statusLoadedMsg carries the results of loading status data.
type statusLoadedMsg struct {
	entries []deploy.StatusEntry
	err     error
}

func newStatusScreen(svc Services, styles Styles, isDark bool) *statusScreen {
	return &statusScreen{svc: svc, styles: styles, isDark: isDark}
}

func (s *statusScreen) Title() string { return "Status" }

// InputActive returns true when the filter input is active,
// suppressing global q/esc key handling.
func (s *statusScreen) InputActive() bool { return s.filtering }

// HelpItems returns context-sensitive help items for the status screen.
func (s *statusScreen) HelpItems() []HelpItem {
	return []HelpItem{
		{"d", "deploy"},
		{"r", "remove"},
		{"f", "fix"},
		{"/", "filter"},
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

// minimalKeyMap returns a viewport KeyMap with only Up/Down bindings enabled.
// All other bindings (d/f/u/b for paging) are disabled to prevent conflicts
// with status screen shortcuts (d=deploy, f=fix).
func minimalKeyMap() viewport.KeyMap {
	km := viewport.DefaultKeyMap()
	km.PageDown.Unbind()
	km.PageUp.Unbind()
	km.HalfPageDown.Unbind()
	km.HalfPageUp.Unbind()
	km.Left.Unbind()
	km.Right.Unbind()
	// Keep Up/Down but rebind to only up/down (remove j/k since we handle
	// j/k manually for cursor movement).
	km.Up.SetKeys("up")
	km.Down.SetKeys("down")
	return km
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
		// Create viewport now that data is loaded.
		if len(s.entries) > 0 {
			s.initViewport()
		}
		return s, nil

	case ScreenSizeMsg:
		if s.vp != nil {
			s.vp.SetWidth(msg.Width)
			s.vp.SetHeight(msg.Height)
		} else {
			s.pendingWidth = msg.Width
			s.pendingHeight = msg.Height
		}
		return s, nil

	case tea.KeyPressMsg:
		return s.handleKey(msg)
	}

	return s, nil
}

// handleKey processes keyboard input for cursor movement, filtering, and shortcuts.
func (s *statusScreen) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if s.filtering {
		return s.handleFilterKey(msg)
	}

	switch msg.String() {
	case "j", "down":
		if s.vp != nil && len(s.selectableLines) > 0 {
			if s.cursor < len(s.selectableLines)-1 {
				s.cursor++
				s.renderAndSync()
			}
		}
		return s, nil

	case "k", "up":
		if s.vp != nil && len(s.selectableLines) > 0 {
			if s.cursor > 0 {
				s.cursor--
				s.renderAndSync()
			}
		}
		return s, nil

	case "/":
		if s.loaded && len(s.entries) > 0 {
			s.filtering = true
		}
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

	// Forward remaining keys to viewport (for arrow-key scrolling).
	if s.vp != nil {
		vp, cmd := s.vp.Update(msg)
		s.vp = &vp
		return s, cmd
	}

	return s, nil
}

// handleFilterKey processes keyboard input while the filter is active.
func (s *statusScreen) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.filter = ""
		s.filtering = false
		s.rebuildContent()
	case "enter":
		s.filtering = false
	case "backspace":
		if len(s.filter) > 0 {
			s.filter = s.filter[:len(s.filter)-1]
			s.rebuildContent()
		}
	default:
		if msg.Text != "" {
			s.filter += msg.Text
			s.rebuildContent()
		}
	}
	return s, nil
}

// initViewport creates the viewport model and renders the initial content.
func (s *statusScreen) initViewport() {
	vp := viewport.New()
	vp.KeyMap = minimalKeyMap()
	vp.SoftWrap = true
	s.vp = &vp

	// Apply pending dimensions from ScreenSizeMsg that arrived before data loaded.
	if s.pendingWidth > 0 || s.pendingHeight > 0 {
		s.vp.SetWidth(s.pendingWidth)
		s.vp.SetHeight(s.pendingHeight)
	}

	s.rebuildContent()
}

// rebuildContent re-renders the grouped content, recalculates selectable lines,
// resets the cursor to the first match, and syncs the viewport.
func (s *statusScreen) rebuildContent() {
	if s.vp == nil {
		return
	}

	filtered := s.filteredEntries()
	content, selectable := s.buildGroupedContent(filtered)
	s.selectableLines = selectable

	// Reset cursor to first selectable item.
	s.cursor = 0

	// Highlight the cursor line and set content.
	s.vp.SetContent(s.applyHighlight(content, selectable))
	s.vp.GotoTop()
}

// renderAndSync re-applies cursor highlighting and scrolls to the cursor.
func (s *statusScreen) renderAndSync() {
	if s.vp == nil {
		return
	}

	filtered := s.filteredEntries()
	content, _ := s.buildGroupedContent(filtered)
	s.vp.SetContent(s.applyHighlight(content, s.selectableLines))

	if s.cursor >= 0 && s.cursor < len(s.selectableLines) {
		s.vp.EnsureVisible(s.selectableLines[s.cursor], 0, 0)
	}
}

// filteredEntries returns entries matching the current filter, or all entries if no filter.
func (s *statusScreen) filteredEntries() []deploy.StatusEntry {
	if s.filter == "" {
		return s.entries
	}
	lower := strings.ToLower(s.filter)
	var out []deploy.StatusEntry
	for _, e := range s.entries {
		name := strings.ToLower(e.Deployment.AssetName)
		typ := strings.ToLower(string(e.Deployment.AssetType))
		src := strings.ToLower(e.Deployment.SourceID)
		if strings.Contains(name, lower) ||
			strings.Contains(typ, lower) ||
			strings.Contains(src, lower) {
			out = append(out, e)
		}
	}
	return out
}

// buildGroupedContent renders the grouped-by-type content string and returns
// the content lines and the 0-based line numbers of selectable (asset) rows.
func (s *statusScreen) buildGroupedContent(entries []deploy.StatusEntry) ([]string, []int) {
	if len(entries) == 0 {
		return nil, nil
	}

	// Group entries by asset type.
	grouped := make(map[nd.AssetType][]deploy.StatusEntry)
	var types []nd.AssetType
	for _, e := range entries {
		t := e.Deployment.AssetType
		if _, ok := grouped[t]; !ok {
			types = append(types, t)
		}
		grouped[t] = append(grouped[t], e)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	var lines []string
	var selectable []int

	for _, t := range types {
		items := grouped[t]
		// Section header line (not selectable).
		lines = append(lines, fmt.Sprintf("  %s (%d)", s.styles.Bold.Render(string(t)), len(items)))

		for _, e := range items {
			glyph := healthGlyph(e.Health)
			styled := s.styleGlyph(glyph, e.Health)
			scope := string(e.Deployment.Scope)
			line := fmt.Sprintf("    %s  %-20s  %s  %s",
				styled, e.Deployment.AssetName, s.styles.Subtle.Render(scope), s.styles.Subtle.Render(e.Deployment.SourceID))
			selectable = append(selectable, len(lines))
			lines = append(lines, line)
		}

		// Blank line after group.
		lines = append(lines, "")
	}

	// Summary line.
	summaryLine := fmt.Sprintf("  %d deployed", len(s.entries))
	if s.issues > 0 {
		summaryLine += fmt.Sprintf("  %s", s.styles.Danger.Render(fmt.Sprintf("%d issues", s.issues)))
	}
	lines = append(lines, summaryLine)

	return lines, selectable
}

// applyHighlight returns the content string with the cursor-highlighted line
// styled using styles.Primary.
func (s *statusScreen) applyHighlight(lines []string, selectable []int) string {
	if len(selectable) == 0 || len(lines) == 0 {
		return strings.Join(lines, "\n")
	}

	highlightLine := -1
	if s.cursor >= 0 && s.cursor < len(selectable) {
		highlightLine = selectable[s.cursor]
	}

	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == highlightLine {
			fmt.Fprintf(&b, "%s %s", s.styles.Primary.Render(">"), line)
		} else if s.isSelectableLine(i, selectable) {
			fmt.Fprintf(&b, "  %s", line)
		} else {
			fmt.Fprintf(&b, "%s", line)
		}
	}
	return b.String()
}

// isSelectableLine checks whether a line index is in the selectable set.
func (s *statusScreen) isSelectableLine(lineIdx int, selectable []int) bool {
	for _, sl := range selectable {
		if sl == lineIdx {
			return true
		}
	}
	return false
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

	var b strings.Builder

	// Filter bar when active.
	if s.filtering || s.filter != "" {
		indicator := " "
		if s.filtering {
			indicator = "_"
		}
		fmt.Fprintf(&b, "  %s %s%s\n\n",
			s.styles.Subtle.Render("/"),
			s.filter,
			indicator)
	}

	if s.vp != nil {
		filtered := s.filteredEntries()
		if len(filtered) == 0 && s.filter != "" {
			fmt.Fprintf(&b, "  %s", s.styles.Subtle.Render("No assets match the filter."))
			return tea.NewView(b.String())
		}
		b.WriteString(s.vp.View())
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
