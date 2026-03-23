package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// testHeader returns a Header with known values for deterministic testing.
func testHeader() Header {
	return Header{
		Profile:  "go-dev",
		Scope:    "global",
		Agent:    "claude",
		Deployed: 12,
		Issues:   0,
		DryRun:   false,
	}
}

// testStyles returns unstyled Styles (no ANSI escapes) for deterministic
// width calculations in tests.
func testStyles() Styles {
	return Styles{
		Subtle:  lipgloss.NewStyle(),
		Primary: lipgloss.NewStyle(),
		Success: lipgloss.NewStyle(),
		Warning: lipgloss.NewStyle(),
		Danger:  lipgloss.NewStyle(),
		Bold:    lipgloss.NewStyle(),
	}
}

func TestHeaderViewZeroIssues(t *testing.T) {
	h := testHeader()
	s := testStyles()
	width := 80

	got := h.View(s, width)

	// Left side must contain profile, scope, agent separated by dot glyph.
	if !strings.Contains(got, "go-dev") {
		t.Error("expected profile 'go-dev' in output")
	}
	if !strings.Contains(got, "global") {
		t.Error("expected scope 'global' in output")
	}
	if !strings.Contains(got, "claude") {
		t.Error("expected agent 'claude' in output")
	}
	if !strings.Contains(got, GlyphDot) {
		t.Errorf("expected dot glyph %q in output", GlyphDot)
	}

	// Right side must contain counts.
	if !strings.Contains(got, "12 deployed") {
		t.Error("expected '12 deployed' in output")
	}
	if !strings.Contains(got, "0 issues") {
		t.Error("expected '0 issues' in output")
	}

	// Should NOT contain DRY RUN.
	if strings.Contains(got, "[DRY RUN]") {
		t.Error("DRY RUN should not appear when DryRun is false")
	}
}

func TestHeaderViewWithIssues(t *testing.T) {
	h := testHeader()
	h.Issues = 3
	s := testStyles()
	width := 80

	got := h.View(s, width)

	// Both deployed count and issues count must appear.
	if !strings.Contains(got, "12 deployed") {
		t.Error("expected '12 deployed' in output")
	}
	if !strings.Contains(got, "3 issues") {
		t.Error("expected '3 issues' in output")
	}
}

func TestHeaderViewDryRun(t *testing.T) {
	h := testHeader()
	h.DryRun = true
	s := testStyles()
	width := 80

	got := h.View(s, width)

	if !strings.Contains(got, "[DRY RUN]") {
		t.Error("expected '[DRY RUN]' prefix when DryRun is true")
	}

	// The rest of the header elements should still be present.
	if !strings.Contains(got, "go-dev") {
		t.Error("expected profile 'go-dev' in output")
	}
	if !strings.Contains(got, "global") {
		t.Error("expected scope 'global' in output")
	}
	if !strings.Contains(got, "claude") {
		t.Error("expected agent 'claude' in output")
	}
}

func TestHeaderViewWidthAdaptation(t *testing.T) {
	h := testHeader()
	s := testStyles()

	// With unstyled output (no ANSI), we can verify exact widths.
	// Left: "  go-dev · global · claude"
	// Right: "12 deployed  0 issues"
	left := "  go-dev · global · claude"
	right := "12 deployed  0 issues"
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)

	width := 100
	got := h.View(s, width)
	gotW := lipgloss.Width(got)

	// Total rendered width should equal the requested width.
	if gotW != width {
		t.Errorf("expected rendered width %d, got %d", width, gotW)
	}

	// The gap should be width - left - right.
	expectedGap := width - leftW - rightW
	if expectedGap < 1 {
		t.Fatal("test setup error: width too narrow for gap test")
	}

	// Verify gap is filled with spaces.
	gapStr := strings.Repeat(" ", expectedGap)
	if !strings.Contains(got, gapStr) {
		t.Errorf("expected gap of %d spaces between left and right", expectedGap)
	}
}

func TestHeaderViewNarrowWidth(t *testing.T) {
	h := testHeader()
	s := testStyles()

	// Width too small to fit both sides — gap should be clamped to 1.
	got := h.View(s, 10)

	// Should still contain left and right content with at least 1 space gap.
	if !strings.Contains(got, "go-dev") {
		t.Error("expected profile in narrow output")
	}
	if !strings.Contains(got, "deployed") {
		t.Error("expected deployed count in narrow output")
	}

	// Minimum gap is 1, so total width should be left + 1 + right.
	left := "  go-dev · global · claude"
	right := "12 deployed  0 issues"
	expectedMinWidth := lipgloss.Width(left) + 1 + lipgloss.Width(right)
	gotW := lipgloss.Width(got)
	if gotW != expectedMinWidth {
		t.Errorf("expected minimum width %d when narrow, got %d", expectedMinWidth, gotW)
	}
}

func TestHeaderViewZeroWidth(t *testing.T) {
	h := testHeader()
	s := testStyles()

	// Width 0 should not panic — gap clamps to 1.
	got := h.View(s, 0)
	if got == "" {
		t.Error("expected non-empty output even with zero width")
	}
}

func TestHeaderViewEmptyFields(t *testing.T) {
	h := Header{}
	s := testStyles()

	got := h.View(s, 80)

	// Should render without panicking.
	if got == "" {
		t.Error("expected non-empty output for zero-value header")
	}
	// Zero-value counts.
	if !strings.Contains(got, "0 deployed") {
		t.Error("expected '0 deployed' for zero-value header")
	}
	if !strings.Contains(got, "0 issues") {
		t.Error("expected '0 issues' for zero-value header")
	}
}
