package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// ColorPair holds light and dark terminal color variants.
type ColorPair struct {
	Light color.Color
	Dark  color.Color
}

// Resolve picks the appropriate color using the app's light/dark function.
func (cp ColorPair) Resolve(ld lipgloss.LightDarkFunc) color.Color {
	return ld(cp.Light, cp.Dark)
}

// Color pairs for adaptive theming. Resolved at render time via LightDarkFunc
// (set from tea.BackgroundColorMsg).
var (
	PairOK      = ColorPair{Light: lipgloss.Color("#22863a"), Dark: lipgloss.Color("#85e89d")}
	PairBroken  = ColorPair{Light: lipgloss.Color("#cb2431"), Dark: lipgloss.Color("#f97583")}
	PairDrifted = ColorPair{Light: lipgloss.Color("#b08800"), Dark: lipgloss.Color("#ffea7f")}
	PairPinned  = ColorPair{Light: lipgloss.Color("#0366d6"), Dark: lipgloss.Color("#79b8ff")}
	PairProfile = ColorPair{Light: lipgloss.Color("#1b7c83"), Dark: lipgloss.Color("#56d4dd")}
	PairDim     = ColorPair{Light: lipgloss.Color("#6a737d"), Dark: lipgloss.Color("#959da5")}
	PairAccent  = ColorPair{Light: lipgloss.Color("#6f42c1"), Dark: lipgloss.Color("#b392f0")}
)

// Styles that don't depend on terminal background color.
var (
	StyleHeader        = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	StyleTabActive     = lipgloss.NewStyle().Bold(true).Underline(true)
	StyleTableSelected = lipgloss.NewStyle().Reverse(true)
	StyleDetailBox     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).MarginLeft(2)
	StyleModal         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
	StyleHelpKey       = lipgloss.NewStyle().Bold(true)
	StyleToast         = lipgloss.NewStyle().Reverse(true).Padding(0, 1)
)

// Styles builds color-resolved styles from a LightDarkFunc.
// Call this after receiving tea.BackgroundColorMsg to get themed styles.
type Styles struct {
	TabInactive     lipgloss.Style
	IssueBadgeError lipgloss.Style
	IssueBadgeWarn  lipgloss.Style
	StatusOK        lipgloss.Style
	StatusBroken    lipgloss.Style
	StatusDrifted   lipgloss.Style
	OriginPinned    lipgloss.Style
	OriginProfile   lipgloss.Style
	HelpBar         lipgloss.Style
	Empty           lipgloss.Style
	Loading         lipgloss.Style
}

// NewStyles creates themed styles using the given LightDarkFunc.
func NewStyles(ld lipgloss.LightDarkFunc) Styles {
	return Styles{
		TabInactive:     lipgloss.NewStyle().Foreground(PairDim.Resolve(ld)),
		IssueBadgeError: lipgloss.NewStyle().Foreground(PairBroken.Resolve(ld)).Bold(true),
		IssueBadgeWarn:  lipgloss.NewStyle().Foreground(PairDrifted.Resolve(ld)).Bold(true),
		StatusOK:        lipgloss.NewStyle().Foreground(PairOK.Resolve(ld)),
		StatusBroken:    lipgloss.NewStyle().Foreground(PairBroken.Resolve(ld)),
		StatusDrifted:   lipgloss.NewStyle().Foreground(PairDrifted.Resolve(ld)),
		OriginPinned:    lipgloss.NewStyle().Foreground(PairPinned.Resolve(ld)),
		OriginProfile:   lipgloss.NewStyle().Foreground(PairProfile.Resolve(ld)),
		HelpBar:         lipgloss.NewStyle().Foreground(PairDim.Resolve(ld)),
		Empty:           lipgloss.NewStyle().Foreground(PairDim.Resolve(ld)).Align(lipgloss.Center),
		Loading:         lipgloss.NewStyle().Foreground(PairDim.Resolve(ld)),
	}
}

// DefaultStyles returns styles assuming a dark terminal background.
func DefaultStyles() Styles {
	return NewStyles(lipgloss.LightDark(true))
}
