package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// RenderScrolledLines renders pre-rendered lines within a scrollable viewport,
// adding ↑/↓ indicators when items are hidden above or below. It reserves
// rows for scroll indicators so they don't push content past the budget.
func RenderScrolledLines(styles Styles, scroll *listScroll, lines []string, pageSize int) string {
	if len(lines) == 0 {
		return ""
	}

	// Reserve rows for scroll indicators.
	if scroll.MoreAbove() > 0 {
		pageSize--
	}
	if scroll.MoreBelow(len(lines), pageSize) > 0 {
		pageSize--
	}
	if pageSize < 1 {
		pageSize = 1
	}
	start, end := scroll.Window(len(lines), pageSize)

	var b strings.Builder
	if above := scroll.MoreAbove(); above > 0 {
		fmt.Fprintf(&b, "%s\n", scrollIndicatorLine(styles, "↑", above))
	}
	b.WriteString(strings.Join(lines[start:end], "\n"))
	if below := scroll.MoreBelow(len(lines), pageSize); below > 0 {
		fmt.Fprintf(&b, "\n%s", scrollIndicatorLine(styles, "↓", below))
	}
	return b.String()
}

// ContentHeight returns the number of content rows that fit in the terminal
// after subtracting chrome lines. Returns listScrollUnlimited when the
// terminal height is unknown (0), disabling windowing.
func ContentHeight(termHeight, chromeLines int) int {
	if termHeight == 0 {
		return listScrollUnlimited
	}
	h := termHeight - chromeLines
	if h < 3 {
		h = 3
	}
	return h
}

// scrollKeyResult indicates how HandleScrollKeys resolved a key press.
type scrollKeyResult int

const (
	scrollKeyUnhandled scrollKeyResult = iota
	scrollKeyHandled
	scrollKeyPopToRoot
)

// HandleScrollKeys processes j/k/enter keys for scrollable result views.
// Returns scrollKeyPopToRoot when enter is pressed, scrollKeyHandled for
// j/k navigation, or scrollKeyUnhandled for anything else.
func HandleScrollKeys(msg tea.KeyPressMsg, scroll *listScroll, totalLines, pageHeight int) scrollKeyResult {
	switch msg.String() {
	case "j", "down":
		scroll.ScrollDown(totalLines, pageHeight)
		return scrollKeyHandled
	case "k", "up":
		scroll.ScrollUp()
		return scrollKeyHandled
	case "enter":
		return scrollKeyPopToRoot
	}
	return scrollKeyUnhandled
}

// filterInput manages text filter state and key handling shared across
// screens that support '/' filtering (browse, status).
type filterInput struct {
	text   string
	active bool
}

// HandleKey processes a key press while filtering is active. Returns true
// when the filter text changed (callers should rebuild their filtered view).
func (f *filterInput) HandleKey(msg tea.KeyPressMsg) (changed bool) {
	switch msg.String() {
	case "esc":
		f.text = ""
		f.active = false
		return true
	case "enter":
		f.active = false
		return false
	case "backspace":
		if len(f.text) > 0 {
			f.text = f.text[:len(f.text)-1]
			return true
		}
		return false
	default:
		if msg.Text != "" {
			f.text += msg.Text
			return true
		}
		return false
	}
}

// Render returns the filter bar string. Returns empty when filter is inactive
// and has no text.
func (f *filterInput) Render(styles Styles) string {
	if !f.active && f.text == "" {
		return ""
	}
	if f.active {
		return fmt.Sprintf("  %s %s█\n", styles.Subtle.Render("/"), f.text)
	}
	return fmt.Sprintf("  %s %s\n", styles.Subtle.Render("/"), f.text)
}

// MatchesAny returns true if any of the given strings contain the filter text
// (case-insensitive). Returns true when filter is empty.
func (f *filterInput) MatchesAny(fields ...string) bool {
	if f.text == "" {
		return true
	}
	lower := strings.ToLower(f.text)
	for _, s := range fields {
		if strings.Contains(strings.ToLower(s), lower) {
			return true
		}
	}
	return false
}
