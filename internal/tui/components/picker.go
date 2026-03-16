package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/agent"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui"
)

// Picker renders a scope/agent selection widget for TUI launch.
type Picker struct {
	Scopes   []nd.Scope
	Agents   []agent.Agent
	ScopeIdx int
	AgentIdx int
	Field    int // 0=scope, 1=agent
	Done     bool
	keys     tui.KeyMap
}

// NewPicker creates a Picker with appropriate defaults.
// If only one agent is available, auto-selects it.
// Default scope: Project if hasProjectDir is true, else Global.
func NewPicker(agents []agent.Agent, hasProjectDir bool) Picker {
	scopes := []nd.Scope{nd.ScopeGlobal, nd.ScopeProject}

	scopeIdx := 0
	if hasProjectDir {
		scopeIdx = 1 // default to Project
	}

	p := Picker{
		Scopes:   scopes,
		Agents:   agents,
		ScopeIdx: scopeIdx,
		keys:     tui.DefaultKeyMap(),
	}

	// Auto-select if only one agent
	if len(agents) <= 1 {
		p.Field = 0 // only show scope
	}

	return p
}

// Selected returns the chosen scope and agent.
func (p Picker) Selected() (nd.Scope, *agent.Agent) {
	scope := p.Scopes[p.ScopeIdx]
	if len(p.Agents) == 0 {
		return scope, nil
	}
	return scope, &p.Agents[p.AgentIdx]
}

// Update handles key input for the picker.
func (p Picker) Update(msg tea.Msg) (Picker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, p.keys.Up), key.Matches(msg, p.keys.Left):
			p.toggleBack()
		case key.Matches(msg, p.keys.Down), key.Matches(msg, p.keys.Right):
			p.toggleForward()
		case key.Matches(msg, p.keys.Enter):
			if p.Field == 0 && len(p.Agents) > 1 {
				p.Field = 1 // move to agent selection
			} else {
				p.Done = true
			}
		}
	}
	return p, nil
}

func (p *Picker) toggleForward() {
	if p.Field == 0 {
		p.ScopeIdx = (p.ScopeIdx + 1) % len(p.Scopes)
	} else {
		p.AgentIdx = (p.AgentIdx + 1) % len(p.Agents)
	}
}

func (p *Picker) toggleBack() {
	if p.Field == 0 {
		p.ScopeIdx = (p.ScopeIdx - 1 + len(p.Scopes)) % len(p.Scopes)
	} else {
		p.AgentIdx = (p.AgentIdx - 1 + len(p.Agents)) % len(p.Agents)
	}
}

// View renders the picker.
func (p Picker) View() string {
	var b strings.Builder

	b.WriteString("  nd — Setup\n\n")

	// Scope selection
	scopeLabel := "Global"
	if p.Scopes[p.ScopeIdx] == nd.ScopeProject {
		scopeLabel = "Project"
	}
	if p.Field == 0 {
		b.WriteString(fmt.Sprintf("  Scope: [ %s ]  (use arrows to change)\n", scopeLabel))
	} else {
		b.WriteString(fmt.Sprintf("  Scope: %s\n", scopeLabel))
	}

	// Agent selection (only if multiple agents)
	if len(p.Agents) > 1 {
		agentName := p.Agents[p.AgentIdx].Name
		if p.Field == 1 {
			b.WriteString(fmt.Sprintf("  Agent: [ %s ]  (use arrows to change)\n", agentName))
		} else {
			b.WriteString(fmt.Sprintf("  Agent: %s\n", agentName))
		}
	} else if len(p.Agents) == 1 {
		b.WriteString(fmt.Sprintf("  Agent: %s\n", p.Agents[0].Name))
	}

	b.WriteString("\n  Press Enter to continue")
	return b.String()
}
