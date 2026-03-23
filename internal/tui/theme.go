package tui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Catppuccin-based semantic accent colors.
// Dark values are Mocha palette; light values are Latte palette.
var (
	cSubtleDark  = lipgloss.Color("#6c7086")
	cSubtleLight = lipgloss.Color("#9ca0b0")

	cPrimaryDark  = lipgloss.Color("#89b4fa")
	cPrimaryLight = lipgloss.Color("#1e66f5")

	cSuccessDark  = lipgloss.Color("#a6e3a1")
	cSuccessLight = lipgloss.Color("#40a02b")

	cWarningDark  = lipgloss.Color("#f9e2af")
	cWarningLight = lipgloss.Color("#df8e1d")

	cDangerDark  = lipgloss.Color("#f38ba8")
	cDangerLight = lipgloss.Color("#d20f39")
)

// Styles is the complete set of reusable TUI styles.
type Styles struct {
	Subtle  lipgloss.Style
	Primary lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Danger  lipgloss.Style
	Bold    lipgloss.Style
}

// NewStyles builds a Styles set for the given color mode.
// Pass true for dark terminals, false for light.
func NewStyles(isDark bool) Styles {
	ld := lipgloss.LightDark(isDark)
	return Styles{
		Subtle:  lipgloss.NewStyle().Foreground(ld(cSubtleLight, cSubtleDark)),
		Primary: lipgloss.NewStyle().Foreground(ld(cPrimaryLight, cPrimaryDark)),
		Success: lipgloss.NewStyle().Foreground(ld(cSuccessLight, cSuccessDark)),
		Warning: lipgloss.NewStyle().Foreground(ld(cWarningLight, cWarningDark)),
		Danger:  lipgloss.NewStyle().Foreground(ld(cDangerLight, cDangerDark)),
		Bold:    lipgloss.NewStyle().Bold(true),
	}
}

// Glyphs — text-based status indicators, readable without color.
const (
	GlyphOK      = "ok"
	GlyphBroken  = "!!"
	GlyphDrifted = "??"
	GlyphOrphan  = "--"
	GlyphMissing = "xx"
	GlyphDot     = "\u00b7" // middle dot ·
	GlyphArrow   = "->"
)

// NdTheme returns huh form styles using the Catppuccin color scheme.
func NdTheme(isDark bool) *huh.Styles {
	return huh.ThemeCatppuccin(isDark)
}
