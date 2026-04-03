package tui

import (
	"strings"
	"testing"
)

// --- listScroll unit tests ---

func TestListScroll_ScrollDown_AdvancesOffset(t *testing.T) {
	var s listScroll
	s.ScrollDown(10, 3) // total=10, pageSize=3, max offset = 7
	if s.offset != 1 {
		t.Fatalf("offset = %d, want 1", s.offset)
	}
}

func TestListScroll_ScrollDown_ClampsAtMax(t *testing.T) {
	s := listScroll{offset: 7}
	s.ScrollDown(10, 3) // max = 10-3 = 7, already at max
	if s.offset != 7 {
		t.Fatalf("offset = %d, want 7 (clamped)", s.offset)
	}
}

func TestListScroll_ScrollDown_SmallList(t *testing.T) {
	// When total <= pageSize, max is clamped to 0 — offset never moves.
	var s listScroll
	s.ScrollDown(3, 10)
	if s.offset != 0 {
		t.Fatalf("offset = %d, want 0 (list fits in page)", s.offset)
	}
}

func TestListScroll_ScrollUp_DecrementsOffset(t *testing.T) {
	s := listScroll{offset: 3}
	s.ScrollUp()
	if s.offset != 2 {
		t.Fatalf("offset = %d, want 2", s.offset)
	}
}

func TestListScroll_ScrollUp_ClampsAtZero(t *testing.T) {
	var s listScroll // offset = 0
	s.ScrollUp()
	if s.offset != 0 {
		t.Fatalf("offset = %d, want 0 (already at top)", s.offset)
	}
}

func TestListScroll_EnsureVisible_CursorAboveViewport(t *testing.T) {
	s := listScroll{offset: 5}
	s.EnsureVisible(2, 3) // cursor 2 < offset 5 → offset becomes 2
	if s.offset != 2 {
		t.Fatalf("offset = %d, want 2", s.offset)
	}
}

func TestListScroll_EnsureVisible_CursorBelowViewport(t *testing.T) {
	s := listScroll{offset: 0}
	s.EnsureVisible(5, 3) // cursor 5 >= 0+3 → offset becomes 5-3+1 = 3
	if s.offset != 3 {
		t.Fatalf("offset = %d, want 3", s.offset)
	}
}

func TestListScroll_EnsureVisible_CursorInViewport(t *testing.T) {
	s := listScroll{offset: 2}
	s.EnsureVisible(4, 3) // cursor 4 in [2,5) → no change
	if s.offset != 2 {
		t.Fatalf("offset = %d, want 2 (unchanged)", s.offset)
	}
}

func TestListScroll_Window_ReturnsCorrectBounds(t *testing.T) {
	s := listScroll{offset: 2}
	start, end := s.Window(10, 3)
	if start != 2 || end != 5 {
		t.Fatalf("Window() = (%d, %d), want (2, 5)", start, end)
	}
}

func TestListScroll_Window_ClampsEndAtTotal(t *testing.T) {
	// offset=5 is the maximum valid start for pageSize=5, total=10.
	// 5+5=10 == total, so end clamps to 10.
	s := listScroll{offset: 5}
	start, end := s.Window(10, 5)
	if start != 5 || end != 10 {
		t.Fatalf("Window() = (%d, %d), want (5, 10)", start, end)
	}
}

// Window clamping — offset > total-pageSize (e.g. after a terminal resize
// that grows the page or shrinks the list).
func TestListScroll_Window_ClampsOffsetWhenOffsetExceedsMax(t *testing.T) {
	// offset=8, but with total=10 and pageSize=6 the max valid offset is 4.
	s := listScroll{offset: 8}
	start, end := s.Window(10, 6)
	if start != 4 || end != 10 {
		t.Fatalf("Window() = (%d, %d), want (4, 10)", start, end)
	}
	// Stored offset must be clamped too so MoreAbove/MoreBelow stay consistent.
	if s.offset != 4 {
		t.Fatalf("offset after Window() = %d, want 4 (clamped)", s.offset)
	}
}

func TestListScroll_Window_ClampsOffsetWhenOffsetExceedsTotal(t *testing.T) {
	// offset is larger than total itself — pathological but must not panic or
	// produce start > end.
	s := listScroll{offset: 15}
	start, end := s.Window(10, 3)
	if start > end {
		t.Fatalf("Window() start %d > end %d — invalid slice bounds", start, end)
	}
	// max = 10-3 = 7; offset should be clamped to 7.
	if start != 7 || end != 10 {
		t.Fatalf("Window() = (%d, %d), want (7, 10)", start, end)
	}
	if s.offset != 7 {
		t.Fatalf("offset after Window() = %d, want 7 (clamped)", s.offset)
	}
}

