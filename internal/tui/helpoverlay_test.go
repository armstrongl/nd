package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// helpTestSectionedScreen implements SectionedHelpProvider for overlay tests.
// It embeds helpTestScreen (defined in helpbar_test.go) for the Screen methods.
type helpTestSectionedScreen struct{ helpTestScreen }

func (helpTestSectionedScreen) HelpSections() []HelpSection {
	return []HelpSection{
		{Title: "Navigation", Items: []HelpItem{{"j/k", "move"}, {"esc", "back"}}},
		{Title: "Actions", Items: []HelpItem{{"d", "deploy"}}},
	}
}

// helpTestLongScreen exposes a section with a line wider than a narrow terminal
// to exercise truncation.
type helpTestLongScreen struct{ helpTestScreen }

func (helpTestLongScreen) HelpSections() []HelpSection {
	return []HelpSection{
		{Title: "Tips", Items: []HelpItem{
			{"", "This is a very long help description that far exceeds twenty columns wide"},
			{"backspace", "delete the character to the left of the cursor position now"},
		}},
	}
}

func TestHelpOverlay_FlatItemsForBasicScreen(t *testing.T) {
	ov := HelpOverlay{}
	s := testStyles()

	out := ov.View(s, helpTestScreen{}, 80, 24)

	// Title is derived from the screen.
	if !strings.Contains(out, "test") {
		t.Errorf("overlay missing screen title; got:\n%s", out)
	}
	// Flat defaultHelp items should be listed.
	for _, want := range []string{"esc", "back", "j/k", "navigate", "enter", "select", "?", "help", "q", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("overlay missing default help token %q; got:\n%s", want, out)
		}
	}
	// Close hint present.
	if !strings.Contains(out, "close") {
		t.Errorf("overlay missing close hint; got:\n%s", out)
	}
}

func TestHelpOverlay_FlatItemsIncludeProviderItems(t *testing.T) {
	ov := HelpOverlay{}
	s := testStyles()

	out := ov.View(s, helpTestScreenWithItems{}, 80, 24)

	for _, want := range []string{"f", "fix", "d", "deploy"} {
		if !strings.Contains(out, want) {
			t.Errorf("overlay missing HelpProvider item %q; got:\n%s", want, out)
		}
	}
}

func TestHelpOverlay_SectionHeadings(t *testing.T) {
	ov := HelpOverlay{}
	s := testStyles()

	out := ov.View(s, helpTestSectionedScreen{}, 80, 24)

	for _, heading := range []string{"Navigation", "Actions"} {
		if !strings.Contains(out, heading) {
			t.Errorf("overlay missing section heading %q; got:\n%s", heading, out)
		}
	}
	for _, item := range []string{"j/k", "move", "d", "deploy"} {
		if !strings.Contains(out, item) {
			t.Errorf("overlay missing section item %q; got:\n%s", item, out)
		}
	}
	// "Navigation" heading must appear before "Actions".
	if strings.Index(out, "Navigation") >= strings.Index(out, "Actions") {
		t.Errorf("expected Navigation before Actions; got:\n%s", out)
	}
}

func TestHelpOverlay_NarrowWidthTruncates(t *testing.T) {
	ov := HelpOverlay{}
	s := testStyles()
	const width = 20

	out := ov.View(s, helpTestLongScreen{}, width, 24)

	for _, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("line exceeds width %d (got %d): %q", width, w, line)
		}
	}
	// The overlay should still render something meaningful.
	if !strings.Contains(out, "Tips") {
		t.Errorf("overlay missing Tips heading; got:\n%s", out)
	}
}

func TestHelpOverlay_QuestionMarkOpensAndClosesViaModel(t *testing.T) {
	question := tea.KeyPressMsg(tea.Key{Code: '?'})
	esc := tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})

	// '?' opens the overlay.
	m := newTestModelWithScreen(helpTestScreen{})
	updated, _ := m.Update(question)
	m = updated.(Model)
	if !m.helpOpen {
		t.Fatal("expected helpOpen=true after pressing '?'")
	}

	// '?' again closes it.
	updated, _ = m.Update(question)
	m = updated.(Model)
	if m.helpOpen {
		t.Fatal("expected helpOpen=false after pressing '?' again")
	}

	// esc closes the overlay without popping the (single) nav stack or quitting.
	updated, _ = m.Update(question)
	m = updated.(Model)
	if !m.helpOpen {
		t.Fatal("expected helpOpen=true after reopening with '?'")
	}
	updated, cmd := m.Update(esc)
	m = updated.(Model)
	if m.helpOpen {
		t.Fatal("expected helpOpen=false after pressing esc")
	}
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("esc while overlay open must not quit")
		}
	}
	if len(m.screens) != 1 {
		t.Fatalf("esc while overlay open must not pop nav stack; got %d screens", len(m.screens))
	}

	// The overlay must be reflected in the rendered view while open.
	updated, _ = m.Update(question)
	m = updated.(Model)
	if !strings.Contains(m.View().Content, "close") {
		t.Errorf("expected view to contain overlay close hint while open; got:\n%s", m.View().Content)
	}
}

func TestHelpOverlay_QuestionMarkSuppressedWhenInputActive(t *testing.T) {
	// A screen with focused text input must not open the overlay on '?'.
	m := newTestModelWithScreen(stubScreen{title: "Input", inputActive: true})
	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: '?', Text: "?"}))
	m = updated.(Model)
	if m.helpOpen {
		t.Fatal("'?' must not open the overlay while text input is active")
	}
}
