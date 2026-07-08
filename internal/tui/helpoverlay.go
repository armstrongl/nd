package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// HelpOverlay renders a full-screen, per-screen help panel that lists the
// active screen's keybindings. The root model toggles it with the '?' key.
type HelpOverlay struct{}

// View renders the help overlay for the given screen, constrained to width.
// When the screen implements SectionedHelpProvider the entries are grouped
// under headings; otherwise the flat defaultHelp list is shown. No rendered
// line exceeds width columns — long lines are truncated with an ellipsis so the
// overlay stays intact on narrow terminals.
func (HelpOverlay) View(s Styles, screen Screen, width, _ int) string {
	// Content is indented two columns to match the rest of the UI chrome.
	const indent = "  "
	inner := width - len(indent)
	if inner < 1 {
		inner = 1
	}

	var b strings.Builder

	title := screen.Title() + " help"
	b.WriteString(indent + s.Bold.Render(truncateToWidth(title, inner)) + "\n\n")

	if shp, ok := screen.(SectionedHelpProvider); ok {
		for i, sec := range shp.HelpSections() {
			if i > 0 {
				b.WriteString("\n")
			}
			if sec.Title != "" {
				b.WriteString(indent + s.Bold.Render(truncateToWidth(sec.Title, inner)) + "\n")
			}
			for _, line := range renderHelpItems(s, sec.Items, inner) {
				b.WriteString(indent + line + "\n")
			}
		}
	} else {
		for _, line := range renderHelpItems(s, defaultHelp(screen), inner) {
			b.WriteString(indent + line + "\n")
		}
	}

	b.WriteString("\n" + indent + s.Subtle.Render(truncateToWidth("Press ? or esc to close", inner)))
	return b.String()
}

// renderHelpItems renders each HelpItem as a "key  desc" line whose display
// width never exceeds width. Keys are padded to a common column so the
// descriptions align. Items with an empty Key render as plain note lines.
func renderHelpItems(s Styles, items []HelpItem, width int) []string {
	keyWidth := 0
	for _, it := range items {
		if w := lipgloss.Width(it.Key); w > keyWidth {
			keyWidth = w
		}
	}

	lines := make([]string, 0, len(items))
	for _, it := range items {
		if it.Key == "" {
			lines = append(lines, s.Subtle.Render(truncateToWidth(it.Desc, width)))
			continue
		}
		pad := strings.Repeat(" ", keyWidth-lipgloss.Width(it.Key))
		plain := it.Key + pad + "  " + it.Desc
		if lipgloss.Width(plain) > width {
			lines = append(lines, s.Subtle.Render(truncateToWidth(plain, width)))
			continue
		}
		lines = append(lines, s.Primary.Render(it.Key+pad)+"  "+s.Subtle.Render(it.Desc))
	}
	return lines
}

// truncateToWidth truncates s to at most width display columns, appending an
// ellipsis when truncation occurs. It is safe for multi-byte runes.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	// Reserve one column for the ellipsis.
	limit := width - 1
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if used+rw > limit {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	b.WriteString("…")
	return b.String()
}
