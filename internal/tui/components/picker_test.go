package components_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui/components"
)

func TestPickerDefaultScopeGlobal(t *testing.T) {
	p := components.NewPicker([]agent.Agent{{Name: "Claude Code"}}, false)
	scope, _ := p.Selected()
	if scope != nd.ScopeGlobal {
		t.Errorf("expected Global scope, got %v", scope)
	}
}

func TestPickerDefaultScopeProject(t *testing.T) {
	p := components.NewPicker([]agent.Agent{{Name: "Claude Code"}}, true)
	scope, _ := p.Selected()
	if scope != nd.ScopeProject {
		t.Errorf("expected Project scope, got %v", scope)
	}
}

func TestPickerAutoSelectSingleAgent(t *testing.T) {
	p := components.NewPicker([]agent.Agent{{Name: "Claude Code"}}, false)
	_, ag := p.Selected()
	if ag == nil || ag.Name != "Claude Code" {
		t.Error("should auto-select single agent")
	}
}

func TestPickerView(t *testing.T) {
	p := components.NewPicker([]agent.Agent{{Name: "Claude Code"}}, false)
	view := p.View()
	if !strings.Contains(view, "nd") {
		t.Error("picker should show nd title")
	}
	if !strings.Contains(view, "Scope") {
		t.Error("picker should show scope selection")
	}
	if !strings.Contains(view, "Claude Code") {
		t.Error("picker should show agent name")
	}
}

func TestPickerDoneOnEnter(t *testing.T) {
	p := components.NewPicker([]agent.Agent{{Name: "CC"}}, false)
	if p.Done {
		t.Error("should not be done initially")
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !p.Done {
		t.Error("should be done after enter with single agent")
	}
}

func TestPickerMultipleAgentsNavigation(t *testing.T) {
	agents := []agent.Agent{{Name: "Claude"}, {Name: "Cursor"}}
	p := components.NewPicker(agents, false)

	// Enter moves from scope to agent field
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.Done {
		t.Error("should not be done — moved to agent field")
	}

	// Down toggles agent
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, ag := p.Selected()
	if ag.Name != "Cursor" {
		t.Errorf("expected Cursor, got %q", ag.Name)
	}

	// Enter confirms
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !p.Done {
		t.Error("should be done after final enter")
	}
}

func TestPickerScopeToggle(t *testing.T) {
	p := components.NewPicker([]agent.Agent{{Name: "CC"}}, false)

	// Default is Global (index 0), toggle forward to Project
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	scope, _ := p.Selected()
	if scope != nd.ScopeProject {
		t.Errorf("after down: expected Project, got %v", scope)
	}

	// Toggle back to Global
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	scope, _ = p.Selected()
	if scope != nd.ScopeGlobal {
		t.Errorf("after up: expected Global, got %v", scope)
	}
}
