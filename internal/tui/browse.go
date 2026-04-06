package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/asset"
)

// browseLoadedMsg carries the results of the async asset load.
type browseLoadedMsg struct {
	assets     []*asset.Asset
	deployed   map[string]bool // identity.String() -> true
	err        error
	generation uint64
}

// browseScreen shows all available assets with deployment status markers.
// It supports a text filter toggled with '/' and cursor navigation with j/k.
// Long lists are windowed to the terminal height; ↑/↓ indicators appear when
// items are hidden above or below the viewport.
type browseScreen struct {
	svc    Services
	styles Styles
	isDark bool

	assets   []*asset.Asset
	deployed map[string]bool
	cursor   int
	filter   filterInput
	notice   string // transient feedback (e.g. "already deployed")
	err       error
	loaded    bool

	// generation is incremented on every ScopeSwitchedMsg so that stale
	// browseLoadedMsg results from a previous scope's goroutine are discarded.
	generation uint64

	height int       // terminal height, updated by tea.WindowSizeMsg
	scroll listScroll
}

func newBrowseScreen(svc Services, styles Styles, isDark bool) *browseScreen {
	return &browseScreen{svc: svc, styles: styles, isDark: isDark}
}

func (b *browseScreen) Title() string { return "Browse" }

// InputActive returns true while the filter input is active,
// suppressing global q/esc key handling.
func (b *browseScreen) InputActive() bool { return b.filter.active }

// FullHelpItems returns step-specific help items for the browse screen.
func (b *browseScreen) FullHelpItems() []HelpItem {
	if b.filter.active {
		return []HelpItem{
			{"esc", "cancel"},
			{"enter", "apply"},
			{"backspace", "delete"},
		}
	}
	return []HelpItem{
		{"esc", "back"},
		{"j/k", "navigate"},
		{"enter", "deploy"},
		{"/", "filter"},
		{"q", "quit"},
	}
}

// Init starts async loading of all assets and deployed state.
func (b *browseScreen) Init() tea.Cmd {
	svc := b.svc
	gen := b.generation
	return func() tea.Msg {
		summary, err := svc.ScanIndex()
		if err != nil {
			return browseLoadedMsg{err: err, generation: gen}
		}

		agentAlias := ""
		if ag, err := svc.DefaultAgent(); err == nil {
			agentAlias = ag.SourceAlias
		}

		var assets []*asset.Asset
		if summary != nil && summary.Index != nil {
			assets = summary.Index.FilterByAgent(agentAlias)
		}

		// Build deployed set by cross-referencing the state store.
		deployed := make(map[string]bool)
		if store := svc.StateStore(); store != nil {
			if st, _, err := store.Load(); err == nil {
				for _, dep := range st.Deployments {
					deployed[dep.Identity().String()] = true
				}
			}
		}

		return browseLoadedMsg{assets: assets, deployed: deployed, generation: gen}
	}
}

// Update handles messages and filter key input.
func (b *browseScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ScopeSwitchedMsg:
		b.generation++
		b.loaded = false
		b.assets = nil
		b.deployed = nil
		b.filter = filterInput{}
		b.cursor = 0
		b.scroll = listScroll{}
		return b, b.Init()

	case browseLoadedMsg:
		if msg.generation != b.generation {
			return b, nil // stale result from a previous scope; discard
		}
		b.loaded = true
		b.assets = msg.assets
		b.deployed = msg.deployed
		b.err = msg.err
		b.clampCursor()
		return b, nil

	case tea.WindowSizeMsg:
		b.height = msg.Height
		b.scroll.EnsureVisible(b.cursor, b.contentHeight())
		return b, nil

	case tea.KeyPressMsg:
		return b.handleKey(msg)
	}

	return b, nil
}

// handleKey processes keyboard input, routing filter keys vs navigation.
func (b *browseScreen) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if b.filter.active {
		b.filter.HandleKey(msg)
		b.clampCursor()
		return b, nil
	}

	// Not filtering: guard against pre-load key handling.
	if !b.loaded {
		return b, nil
	}

	b.notice = ""
	visible := b.visibleAssets()

	switch msg.String() {
	case "/":
		b.filter.active = true
		return b, nil

	case "j", "down":
		if b.cursor < len(visible)-1 {
			b.cursor++
			b.scroll.EnsureVisible(b.cursor, b.contentHeight())
		}
		return b, nil

	case "k", "up":
		if b.cursor > 0 {
			b.cursor--
			b.scroll.EnsureVisible(b.cursor, b.contentHeight())
		}
		return b, nil

	case "enter":
		if len(visible) == 0 {
			return b, nil
		}
		a := visible[b.cursor]
		if b.deployed[a.String()] {
			b.notice = fmt.Sprintf("%s is already deployed", a.Name)
			return b, nil
		}
		screen := newDeployScreen(b.svc, b.styles, b.isDark)
		return b, func() tea.Msg { return NavigateMsg{Screen: screen} }
	}

	return b, nil
}

