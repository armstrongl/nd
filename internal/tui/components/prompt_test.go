package components_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui/components"
)

func TestPromptView(t *testing.T) {
	p := components.NewPrompt("Save Snapshot", "snapshot name")
	view := p.View()
	if !strings.Contains(view, "Save Snapshot") {
		t.Error("view should contain title")
	}
	if !strings.Contains(view, "Enter to confirm") {
		t.Error("view should contain instructions")
	}
}

func TestPromptInput(t *testing.T) {
	p := components.NewPrompt("Name", "enter name")

	// Type characters
	p, _ = p.Update(tea.KeyPressMsg{Text: "m"})
	p, _ = p.Update(tea.KeyPressMsg{Text: "y"})
	p, _ = p.Update(tea.KeyPressMsg{Text: "-"})
	p, _ = p.Update(tea.KeyPressMsg{Text: "s"})

	if p.Value() != "my-s" {
		t.Errorf("expected 'my-s', got %q", p.Value())
	}
}

func TestPromptEmptyValue(t *testing.T) {
	p := components.NewPrompt("Name", "placeholder")
	if p.Value() != "" {
		t.Error("initial value should be empty")
	}
}