func TestListScroll_Window_ClampsOffsetToZeroWhenTotalZero(t *testing.T) {
	// Empty list: total=0, pageSize=anything. offset must collapse to 0.
	s := listScroll{offset: 5}
	start, end := s.Window(0, 10)
	if start != 0 || end != 0 {
		t.Fatalf("Window() = (%d, %d), want (0, 0) for empty list", start, end)
	}
	if s.offset != 0 {
		t.Fatalf("offset after Window() = %d, want 0 for empty list", s.offset)
	}
}

func TestListScroll_Window_ClampsOffset_MoreAboveConsistent(t *testing.T) {
	// After Window() clamps the offset, MoreAbove() must reflect the new value.
	s := listScroll{offset: 10}
	s.Window(8, 4) // max = 8-4 = 4; offset clamped to 4
	if got := s.MoreAbove(); got != 4 {
		t.Fatalf("MoreAbove() after clamp = %d, want 4", got)
	}
}

func TestListScroll_Window_ClampsOffset_MoreBelowConsistent(t *testing.T) {
	// After Window() clamps the offset, MoreBelow() must return 0 (at bottom).
	s := listScroll{offset: 10}
	s.Window(8, 4) // max = 4; offset clamped to 4; 4+4=8 == total → 0 below
	if got := s.MoreBelow(8, 4); got != 0 {
		t.Fatalf("MoreBelow() after clamp = %d, want 0", got)
	}
}

func TestListScroll_Window_ResizeScenario(t *testing.T) {
	// Simulate a user scrolled to offset 7 (valid for pageSize=3, total=10),
	// then the terminal is resized so pageSize becomes 8. The max valid offset
	// is now 10-8=2, so Window() must clamp offset to 2.
	s := listScroll{offset: 7}
	start, end := s.Window(10, 8)
	if start != 2 || end != 10 {
		t.Fatalf("Window() = (%d, %d), want (2, 10) after resize", start, end)
	}
	if s.offset != 2 {
		t.Fatalf("offset after resize clamp = %d, want 2", s.offset)
	}
}

func TestListScroll_MoreAbove(t *testing.T) {
	s := listScroll{offset: 4}
	if got := s.MoreAbove(); got != 4 {
		t.Fatalf("MoreAbove() = %d, want 4", got)
	}
}

func TestListScroll_MoreAbove_AtTop(t *testing.T) {
	var s listScroll
	if got := s.MoreAbove(); got != 0 {
		t.Fatalf("MoreAbove() = %d, want 0", got)
	}
}

func TestListScroll_MoreBelow_HasItems(t *testing.T) {
	s := listScroll{offset: 2}
	// total=10, pageSize=3: below = 10-(2+3) = 5
	if got := s.MoreBelow(10, 3); got != 5 {
		t.Fatalf("MoreBelow() = %d, want 5", got)
	}
}

func TestListScroll_MoreBelow_AtBottom(t *testing.T) {
	s := listScroll{offset: 7}
	// total=10, pageSize=3: below = 10-(7+3) = 0
	if got := s.MoreBelow(10, 3); got != 0 {
		t.Fatalf("MoreBelow() = %d, want 0", got)
	}
}

func TestListScroll_MoreBelow_ClampsNegative(t *testing.T) {
	s := listScroll{offset: 0}
	// pageSize > total: negative → clamped to 0
	if got := s.MoreBelow(3, 10); got != 0 {
		t.Fatalf("MoreBelow() = %d, want 0", got)
	}
}

// --- splitLines ---

func TestSplitLines_BasicLines(t *testing.T) {
	lines := splitLines("a\nb\nc\n")
	if len(lines) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(lines), lines)
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Fatalf("unexpected lines: %v", lines)
	}
}

func TestSplitLines_EmptyString(t *testing.T) {
	if got := splitLines(""); got != nil {
		t.Fatalf("splitLines(\"\") = %v, want nil", got)
	}
}

func TestSplitLines_NoTrailingNewline(t *testing.T) {
	lines := splitLines("x\ny")
	if len(lines) != 2 {
		t.Fatalf("len = %d, want 2: %v", len(lines), lines)
	}
}

// --- scrollIndicatorLine ---

func TestScrollIndicatorLine_ContainsDirectionAndCount(t *testing.T) {
	styles := NewStyles(false)
	line := scrollIndicatorLine(styles, "↑", 5)
	if !strings.Contains(line, "↑") {
		t.Errorf("indicator line missing direction glyph: %q", line)
	}
	if !strings.Contains(line, "5") {
		t.Errorf("indicator line missing count: %q", line)
	}
}
