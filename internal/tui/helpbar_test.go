package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// helpTestScreen is a minimal Screen for testing. Since screens.go is being
// created concurrently, this mock satisfies the Screen interface
// (tea.Model + Title() + InputActive()).
type helpTestScreen struct{}

func (helpTestScreen) Init() tea.Cmd                            { return nil }
func (helpTestScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return helpTestScreen{}, nil }
func (helpTestScreen) View() tea.View                           { return tea.NewView("") }
func (helpTestScreen) Title() string                            { return "test" }
func (helpTestScreen) InputActive() bool                        { return false }

// helpTestScreenWithItems is a screen that also implements HelpProvider.
type helpTestScreenWithItems struct{ helpTestScreen }

func (helpTestScreenWithItems) HelpItems() []HelpItem {
	return []HelpItem{{"f", "fix"}, {"d", "deploy"}}
}

func TestHelpBarView_BasicScreen(t *testing.T) {
	hb := HelpBar{}
	s := NewStyles(true)
	out := hb.View(s, helpTestScreen{}, 80)

	expected := []string{"esc back", "j/k navigate", "enter select", "? help", "q quit"}
	for _, want := range expected {
		if !strings.Contains(out, want) {
			t.Errorf("View output missing %q\ngot: %s", want, out)
		}
	}
}

func TestHelpBarView_HelpProviderScreen(t *testing.T) {
	hb := HelpBar{}
	s := NewStyles(true)
	out := hb.View(s, helpTestScreenWithItems{}, 80)

	// Custom items should be present
	custom := []string{"f fix", "d deploy"}
	for _, want := range custom {
		if !strings.Contains(out, want) {
			t.Errorf("View output missing custom item %q\ngot: %s", want, out)
		}
	}

	// Default items should still be present
	defaults := []string{"esc back", "j/k navigate", "enter select", "? help", "q quit"}
	for _, want := range defaults {
		if !strings.Contains(out, want) {
			t.Errorf("View output missing default item %q\ngot: %s", want, out)
		}
	}

	// Custom items must appear between "enter select" and "? help"
	enterIdx := strings.Index(out, "enter select")
	helpIdx := strings.Index(out, "? help")
	fixIdx := strings.Index(out, "f fix")
	deployIdx := strings.Index(out, "d deploy")

	if enterIdx >= fixIdx || fixIdx >= helpIdx {
		t.Errorf("custom item 'f fix' not between 'enter select' and '? help'\n"+
			"enterIdx=%d fixIdx=%d helpIdx=%d", enterIdx, fixIdx, helpIdx)
	}
	if enterIdx >= deployIdx || deployIdx >= helpIdx {
		t.Errorf("custom item 'd deploy' not between 'enter select' and '? help'\n"+
			"enterIdx=%d deployIdx=%d helpIdx=%d", enterIdx, deployIdx, helpIdx)
	}
}

// helpTestFullHelpScreen implements FullHelpProvider to replace defaults entirely.
type helpTestFullHelpScreen struct{ helpTestScreen }

func (helpTestFullHelpScreen) FullHelpItems() []HelpItem {
	return []HelpItem{{"x/space", "toggle"}, {"enter", "confirm"}}
}

func TestHelpBarView_FullHelpProviderScreen(t *testing.T) {
	hb := HelpBar{}
	s := NewStyles(true)
	out := hb.View(s, helpTestFullHelpScreen{}, 80)

	// Full help items should replace defaults.
	if !strings.Contains(out, "x/space toggle") {
		t.Errorf("View output missing 'x/space toggle'\ngot: %s", out)
	}
	if !strings.Contains(out, "enter confirm") {
		t.Errorf("View output missing 'enter confirm'\ngot: %s", out)
	}
	// Default "enter select" should NOT be present.
	if strings.Contains(out, "enter select") {
		t.Errorf("View output should not contain 'enter select' when FullHelpProvider is used\ngot: %s", out)
	}
}

func TestDefaultHelp_FullHelpProvider_OverridesDefaults(t *testing.T) {
	items := defaultHelp(helpTestFullHelpScreen{})
	if len(items) != 2 {
		t.Fatalf("defaultHelp(fullHelpProvider) returned %d items, want 2", len(items))
	}
	if items[0].Key != "x/space" || items[0].Desc != "toggle" {
		t.Errorf("first item = %+v, want {x/space toggle}", items[0])
	}
}

func TestDefaultHelp_ItemCounts(t *testing.T) {
	t.Run("basic screen returns 5 items", func(t *testing.T) {
		items := defaultHelp(helpTestScreen{})
		if got := len(items); got != 5 {
			t.Errorf("defaultHelp(basic) returned %d items, want 5", got)
		}
	})

	t.Run("provider screen returns 7 items", func(t *testing.T) {
		items := defaultHelp(helpTestScreenWithItems{})
		if got := len(items); got != 7 {
			t.Errorf("defaultHelp(provider) returned %d items, want 7", got)
		}
	})
}