// View renders the asset list with optional filter.
func (b *browseScreen) View() tea.View {
	if !b.loaded {
		return tea.NewView("  Loading assets...")
	}
	if b.err != nil {
		return tea.NewView(fmt.Sprintf("  %s\n\n  %s\n\n  %s",
			b.styles.Danger.Render("Error"),
			b.err.Error(),
			b.styles.Subtle.Render("Press esc to go back.")))
	}

	visible := b.visibleAssets()
	if len(visible) == 0 && len(b.assets) == 0 {
		return tea.NewView("  " + NoAssets())
	}

	var buf strings.Builder

	// Filter bar when active.
	if filterBar := b.filter.Render(b.styles); filterBar != "" {
		buf.WriteString(filterBar)
		buf.WriteString("\n")
	}

	if len(visible) == 0 {
		fmt.Fprintf(&buf, "  %s",
			b.styles.Subtle.Render("No assets match the filter."))
		return tea.NewView(buf.String())
	}

	pageSize := b.contentHeight()
	// Reserve rows for scroll indicators so they don't push content past the
	// terminal height budget.
	if b.scroll.MoreAbove() > 0 {
		pageSize--
	}
	if b.scroll.MoreBelow(len(visible), pageSize) > 0 {
		pageSize--
	}
	if pageSize < 1 {
		pageSize = 1
	}
	start, end := b.scroll.Window(len(visible), pageSize)

	if above := b.scroll.MoreAbove(); above > 0 {
		fmt.Fprintf(&buf, "%s\n", scrollIndicatorLine(b.styles, "↑", above))
	}

	for i, a := range visible[start:end] {
		absIdx := start + i
		cursor := "  "
		if absIdx == b.cursor {
			cursor = GlyphArrow + " "
		}

		marker := " "
		if b.deployed[a.String()] {
			marker = "*"
		}

		typePart := b.styles.Subtle.Render(string(a.Type))
		srcPart := b.styles.Subtle.Render(a.SourceID)

		description := ""
		if a.Meta != nil && a.Meta.Description != "" {
			description = b.styles.Subtle.Render("  " + a.Meta.Description)
		}

		fmt.Fprintf(&buf, "%s%s  %-12s  %-24s  %s%s\n",
			cursor, marker, typePart, a.Name, srcPart, description)
	}

	if below := b.scroll.MoreBelow(len(visible), pageSize); below > 0 {
		fmt.Fprintf(&buf, "%s\n", scrollIndicatorLine(b.styles, "↓", below))
	}

	// Transient feedback message (e.g. "already deployed").
	if b.notice != "" {
		fmt.Fprintf(&buf, "\n  %s", b.styles.Warning.Render(b.notice))
	}

	total := len(b.assets)
	shown := len(visible)
	if b.filter.text != "" {
		fmt.Fprintf(&buf, "\n  %s",
			b.styles.Subtle.Render(fmt.Sprintf("%d of %d  · / to filter", shown, total)))
	} else {
		fmt.Fprintf(&buf, "\n  %s",
			b.styles.Subtle.Render(fmt.Sprintf("%d assets  · / to filter", total)))
	}

	return tea.NewView(buf.String())
}

// contentHeight returns the number of list rows that fit in the terminal.
// Browse has extra chrome beyond the standard 4 lines (summary footer,
// optional filter bar, optional notice).
func (b *browseScreen) contentHeight() int {
	extra := 4 + 2 // root chrome + summary footer
	if b.filter.active || b.filter.text != "" {
		extra += 2 // filter bar + blank line below it
	}
	if b.notice != "" {
		extra += 2 // transient notice + blank line before it
	}
	return ContentHeight(b.height, extra)
}

// clampCursor ensures the cursor stays within the visible asset list bounds
// and calls EnsureVisible so the scroll offset tracks it.
func (b *browseScreen) clampCursor() {
	visible := b.visibleAssets()
	if len(visible) == 0 {
		b.cursor = 0
		return
	}
	if b.cursor >= len(visible) {
		b.cursor = len(visible) - 1
	}
	b.scroll.EnsureVisible(b.cursor, b.contentHeight())
}

// visibleAssets returns the filtered asset list (or all if no filter set).
func (b *browseScreen) visibleAssets() []*asset.Asset {
	if b.filter.text == "" {
		return b.assets
	}
	var out []*asset.Asset
	for _, a := range b.assets {
		if b.filter.MatchesAny(a.Name, string(a.Type), a.SourceID) {
			out = append(out, a)
		}
	}
	return out
}
