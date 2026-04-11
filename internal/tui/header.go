package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/armstrongl/nd/internal/state"
)

// Header renders the persistent top bar showing profile, scope, agent, and
// deployment counts. It is a plain value type — call Refresh to re-query
// service state.
type Header struct {
	Profile  string
	Scope    string
	Agent    string
	Deployed int
	Issues   int
	DryRun   bool
}

// View renders the header as a single line spanning width columns.
// Left side: profile · scope · agent (with optional [DRY RUN] prefix).
// Right side: deployment count and issue count (issues styled danger when > 0).
func (h Header) View(s Styles, width int) string {
	left := fmt.Sprintf("  %s %s %s %s %s", h.Profile, GlyphDot, h.Scope, GlyphDot, h.Agent)
	if h.DryRun {
		left = "  [DRY RUN] " + left[2:]
	}

	right := fmt.Sprintf("%d deployed  %d issues", h.Deployed, h.Issues)

	leftStyled := left
	var rightStyled string
	if h.Issues > 0 {
		rightStyled = fmt.Sprintf("%s  %s",
			s.Subtle.Render(fmt.Sprintf("%d deployed", h.Deployed)),
			s.Danger.Render(fmt.Sprintf("%d issues", h.Issues)))
	} else {
		rightStyled = s.Subtle.Render(right)
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return leftStyled + strings.Repeat(" ", gap) + rightStyled
}

// Refresh re-queries the service layer and returns an updated Header.
// It is safe to call from tea.Cmd contexts.
func (h Header) Refresh(svc Services) Header {
	if pm, err := svc.ProfileManager(); err == nil && pm != nil {
		h.Profile, _ = pm.ActiveProfile()
	}
	if h.Profile == "" {
		h.Profile = "no profile"
	}
	h.Scope = string(svc.GetScope())

	if ag, err := svc.ActiveAgent(); err == nil && ag != nil {
		h.Agent = ag.Name
	}

	h.DryRun = svc.IsDryRun()

	if eng, err := svc.DeployEngine(); err == nil && eng != nil {
		if entries, err := eng.Status(); err == nil {
			h.Deployed = len(entries)
			h.Issues = 0
			for _, e := range entries {
				if e.Health != state.HealthOK {
					h.Issues++
				}
			}
		}
	}
	return h
}
