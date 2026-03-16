package components_test

import (
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/tui/components"
)

func TestHelpForStateDashboard(t *testing.T) {
	bindings := components.HelpForState(tui.StateDashboard)
	if len(bindings) == 0 {
		t.Fatal("dashboard should have bindings")
	}

	// Verify expected keys are present
	keys := make(map[string]bool)
	for _, b := range bindings {
		keys[b.Key] = true
	}
	for _, expected := range []string{"d", "r", "s", "f", "P", "W"} {
		if !keys[expected] {
			t.Errorf("dashboard missing key %q", expected)
		}
	}
}

func TestHelpForStatePicker(t *testing.T) {
	bindings := components.HelpForState(tui.StatePicker)
	if len(bindings) == 0 {
		t.Fatal("picker should have bindings")
	}
}

func TestHelpForStateMenu(t *testing.T) {
	bindings := components.HelpForState(tui.StateMenu)
	if len(bindings) == 0 {
		t.Fatal("menu should have bindings")
	}
	keys := make(map[string]bool)
	for _, b := range bindings {
		keys[b.Key] = true
	}
	if !keys["q"] {
		t.Error("menu should have quit key")
	}
}

func TestHelpForStateConfirm(t *testing.T) {
	bindings := components.HelpForState(tui.StateConfirm)
	keys := make(map[string]bool)
	for _, b := range bindings {
		keys[b.Key] = true
	}
	if !keys["y"] || !keys["n"] {
		t.Error("confirm should have y/n keys")
	}
}

func TestHelpBarView(t *testing.T) {
	hb := components.HelpBar{
		Bindings: components.HelpForState(tui.StateDashboard),
		Width:    120,
		Styles:   tui.DefaultStyles(),
	}
	view := hb.View()
	if view == "" {
		t.Error("help bar should not be empty")
	}
	if !strings.Contains(view, "deploy") {
		t.Error("help bar should contain deploy help text")
	}
}

func TestHelpBarViewTruncation(t *testing.T) {
	hb := components.HelpBar{
		Bindings: components.HelpForState(tui.StateDashboard),
		Width:    30,
		Styles:   tui.DefaultStyles(),
	}
	view := hb.View()
	// Should show some bindings but not all
	if view == "" {
		t.Error("truncated help bar should not be empty")
	}
}

func TestHelpBarViewEmpty(t *testing.T) {
	hb := components.HelpBar{
		Width:  120,
		Styles: tui.DefaultStyles(),
	}
	view := hb.View()
	if view != "" {
		t.Error("help bar with no bindings should be empty")
	}
}
