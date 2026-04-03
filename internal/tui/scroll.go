package tui

import (
	"fmt"
	"math"
	"strings"
)

// listScroll manages a scrollable window over a flat list.
// Only the offset (index of the first visible row) is tracked as state;
// the page size is passed per-call so callers derive it from terminal height
// at the moment of the operation without storing it redundantly.
//
// When height is unknown (e.g. before the first tea.WindowSizeMsg), callers
// should pass a very large page size (use listScrollUnlimited) so that all
// items are visible with no indicators — matching the pre-scroll behaviour.
type listScroll struct {
	offset int
}

// listScrollUnlimited is a page size that effectively disables windowing.
// Use it when the terminal height is not yet known (height == 0).
const listScrollUnlimited = math.MaxInt

// ScrollDown advances the viewport down by one row.
// total is the total number of items; pageSize is the visible row budget.
func (s *listScroll) ScrollDown(total, pageSize int) {
	max := total - pageSize
	if max < 0 {
		max = 0
	}
	if s.offset < max {
		s.offset++
	}
}

// ScrollUp moves the viewport up by one row.
func (s *listScroll) ScrollUp() {
	if s.offset > 0 {
		s.offset--
	}
}

// EnsureVisible adjusts the offset so that cursor falls within
// [offset, offset+pageSize). Call this after moving a selection cursor.
func (s *listScroll) EnsureVisible(cursor, pageSize int) {
	if cursor < s.offset {
		s.offset = cursor
	}
	if pageSize > 0 && cursor >= s.offset+pageSize {
		s.offset = cursor - pageSize + 1
	}
}

// Window returns [start, end) slice bounds for the currently visible items.
// It also clamps the stored offset to a valid range for the given total and
// pageSize, so that a terminal resize (which can shrink pageSize or total)
// never leaves the offset pointing past the end of the list.
func (s *listScroll) Window(total, pageSize int) (start, end int) {
	// Clamp offset so that the viewport cannot start beyond the last page.
	max := total - pageSize
	if max < 0 {
		max = 0
	}
	if s.offset > max {
		s.offset = max
	}
	if s.offset < 0 {
		s.offset = 0
	}

	start = s.offset
	end = s.offset + pageSize
	if end > total {
		end = total
	}
	return
}

// MoreAbove returns the count of items hidden above the current viewport.
func (s *listScroll) MoreAbove() int { return s.offset }

// MoreBelow returns the count of items hidden below the current viewport.
func (s *listScroll) MoreBelow(total, pageSize int) int {
	below := total - (s.offset + pageSize)
	if below < 0 {
		return 0
	}
	return below
}

// scrollIndicatorLine returns a rendered "↑ N more" or "↓ N more" hint line.
func scrollIndicatorLine(styles Styles, dir string, count int) string {
	return "  " + styles.Subtle.Render(fmt.Sprintf("%s %d more", dir, count))
}

// splitLines splits a content string into individual lines.
// A single trailing newline is stripped to avoid a spurious empty last element.
func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
