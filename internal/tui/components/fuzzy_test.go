package components_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui/components"
)

func sampleFuzzyItems() []components.FuzzyItem {
	return []components.FuzzyItem{
		{Name: "go-backend", Type: nd.AssetSkill, Source: "local"},
		{Name: "python-dev", Type: nd.AssetSkill, Source: "local"},
		{Name: "review-agent", Type: nd.AssetAgent, Source: "remote"},
		{Name: "commit-msg", Type: nd.AssetCommand, Source: "local"},
	}
}

func TestFuzzyFinderNoFilter(t *testing.T) {
	f := components.NewFuzzyFinder(sampleFuzzyItems(), "")
	if len(f.Filtered) != 4 {
		t.Errorf("expected 4 items, got %d", len(f.Filtered))
	}
}

func TestFuzzyFinderPreFilter(t *testing.T) {
	f := components.NewFuzzyFinder(sampleFuzzyItems(), nd.AssetSkill)
	if len(f.Filtered) != 2 {
		t.Errorf("expected 2 skills, got %d", len(f.Filtered))
	}
}

func TestFuzzyFinderTextFilter(t *testing.T) {
	f := components.NewFuzzyFinder(sampleFuzzyItems(), "")

	// Type "go" into the input
	f, _ = f.Update(tea.KeyPressMsg{Text: "g"})
	f, _ = f.Update(tea.KeyPressMsg{Text: "o"})

	if len(f.Filtered) != 1 {
		t.Errorf("expected 1 match for 'go', got %d", len(f.Filtered))
	}
	if f.Filtered[0].Name != "go-backend" {
		t.Errorf("expected go-backend, got %q", f.Filtered[0].Name)
	}
}

func TestFuzzyFinderNavigation(t *testing.T) {
	f := components.NewFuzzyFinder(sampleFuzzyItems(), "")

	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if f.Selected != 1 {
		t.Errorf("after down: expected 1, got %d", f.Selected)
	}

	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if f.Selected != 0 {
		t.Errorf("after up: expected 0, got %d", f.Selected)
	}
}

func TestFuzzyFinderSelectedItem(t *testing.T) {
	f := components.NewFuzzyFinder(sampleFuzzyItems(), "")
	item := f.SelectedItem()
	if item == nil {
		t.Fatal("selected item should not be nil")
	}
	if item.Name != "go-backend" {
		t.Errorf("expected go-backend, got %q", item.Name)
	}
}

func TestFuzzyFinderSelectedItemEmpty(t *testing.T) {
	f := components.NewFuzzyFinder(nil, "")
	item := f.SelectedItem()
	if item != nil {
		t.Error("selected item should be nil for empty list")
	}
}

func TestFuzzyFinderView(t *testing.T) {
	f := components.NewFuzzyFinder(sampleFuzzyItems(), "")
	f.Width = 80
	f.Height = 20
	view := f.View()
	if !strings.Contains(view, "Deploy Asset") {
		t.Error("view should contain title")
	}
	if !strings.Contains(view, "4/4 matches") {
		t.Error("view should show match count")
	}
}

func TestFuzzyFinderViewLoading(t *testing.T) {
	f := components.NewFuzzyFinder(nil, "")
	f.Loading = true
	view := f.View()
	if !strings.Contains(view, "Scanning sources") {
		t.Error("loading view should show scanning message")
	}
}

func TestFuzzyFinderViewNoMatches(t *testing.T) {
	f := components.NewFuzzyFinder(sampleFuzzyItems(), "")
	// Type something that matches nothing
	f, _ = f.Update(tea.KeyPressMsg{Text: "z"})
	f, _ = f.Update(tea.KeyPressMsg{Text: "z"})
	f, _ = f.Update(tea.KeyPressMsg{Text: "z"})

	view := f.View()
	if !strings.Contains(view, "No matching") {
		t.Error("should show no matching assets message")
	}
}
