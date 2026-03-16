package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui"
)

// Prompt renders a text input modal for snapshot naming and similar inputs.
type Prompt struct {
	Title string
	Input textinput.Model
	Width int
}

// NewPrompt creates a prompt with the given title and placeholder.
func NewPrompt(title, placeholder string) Prompt {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()

	return Prompt{
		Title: title,
		Input: ti,
	}
}

// Value returns the current input text.
func (p Prompt) Value() string {
	return p.Input.Value()
}

// Update handles key input for the prompt. Enter/Esc are handled by the parent.
func (p Prompt) Update(msg tea.Msg) (Prompt, tea.Cmd) {
	var cmd tea.Cmd
	p.Input, cmd = p.Input.Update(msg)
	return p, cmd
}

// View renders the prompt modal.
func (p Prompt) View() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s\n\n", p.Title))
	b.WriteString("  " + p.Input.View() + "\n")
	b.WriteString("\n  Enter to confirm, Esc to cancel")

	return tui.StyleModal.Render(b.String())
}
