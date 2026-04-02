package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/armstrongl/nd/internal/asset"
)

// browseLoadedMsg carries the results of the async asset load.
type browseLoadedMsg struct {
	assets   []*asset.Asset
	deployed map[string]bool // identity.String() -> true
	err      error
}

// browseScreen shows all available assets with deployment status markers.
// It supports a text filter toggled with '/' and cursor navigation with j/k.
// Long lists are windowed to the terminal height; ↑/↓ indicators appear when
// items are hidden above or below the viewport.
type browseScreen struct {
	svc    Services
	styles Styles
	isDark bool

	assets    []*asset.Asset
	deployed  map[string]bool
	cursor    int
	filter    string
	filtering bool
	notice    string // transient feedback (e.g. "already deployed")
	err       error
	loaded    bool

	height int       // terminal height, updated by tea.WindowSizeMsg
	scroll listScroll
}

func newBrowseScreen(svc Services, styles Styles, isDark bool) *browseScreen {
	return &browseScreen{svc: svc, styles: styles, isDark: isDark}
}

func (b *browseScreen) Title() string { return "Browse" }

// InputActive returns true while the filter input is active,
// suppressing global q/esc key handling.
func (b *browseScreen) InputActive() bool { return b.filtering }

// FullHelpItems returns step-specific help items for the browse screen.
func (b *browseScreen) FullHelpItems() []HelpItem {
	if b.filtering {
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
	return func() tea.Msg {
		summary, err := svc.ScanIndex()
		if err != nil {
			return browseLoadedMsg{err: err}
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

		return browseLoadedMsg{assets: assets, deployed: deployed}
	}
}

// Update handles messages and filter key input.
func (b *browseScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case browseLoadedMsg:
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
	if b.filtering {
		switch msg.String() {
		case "esc":
			b.filter = ""
			b.filtering = false
		case "enter":
			b.filtering = false
		case "backspace":
			if len(b.filter) > 0 {
				b.filter = b.filter[:len(b.filter)-1]
			}
		default:
			// Append printable characters to filter.
			if msg.Text != "" {
				b.filter += msg.Text
			}
		}
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
		b.filtering = true
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
	if b.filtering || b.filter != "" {
		indicator := " "
		if b.filtering {
			indicator = "_"
		}
		fmt.Fprintf(&buf, "  %s %s%s\n\n",
			b.styles.Subtle.Render("/"),
			b.filter,
			indicator)
	}

	if len(visible) == 0 {
		fmt.Fprintf(&buf, "  %s",
			b.styles.Subtle.Render("No assets match the filter."))
		return tea.NewView(buf.String())
	}

	pageSize := b.contentHeight()
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
	if b.filter != "" {
		fmt.Fprintf(&buf, "\n  %s",
			b.styles.Subtle.Render(fmt.Sprintf("%d of %d  · / to filter", shown, total)))
	} else {
		fmt.Fprintf(&buf, "\n  %s",
			b.styles.Subtle.Render(fmt.Sprintf("%d assets  · / to filter", total)))
	}

	return tea.NewView(buf.String())
}

// contentHeight returns the number of list rows that fit in the terminal.
// When height is unknown (0), it returns listScrollUnlimited so all assets
// are visible without windowing (matching pre-scroll behaviour in tests).
func (b *browseScreen) contentHeight() int {
	if b.height == 0 {
		return listScrollUnlimited
	}
	h := b.height
	h -= 4 // root chrome: header + 2 blank separators + helpbar
	h -= 2 // summary footer: blank line + summary line
	if b.filtering || b.filter != "" {
		h -= 2 // filter bar + blank line below it
	}
	if b.notice != "" {
		h -= 2 // transient notice + blank line before it
	}
	if h < 3 {
		h = 3
	}
	return h
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
	if b.filter == "" {
		return b.assets
	}
	lower := strings.ToLower(b.filter)
	var out []*asset.Asset
	for _, a := range b.assets {
		if strings.Contains(strings.ToLower(a.Name), lower) ||
			strings.Contains(strings.ToLower(string(a.Type)), lower) ||
			strings.Contains(strings.ToLower(a.SourceID), lower) {
			out = append(out, a)
		}
	}
	return out
}
