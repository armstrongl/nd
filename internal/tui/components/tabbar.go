package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/armstrongl/nd/internal/tui"
)

// Tab represents one tab in the tab bar.
type Tab struct {
	Label      string
	IssueCount int
}

// TabBar renders a horizontal tab bar with issue badges.
type TabBar struct {
	Tabs   []Tab
	Active int
	Width  int
	Styles tui.Styles
}

// DefaultTabs returns the standard tab order matching spec A5 asset type order.
func DefaultTabs() []Tab {
	return []Tab{
		{Label: "Overview"},
		{Label: "Skills"},
		{Label: "Agents"},
		{Label: "Commands"},
		{Label: "Output Styles"},
		{Label: "Rules"},
		{Label: "Context"},
		{Label: "Plugins"},
		{Label: "Hooks"},
	}
}

// abbreviated returns shortened tab labels for medium-width terminals.
var abbreviated = map[string]string{
	"Overview":      "Over",
	"Skills":        "Skil",
	"Agents":        "Agnt",
	"Commands":      "Cmds",
	"Output Styles": "OutS",
	"Rules":         "Rule",
	"Context":       "Ctx",
	"Plugins":       "Plug",
	"Hooks":         "Hook",
}

// Next moves to the next tab, wrapping around.
func (t *TabBar) Next() {
	t.Active = (t.Active + 1) % len(t.Tabs)
}

// Prev moves to the previous tab, wrapping around.
func (t *TabBar) Prev() {
	t.Active = (t.Active - 1 + len(t.Tabs)) % len(t.Tabs)
}

// View renders the tab bar.
func (t TabBar) View() string {
	if len(t.Tabs) == 0 {
		return ""
	}

	// Narrow mode: single tab with arrows
	if t.Width > 0 && t.Width < 60 {
		return t.narrowView()
	}

	useAbbrev := t.Width > 0 && t.Width < 95

	var tabs []string
	for i, tab := range t.Tabs {
		label := tab.Label
		if useAbbrev {
			if abbr, ok := abbreviated[label]; ok {
				label = abbr
			}
		}

		// Add issue badge
		if tab.IssueCount > 0 {
			label += fmt.Sprintf(" (%d!)", tab.IssueCount)
		}

		if i == t.Active {
			tabs = append(tabs, tui.StyleTabActive.Render(label))
		} else {
			tabs = append(tabs, t.Styles.TabInactive.Render(label))
		}
	}

	content := strings.Join(tabs, "  ")

	// Truncate if needed
	if t.Width > 0 && lipgloss.Width(content) > t.Width {
		content = truncate(content, t.Width)
	}

	return content
}

// narrowView renders a single-tab view with navigation arrows.
func (t TabBar) narrowView() string {
	tab := t.Tabs[t.Active]
	label := tab.Label
	if tab.IssueCount > 0 {
		label += fmt.Sprintf(" (%d!)", tab.IssueCount)
	}
	return fmt.Sprintf("< %s (%d/%d) >",
		tui.StyleTabActive.Render(label),
		t.Active+1, len(t.Tabs))
}
