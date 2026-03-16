package components_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/tui/components"
)

func TestNewMenuItems(t *testing.T) {
	m := components.NewMenu()
	if len(m.Items) != 9 {
		t.Fatalf("expected 9 menu items, got %d", len(m.Items))
	}
	if m.Items[0].Label != "Dashboard" {
		t.Errorf("first item should be Dashboard, got %q", m.Items[0].Label)
	}
	if m.Items[8].Label != "Hooks" {
		t.Errorf("last item should be Hooks, got %q", m.Items[8].Label)
	}
}

func TestMenuNavigation(t *testing.T) {
	m := components.NewMenu()
	m.Styles = tui.DefaultStyles()

	// Move down
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.Selected != 1 {
		t.Errorf("after down: expected 1, got %d", m.Selected)
	}

	// Move up
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.Selected != 0 {
		t.Errorf("after up: expected 0, got %d", m.Selected)
	}

	// Don't go below 0
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.Selected != 0 {
		t.Errorf("should stay at 0, got %d", m.Selected)
	}

	// Don't go past last item
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	if m.Selected != len(m.Items)-1 {
		t.Errorf("should clamp at %d, got %d", len(m.Items)-1, m.Selected)
	}
}

func TestMenuView(t *testing.T) {
	m := components.NewMenu()
	m.Styles = tui.DefaultStyles()
	m.Summary = components.MenuSummary{Sources: 3, Deployed: 12, Issues: 2}

	view := m.View()
	if !strings.Contains(view, "Dashboard") {
		t.Error("view should contain Dashboard")
	}
	if !strings.Contains(view, ">") {
		t.Error("view should show cursor on selected item")
	}
	if !strings.Contains(view, "3 sources") {
		t.Error("view should show source count")
	}
	if !strings.Contains(view, "12 deployed") {
		t.Error("view should show deployed count")
	}
	if !strings.Contains(view, "2 issues") {
		t.Error("view should show issue count")
	}
}

func TestMenuViewLoading(t *testing.T) {
	m := components.NewMenu()
	m.Styles = tui.DefaultStyles()
	m.Summary = components.MenuSummary{Loading: true}

	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Error("view should show loading state")
	}
}

func TestMenuSelectedItem(t *testing.T) {
	m := components.NewMenu()
	m.Styles = tui.DefaultStyles()

	item := m.SelectedItem()
	if item.Label != "Dashboard" {
		t.Errorf("default selected item should be Dashboard, got %q", item.Label)
	}

	m.Selected = 1
	item = m.SelectedItem()
	if item.Label != "Skills" {
		t.Errorf("expected Skills, got %q", item.Label)
	}
	if item.TabIdx != 1 {
		t.Errorf("Skills tab index should be 1, got %d", item.TabIdx)
	}
}

func TestMenuTabIndices(t *testing.T) {
	m := components.NewMenu()
	for i, item := range m.Items {
		if item.Target != tui.StateDashboard {
			t.Errorf("item %d (%s): expected target StateDashboard", i, item.Label)
		}
		if item.TabIdx != i {
			t.Errorf("item %d (%s): expected TabIdx %d, got %d", i, item.Label, i, item.TabIdx)
		}
	}
}
