package components

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/armstrongl/nd/internal/tui"
)

// HelpBinding represents a single key-help pair.
type HelpBinding struct {
	Key  string
	Help string
}

// HelpBar renders a context-sensitive single-line help bar.
type HelpBar struct {
	Bindings []HelpBinding
	Width    int
	Styles   tui.Styles
}

// HelpForState returns the appropriate bindings for each app state.
func HelpForState(state tui.AppState) []HelpBinding {
	switch state {
	case tui.StatePicker:
		return []HelpBinding{
			{"up/down", "toggle"},
			{"left/right", "toggle"},
			{"enter", "confirm"},
		}
	case tui.StateMenu:
		return []HelpBinding{
			{"up/down", "navigate"},
			{"enter", "select"},
			{"q", "quit"},
		}
	case tui.StateDashboard:
		return []HelpBinding{
			{"up/down", "navigate"},
			{"left/right", "tabs"},
			{"enter", "expand"},
			{"d", "deploy"},
			{"r", "remove"},
			{"s", "sync"},
			{"f", "fix"},
			{"P", "profile"},
			{"W", "snapshots"},
			{"esc", "menu"},
		}
	case tui.StateDetail:
		return []HelpBinding{
			{"enter", "collapse"},
			{"esc", "collapse"},
			{"p", "pin/unpin"},
		}
	case tui.StateFuzzy:
		return []HelpBinding{
			{"up/down", "navigate"},
			{"enter", "deploy"},
			{"esc", "close"},
		}
	case tui.StateListPicker:
		return []HelpBinding{
			{"up/down", "navigate"},
			{"enter", "select"},
			{"esc", "close"},
		}
	case tui.StatePrompt:
		return []HelpBinding{
			{"enter", "submit"},
			{"esc", "cancel"},
		}
	case tui.StateConfirm:
		return []HelpBinding{
			{"y", "yes"},
			{"n", "no"},
			{"esc", "cancel"},
		}
	case tui.StateLoading:
		return []HelpBinding{
			{"esc", "cancel"},
		}
	default:
		return nil
	}
}

// View renders the help bar.
func (h HelpBar) View() string {
	if len(h.Bindings) == 0 {
		return ""
	}

	var parts []string
	totalWidth := 0

	for _, b := range h.Bindings {
		entry := tui.StyleHelpKey.Render(b.Key) + " " + h.Styles.HelpBar.Render(b.Help)
		entryWidth := lipgloss.Width(entry)

		// Truncate from right if terminal is too narrow
		if h.Width > 0 && totalWidth+entryWidth+2 > h.Width {
			break
		}

		parts = append(parts, entry)
		totalWidth += entryWidth + 2 // account for separator
	}

	return h.Styles.HelpBar.Render(strings.Join(parts, "  "))
}
