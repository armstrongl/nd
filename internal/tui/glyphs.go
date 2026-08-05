package tui

// Status glyphs and badges shared across every TUI screen.
//
// Centralising them here — instead of scattering string literals like "*",
// "!", "[DRY RUN]" and "(active)" across screens — guarantees the same concept
// renders identically everywhere. Following the convention established for the
// health glyphs, these stay readable without colour: the character itself, not
// just a lipgloss colour, communicates the state.
const (
	// Health / status glyphs (single home; previously lived in theme.go).
	GlyphOK      = "ok"
	GlyphBroken  = "!!"
	GlyphDrifted = "??"
	GlyphOrphan  = "--"
	GlyphMissing = "xx"
	GlyphDot     = "·" // middle dot ·
	GlyphArrow   = "->"

	// GlyphDeployed marks an already-deployed asset in the browse list.
	GlyphDeployed = "✓" // checkmark ✓
	// GlyphNew marks an asset whose source file was modified within the recency window.
	GlyphNew = "new"
	// GlyphActive marks the active profile in the profile list and switch picker.
	GlyphActive = "*"
	// GlyphPinned marks a pinned deployment in the pin picker.
	GlyphPinned = "[pinned]"
	// GlyphScrollUp / GlyphScrollDown are the "more above" / "more below" hints.
	GlyphScrollUp   = "↑" // ↑
	GlyphScrollDown = "↓" // ↓
	// GlyphDryRun prefixes headers and previews while dry-run mode is active.
	GlyphDryRun = "[DRY RUN]"
	// GlyphWarning flags a non-fatal warning (e.g. doctor output).
	GlyphWarning = "!"
)

// Deployed returns the deployed marker rendered in the success colour.
func (s Styles) Deployed() string { return s.Success.Render(GlyphDeployed) }

// NewBadge returns the "new" badge rendered in the primary colour.
func (s Styles) NewBadge() string { return s.Primary.Render(GlyphNew) }

// Active returns the active-profile marker rendered in the primary colour.
func (s Styles) Active() string { return s.Primary.Render(GlyphActive) }

// Pinned returns the pinned badge rendered in the primary colour.
func (s Styles) Pinned() string { return s.Primary.Render(GlyphPinned) }
