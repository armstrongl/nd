package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui"
)

// FuzzyItem represents one asset available for deployment.
type FuzzyItem struct {
	Name   string
	Type   nd.AssetType
	Source string
}

// FuzzyFinder renders a fuzzy search overlay for deploying assets.
type FuzzyFinder struct {
	Input     textinput.Model
	Items     []FuzzyItem
	Filtered  []FuzzyItem
	Selected  int
	Width     int
	Height    int
	PreFilter nd.AssetType // empty = all types
	Loading   bool
	keys      tui.KeyMap
}

// NewFuzzyFinder creates a fuzzy finder with the given items.
func NewFuzzyFinder(items []FuzzyItem, preFilter nd.AssetType) FuzzyFinder {
	ti := textinput.New()
	ti.Placeholder = "Search assets..."
	ti.Focus()

	f := FuzzyFinder{
		Input:     ti,
		Items:     items,
		PreFilter: preFilter,
		keys:      tui.DefaultKeyMap(),
	}
	f.filter()
	return f
}

// SelectedItem returns the currently highlighted item, or nil if none.
func (f FuzzyFinder) SelectedItem() *FuzzyItem {
	if len(f.Filtered) == 0 || f.Selected >= len(f.Filtered) {
		return nil
	}
	item := f.Filtered[f.Selected]
	return &item
}

// Update handles key input for the fuzzy finder.
func (f FuzzyFinder) Update(msg tea.Msg) (FuzzyFinder, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, f.keys.Up):
			if f.Selected > 0 {
				f.Selected--
			}
			return f, nil
		case key.Matches(msg, f.keys.Down):
			if f.Selected < len(f.Filtered)-1 {
				f.Selected++
			}
			return f, nil
		}
	}

	// Forward to text input for typing
	var cmd tea.Cmd
	f.Input, cmd = f.Input.Update(msg)
	f.filter()
	return f, cmd
}

// filter applies the current search query and pre-filter to the item list.
func (f *FuzzyFinder) filter() {
	query := strings.ToLower(f.Input.Value())
	f.Filtered = nil

	for _, item := range f.Items {
		// Apply type pre-filter
		if f.PreFilter != "" && item.Type != f.PreFilter {
			continue
		}
		// Apply text filter
		if query != "" && !strings.Contains(strings.ToLower(item.Name), query) {
			continue
		}
		f.Filtered = append(f.Filtered, item)
	}

	// Reset selection if out of bounds
	if f.Selected >= len(f.Filtered) {
		f.Selected = max(0, len(f.Filtered)-1)
	}
}

// View renders the fuzzy finder.
func (f FuzzyFinder) View() string {
	var b strings.Builder

	b.WriteString("  Deploy Asset\n\n")
	b.WriteString("  " + f.Input.View() + "\n\n")

	if f.Loading {
		b.WriteString("  Scanning sources...\n")
		return tui.StyleModal.Render(b.String())
	}

	// Match count
	b.WriteString(fmt.Sprintf("  %d/%d matches\n\n", len(f.Filtered), len(f.Items)))

	// Render visible items
	maxVisible := 10
	if f.Height > 0 {
		maxVisible = f.Height - 6 // header + input + match count + padding
		if maxVisible < 3 {
			maxVisible = 3
		}
	}

	start := 0
	if f.Selected >= maxVisible {
		start = f.Selected - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(f.Filtered) {
		end = len(f.Filtered)
	}

	for i := start; i < end; i++ {
		item := f.Filtered[i]
		cursor := "  "
		if i == f.Selected {
			cursor = "> "
		}
		b.WriteString(fmt.Sprintf("  %s%s  [%s]  %s\n", cursor, item.Name, item.Type, item.Source))
	}

	if len(f.Filtered) == 0 && !f.Loading {
		b.WriteString("  No matching assets\n")
	}

	return tui.StyleModal.Render(b.String())
}
