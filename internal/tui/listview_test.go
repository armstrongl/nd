package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRenderScrolledLines_AllVisible(t *testing.T) {
	styles := testStyles()
	scroll := listScroll{}
	lines := []string{"line1", "line2", "line3"}

	got := RenderScrolledLines(styles, &scroll, lines, 10)
	if got != "line1\nline2\nline3" {
		t.Errorf("expected all lines joined, got %q", got)
	}
}

func TestRenderScrolledLines_Empty(t *testing.T) {
	styles := testStyles()
	scroll := listScroll{}

	got := RenderScrolledLines(styles, &scroll, nil, 10)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestRenderScrolledLines_Windowed(t *testing.T) {
	styles := testStyles()
	scroll := listScroll{offset: 1}
	lines := []string{"a", "b", "c", "d", "e"}

	got := RenderScrolledLines(styles, &scroll, lines, 2)
	// With offset 1 and pageSize 2, scroll indicators consume both rows,
	// so only one content line ("b") is rendered alongside the indicators.
	if !strings.Contains(got, "b") {
		t.Errorf("expected to contain 'b', got %q", got)
	}
	if strings.Contains(got, "c") {
		t.Errorf("expected 'c' to be hidden (indicators consume page budget), got %q", got)
	}
	if !strings.Contains(got, "more") {
		t.Errorf("expected scroll indicators, got %q", got)
	}
}

func TestContentHeight_ZeroReturnsUnlimited(t *testing.T) {
	got := ContentHeight(0, 4)
	if got != listScrollUnlimited {
		t.Errorf("expected listScrollUnlimited, got %d", got)
	}
}

func TestContentHeight_SubtractsChrome(t *testing.T) {
	got := ContentHeight(30, 4)
	if got != 26 {
		t.Errorf("expected 26, got %d", got)
	}
}

func TestContentHeight_MinThree(t *testing.T) {
	got := ContentHeight(5, 4)
	if got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestHandleScrollKeys_Down(t *testing.T) {
	scroll := listScroll{}
	msg := tea.KeyPressMsg{Code: 'j'}

	result := HandleScrollKeys(msg, &scroll, 10, 5)
	if result != scrollKeyHandled {
		t.Errorf("expected scrollKeyHandled, got %d", result)
	}
	if scroll.offset != 1 {
		t.Errorf("expected offset 1, got %d", scroll.offset)
	}
}

func TestHandleScrollKeys_Up(t *testing.T) {
	scroll := listScroll{offset: 3}
	msg := tea.KeyPressMsg{Code: 'k'}

	result := HandleScrollKeys(msg, &scroll, 10, 5)
	if result != scrollKeyHandled {
		t.Errorf("expected scrollKeyHandled, got %d", result)
	}
	if scroll.offset != 2 {
		t.Errorf("expected offset 2, got %d", scroll.offset)
	}
}

func TestHandleScrollKeys_Enter(t *testing.T) {
	scroll := listScroll{}
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}

	result := HandleScrollKeys(msg, &scroll, 10, 5)
	if result != scrollKeyPopToRoot {
		t.Errorf("expected scrollKeyPopToRoot, got %d", result)
	}
}

func TestHandleScrollKeys_Unhandled(t *testing.T) {
	scroll := listScroll{}
	msg := tea.KeyPressMsg{Code: 'x'}

	result := HandleScrollKeys(msg, &scroll, 10, 5)
	if result != scrollKeyUnhandled {
		t.Errorf("expected scrollKeyUnhandled, got %d", result)
	}
}

func TestFilterInput_HandleKey_Append(t *testing.T) {
	f := filterInput{active: true}
	msg := tea.KeyPressMsg{Code: 'a', Text: "a"}
	changed := f.HandleKey(msg)
	if !changed || f.text != "a" {
		t.Errorf("expected text='a' changed=true, got text=%q changed=%v", f.text, changed)
	}
}

func TestFilterInput_HandleKey_Backspace(t *testing.T) {
	f := filterInput{active: true, text: "abc"}
	msg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	changed := f.HandleKey(msg)
	if !changed || f.text != "ab" {
		t.Errorf("expected text='ab' changed=true, got text=%q changed=%v", f.text, changed)
	}
}

func TestFilterInput_HandleKey_BackspaceEmpty(t *testing.T) {
	f := filterInput{active: true, text: ""}
	msg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	changed := f.HandleKey(msg)
	if changed {
		t.Error("expected changed=false for backspace on empty")
	}
}

func TestFilterInput_HandleKey_Esc(t *testing.T) {
	f := filterInput{active: true, text: "abc"}
	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	changed := f.HandleKey(msg)
	if !changed || f.text != "" || f.active {
		t.Errorf("expected text='' active=false changed=true, got text=%q active=%v changed=%v", f.text, f.active, changed)
	}
}

func TestFilterInput_HandleKey_Enter(t *testing.T) {
	f := filterInput{active: true, text: "abc"}
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	changed := f.HandleKey(msg)
	if changed || f.active {
		t.Errorf("expected active=false changed=false, got active=%v changed=%v", f.active, changed)
	}
	if f.text != "abc" {
		t.Errorf("expected text preserved, got %q", f.text)
	}
}

func TestFilterInput_MatchesAny(t *testing.T) {
	f := filterInput{text: "foo"}
	if !f.MatchesAny("foobar", "baz") {
		t.Error("expected match on 'foobar'")
	}
	if f.MatchesAny("bar", "baz") {
		t.Error("expected no match")
	}
}

func TestFilterInput_MatchesAny_CaseInsensitive(t *testing.T) {
	f := filterInput{text: "FOO"}
	if !f.MatchesAny("foobar") {
		t.Error("expected case-insensitive match")
	}
}

func TestFilterInput_MatchesAny_EmptyFilter(t *testing.T) {
	f := filterInput{text: ""}
	if !f.MatchesAny("anything") {
		t.Error("expected match when filter is empty")
	}
}

func TestFilterInput_Render_Inactive(t *testing.T) {
	styles := testStyles()
	f := filterInput{}
	if got := f.Render(styles); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestFilterInput_Render_Active(t *testing.T) {
	styles := testStyles()
	f := filterInput{active: true, text: "abc"}
	got := f.Render(styles)
	if !strings.Contains(got, "abc") || !strings.Contains(got, "█") {
		t.Errorf("expected active cursor, got %q", got)
	}
}

func TestFilterInput_Render_InactiveWithText(t *testing.T) {
	styles := testStyles()
	f := filterInput{active: false, text: "abc"}
	got := f.Render(styles)
	if !strings.Contains(got, "abc") {
		t.Errorf("expected text shown, got %q", got)
	}
	if strings.Contains(got, "█") {
		t.Errorf("expected no cursor when inactive, got %q", got)
	}
}
