package tui

import (
	"image/color"
	"testing"
)

func TestNewStylesDark(t *testing.T) {
	s := NewStyles(true)
	assertStyleHasForeground(t, "Subtle", s.Subtle)
	assertStyleHasForeground(t, "Primary", s.Primary)
	assertStyleHasForeground(t, "Success", s.Success)
	assertStyleHasForeground(t, "Warning", s.Warning)
	assertStyleHasForeground(t, "Danger", s.Danger)
}

func TestNewStylesLight(t *testing.T) {
	s := NewStyles(false)
	assertStyleHasForeground(t, "Subtle", s.Subtle)
	assertStyleHasForeground(t, "Primary", s.Primary)
	assertStyleHasForeground(t, "Success", s.Success)
	assertStyleHasForeground(t, "Warning", s.Warning)
	assertStyleHasForeground(t, "Danger", s.Danger)
}

func TestNewStylesBold(t *testing.T) {
	s := NewStyles(true)
	if !s.Bold.GetBold() {
		t.Error("Bold style should have bold attribute set")
	}
}

func TestNewStylesDarkLightDiffer(t *testing.T) {
	dark := NewStyles(true)
	light := NewStyles(false)

	darkFg := dark.Primary.GetForeground()
	lightFg := light.Primary.GetForeground()

	if colorsEqual(darkFg, lightFg) {
		t.Error("Primary foreground should differ between dark and light modes")
	}
}

func TestAllFieldsPopulated(t *testing.T) {
	s := NewStyles(true)

	// Verify all 6 fields are populated (non-zero foreground or bold)
	fields := map[string]bool{
		"Subtle":  s.Subtle.GetForeground() != nil,
		"Primary": s.Primary.GetForeground() != nil,
		"Success": s.Success.GetForeground() != nil,
		"Warning": s.Warning.GetForeground() != nil,
		"Danger":  s.Danger.GetForeground() != nil,
		"Bold":    s.Bold.GetBold(),
	}

	for name, populated := range fields {
		if !populated {
			t.Errorf("Styles.%s is not populated", name)
		}
	}
}

func TestGlyphsNonEmpty(t *testing.T) {
	glyphs := map[string]string{
		"GlyphOK":      GlyphOK,
		"GlyphBroken":  GlyphBroken,
		"GlyphDrifted": GlyphDrifted,
		"GlyphOrphan":  GlyphOrphan,
		"GlyphMissing": GlyphMissing,
		"GlyphDot":     GlyphDot,
		"GlyphArrow":   GlyphArrow,
	}

	for name, val := range glyphs {
		if val == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestNdThemeDark(t *testing.T) {
	theme := NdTheme(true)
	if theme == nil {
		t.Fatal("NdTheme(true) returned nil")
	}
}

func TestNdThemeLight(t *testing.T) {
	theme := NdTheme(false)
	if theme == nil {
		t.Fatal("NdTheme(false) returned nil")
	}
}

// assertStyleHasForeground checks that a style has a non-nil, non-zero foreground color.
func assertStyleHasForeground(t *testing.T, name string, s interface{ GetForeground() color.Color }) {
	t.Helper()
	fg := s.GetForeground()
	if fg == nil {
		t.Errorf("%s style has nil foreground", name)
		return
	}
	r, g, b, _ := fg.RGBA()
	if r == 0 && g == 0 && b == 0 {
		t.Errorf("%s style has zero foreground color (black); expected a non-zero color", name)
	}
}

// colorsEqual returns true if two colors have the same RGBA values.
func colorsEqual(a, b color.Color) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
