package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui"
)

// ListPickerItem represents one entry in a list picker.
type ListPickerItem struct {
	Label       string
	Description string
	Active      bool // for current profile indicator
}

// ListPicker renders a generic selection list for profiles and snapshots.
type ListPicker struct {
	Title    string
	Items    []ListPickerItem
	Selected int
	Width    int
	Height   int
	keys     tui.KeyMap
}

// NewListPicker creates a list picker with the given items.
func NewListPicker(title string, items []ListPickerItem) ListPicker {
	return ListPicker{
		Title: title,
		Items: items,
		keys:  tui.DefaultKeyMap(),
	}
}

// SelectedItem returns the currently highlighted item, or nil if empty.
func (l ListPicker) SelectedItem() *ListPickerItem {
	if len(l.Items) == 0 || l.Selected >= len(l.Items) {
		return nil
	}
	item := l.Items[l.Selected]
	return &item
}

// Update handles key input for the list picker.
func (l ListPicker) Update(msg tea.Msg) (ListPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, l.keys.Up):
			if l.Selected > 0 {
				l.Selected--
			}
		case key.Matches(msg, l.keys.Down):
			if l.Selected < len(l.Items)-1 {
				l.Selected++
			}
		}
	}
	return l, nil
}

// View renders the list picker.
func (l ListPicker) View() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s\n\n", l.Title))

	if len(l.Items) == 0 {
		b.WriteString("  No items available.\n")
		return tui.StyleModal.Render(b.String())
	}

	for i, item := range l.Items {
		cursor := "  "
		if i == l.Selected {
			cursor = "> "
		}

		label := item.Label
		if item.Active {
			label += " (active)"
		}

		if item.Description != "" {
			b.WriteString(fmt.Sprintf("  %s%s — %s\n", cursor, label, item.Description))
		} else {
			b.WriteString(fmt.Sprintf("  %s%s\n", cursor, label))
		}
	}

	return tui.StyleModal.Render(b.String())
}
