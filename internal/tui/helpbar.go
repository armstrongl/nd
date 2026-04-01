package tui

import "strings"

// HelpItem represents a single key binding shown in the help bar.
type HelpItem struct {
	Key  string
	Desc string
}

// HelpProvider is an optional interface screens can implement to add custom help items.
type HelpProvider interface {
	HelpItems() []HelpItem
}

// FullHelpProvider is an optional interface screens can implement to replace
// the default help items entirely. Use this when a screen needs step-specific
// help (e.g. MultiSelect uses "x/space toggle" instead of "enter select").
type FullHelpProvider interface {
	FullHelpItems() []HelpItem
}

// HelpBar renders context-sensitive help at the bottom of the TUI.
type HelpBar struct{}

// View renders the help bar for the given screen.
func (hb HelpBar) View(s Styles, screen Screen, _ int) string {
	items := defaultHelp(screen)
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = s.Subtle.Render(item.Key + " " + item.Desc)
	}
	return "  " + strings.Join(parts, "  ")
}

func defaultHelp(screen Screen) []HelpItem {
	// Allow screens to fully replace the help items.
	if fhp, ok := screen.(FullHelpProvider); ok {
		return fhp.FullHelpItems()
	}

	items := []HelpItem{
		{"esc", "back"},
		{"j/k", "navigate"},
		{"enter", "select"},
	}
	if hp, ok := screen.(HelpProvider); ok {
		items = append(items, hp.HelpItems()...)
	}
	items = append(items, HelpItem{"?", "help"}, HelpItem{"q", "quit"})
	return items
}
