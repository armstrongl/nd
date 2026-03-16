package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui"
)

// MenuItem represents one entry in the main menu.
type MenuItem struct {
	Label  string
	Target tui.AppState
	TabIdx int // for asset type items, which tab to select
}

// MenuSummary holds the async-loaded status summary.
type MenuSummary struct {
	Sources  int
	Deployed int
	Issues   int
	Loading  bool
}

// Menu renders the main menu with navigation and a status summary (FR-028).
type Menu struct {
	Items    []MenuItem
	Selected int
	Summary  MenuSummary
	Width    int
	Height   int
	Styles   tui.Styles
	keys     tui.KeyMap
}

// NewMenu creates a menu with the standard item list.
func NewMenu() *Menu {
	return &Menu{
		Items: []MenuItem{
			{Label: "Dashboard", Target: tui.StateDashboard, TabIdx: 0},
			{Label: "Skills", Target: tui.StateDashboard, TabIdx: 1},
			{Label: "Agents", Target: tui.StateDashboard, TabIdx: 2},
			{Label: "Commands", Target: tui.StateDashboard, TabIdx: 3},
			{Label: "Output Styles", Target: tui.StateDashboard, TabIdx: 4},
			{Label: "Rules", Target: tui.StateDashboard, TabIdx: 5},
			{Label: "Context", Target: tui.StateDashboard, TabIdx: 6},
			{Label: "Plugins", Target: tui.StateDashboard, TabIdx: 7},
			{Label: "Hooks", Target: tui.StateDashboard, TabIdx: 8},
		},
		keys: tui.DefaultKeyMap(),
	}
}

// SelectedItem returns the currently selected menu item.
func (m *Menu) SelectedItem() MenuItem {
	return m.Items[m.Selected]
}

// Update handles key input for menu navigation.
func (m *Menu) Update(msg tea.Msg) (*Menu, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.Selected > 0 {
				m.Selected--
			}
		case key.Matches(msg, m.keys.Down):
			if m.Selected < len(m.Items)-1 {
				m.Selected++
			}
		}
	}
	return m, nil
}

// View renders the menu.
func (m Menu) View() string {
	var b strings.Builder

	b.WriteString("\n  nd — Main Menu\n\n")

	for i, item := range m.Items {
		cursor := "  "
		if i == m.Selected {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, item.Label))
	}

	// Status summary
	b.WriteString("\n")
	if m.Summary.Loading {
		b.WriteString(m.Styles.Loading.Render("  Loading status..."))
	} else {
		b.WriteString(fmt.Sprintf("  %d sources | %d deployed",
			m.Summary.Sources, m.Summary.Deployed))
		if m.Summary.Issues > 0 {
			b.WriteString(m.Styles.StatusBroken.Render(
				fmt.Sprintf(" | %d issues", m.Summary.Issues)))
		}
	}
	b.WriteString("\n")

	return b.String()
}
