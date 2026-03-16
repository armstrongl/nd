package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui"
)

// Header renders a single-line status bar: scope, agent, profile, issue count.
type Header struct {
	Scope      nd.Scope
	Agent      string
	Profile    string
	IssueCount int
	SourceWarn int
	Width      int
	Styles     tui.Styles
}

// View renders the header bar.
func (h Header) View() string {
	parts := []string{"nd"}

	// Scope
	scope := "Global"
	if h.Scope == nd.ScopeProject {
		scope = "Project"
	}
	parts = append(parts, scope)

	// Agent
	if h.Agent != "" {
		parts = append(parts, h.Agent)
	}

	// Profile
	if h.Profile != "" {
		parts = append(parts, fmt.Sprintf("profile: %s", h.Profile))
	}

	// Issues
	if h.IssueCount > 0 {
		issueText := fmt.Sprintf("%d issue", h.IssueCount)
		if h.IssueCount != 1 {
			issueText += "s"
		}
		parts = append(parts, h.Styles.StatusBroken.Render(issueText))
	}

	// Source warnings
	if h.SourceWarn > 0 {
		warnText := fmt.Sprintf("%d source unavailable", h.SourceWarn)
		if h.SourceWarn != 1 {
			warnText = fmt.Sprintf("%d sources unavailable", h.SourceWarn)
		}
		parts = append(parts, h.Styles.StatusDrifted.Render(warnText))
	}

	content := strings.Join(parts, " | ")

	// Truncate if wider than terminal
	if h.Width > 0 && lipgloss.Width(content) > h.Width {
		content = truncate(content, h.Width)
	}

	return tui.StyleHeader.Width(h.Width).Render(content)
}

// truncate cuts a string to fit within maxWidth, adding ellipsis.
func truncate(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	return string(runes[:maxWidth-3]) + "..."
}
