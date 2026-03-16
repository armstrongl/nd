package components_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui/components"
)

func TestListPickerView(t *testing.T) {
	lp := components.NewListPicker("Switch Profile", []components.ListPickerItem{
		{Label: "default", Description: "3 assets", Active: true},
		{Label: "go-dev", Description: "12 assets"},
	})
	view := lp.View()
	if !strings.Contains(view, "Switch Profile") {
		t.Error("view should contain title")
	}
	if !strings.Contains(view, "default") {
		t.Error("view should contain items")
	}
	if !strings.Contains(view, "(active)") {
		t.Error("active item should be marked")
	}
}

func TestListPickerNavigation(t *testing.T) {
	lp := components.NewListPicker("Pick", []components.ListPickerItem{
		{Label: "a"},
		{Label: "b"},
		{Label: "c"},
	})

	lp, _ = lp.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if lp.Selected != 1 {
		t.Errorf("after down: expected 1, got %d", lp.Selected)
	}

	lp, _ = lp.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if lp.Selected != 2 {
		t.Errorf("after down: expected 2, got %d", lp.Selected)
	}

	// Don't go past end
	lp, _ = lp.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if lp.Selected != 2 {
		t.Errorf("should clamp at 2, got %d", lp.Selected)
	}

	lp, _ = lp.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if lp.Selected != 1 {
		t.Errorf("after up: expected 1, got %d", lp.Selected)
	}
}

func TestListPickerSelectedItem(t *testing.T) {
	lp := components.NewListPicker("Pick", []components.ListPickerItem{
		{Label: "a"},
		{Label: "b"},
	})
	item := lp.SelectedItem()
	if item == nil || item.Label != "a" {
		t.Error("should return first item")
	}

	lp.Selected = 1
	item = lp.SelectedItem()
	if item == nil || item.Label != "b" {
		t.Error("should return second item")
	}
}

func TestListPickerSelectedItemEmpty(t *testing.T) {
	lp := components.NewListPicker("Pick", nil)
	item := lp.SelectedItem()
	if item != nil {
		t.Error("should return nil for empty list")
	}
}

func TestListPickerEmptyView(t *testing.T) {
	lp := components.NewListPicker("Empty", nil)
	view := lp.View()
	if !strings.Contains(view, "No items available") {
		t.Error("should show empty state")
	}
}
