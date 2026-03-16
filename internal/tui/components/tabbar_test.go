package components_test

import (
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/tui/components"
)

func TestTabBarDefaultTabs(t *testing.T) {
	tabs := components.DefaultTabs()
	if len(tabs) != 9 {
		t.Fatalf("expected 9 tabs, got %d", len(tabs))
	}
	expected := []string{"Overview", "Skills", "Agents", "Commands", "Output Styles", "Rules", "Context", "Plugins", "Hooks"}
	for i, tab := range tabs {
		if tab.Label != expected[i] {
			t.Errorf("tab[%d]: expected %q, got %q", i, expected[i], tab.Label)
		}
	}
}

func TestTabBarViewWide(t *testing.T) {
	tb := components.TabBar{
		Tabs:   components.DefaultTabs(),
		Active: 0,
		Width:  120,
		Styles: tui.DefaultStyles(),
	}
	view := tb.View()
	// Active tab (Overview) should be present
	// Inactive tabs render with color codes wrapping the whole label, so
	// we can check them with simple Contains. The active tab (Overview)
	// may have per-character ANSI codes from bold+underline.
	if !strings.Contains(view, "Skills") {
		t.Errorf("wide view should contain full tab names, got: %q", view)
	}
	if !strings.Contains(view, "Hooks") {
		t.Error("wide view should contain all tabs")
	}
	if !strings.Contains(view, "Output Styles") {
		t.Error("wide view should contain unabbreviated names")
	}
}

func TestTabBarViewMedium(t *testing.T) {
	tb := components.TabBar{
		Tabs:   components.DefaultTabs(),
		Active: 0,
		Width:  80,
		Styles: tui.DefaultStyles(),
	}
	view := tb.View()
	// Should use abbreviated names
	if strings.Contains(view, "Output Styles") {
		t.Error("medium view should abbreviate long tab names")
	}
}

func TestTabBarViewNarrow(t *testing.T) {
	tb := components.TabBar{
		Tabs:   components.DefaultTabs(),
		Active: 2,
		Width:  50,
		Styles: tui.DefaultStyles(),
	}
	view := tb.View()
	// Should show single tab with arrows
	if !strings.Contains(view, "<") || !strings.Contains(view, ">") {
		t.Error("narrow view should show arrow navigation")
	}
	if !strings.Contains(view, "3/9") {
		t.Error("narrow view should show position indicator")
	}
}

func TestTabBarIssueBadge(t *testing.T) {
	tabs := components.DefaultTabs()
	tabs[1].IssueCount = 3 // Skills has 3 issues
	tb := components.TabBar{
		Tabs:   tabs,
		Active: 0,
		Width:  120,
		Styles: tui.DefaultStyles(),
	}
	view := tb.View()
	if !strings.Contains(view, "(3!)") {
		t.Error("should show issue badge")
	}
}

func TestTabBarNextPrev(t *testing.T) {
	tb := components.TabBar{
		Tabs:   components.DefaultTabs(),
		Active: 0,
	}

	tb.Next()
	if tb.Active != 1 {
		t.Errorf("Next: expected 1, got %d", tb.Active)
	}

	tb.Prev()
	if tb.Active != 0 {
		t.Errorf("Prev: expected 0, got %d", tb.Active)
	}

	// Wrap around
	tb.Prev()
	if tb.Active != 8 {
		t.Errorf("Prev wrap: expected 8, got %d", tb.Active)
	}

	tb.Next()
	if tb.Active != 0 {
		t.Errorf("Next wrap: expected 0, got %d", tb.Active)
	}
}

func TestTabBarEmptyView(t *testing.T) {
	tb := components.TabBar{}
	view := tb.View()
	if view != "" {
		t.Error("empty tab bar should render empty string")
	}
}
