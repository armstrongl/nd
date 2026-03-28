package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/list"

	"github.com/armstrongl/nd/internal/asset"
)

// browseLoadedMsg carries the results of the async asset load.
type browseLoadedMsg struct {
	assets   []*asset.Asset
	deployed map[string]bool // identity.String() -> true
	err      error
}

// assetItem implements list.DefaultItem for use in the bubbles/list component.
type assetItem struct {
	name     string
	desc     string // "type · source"
	filterV  string // name + type + source for fuzzy matching
	deployed bool
}

func (i assetItem) Title() string       { return i.title() }
func (i assetItem) Description() string { return i.desc }
func (i assetItem) FilterValue() string { return i.filterV }

func (i assetItem) title() string {
	marker := " "
	if i.deployed {
		marker = "*"
	}
	return fmt.Sprintf("%s %s", marker, i.name)
}

// browseScreen shows all available assets with deployment status markers.
// It uses bubbles/list for cursor navigation, pagination, and built-in fuzzy filtering.
type browseScreen struct {
	svc    Services
	styles Styles
	isDark bool

	list    *list.Model
	err     error
	loaded  bool
	noItems bool // true when loaded with zero assets

	// Two-phase init: store dimensions until list is created.
	pendingWidth  int
	pendingHeight int
}

func newBrowseScreen(svc Services, styles Styles, isDark bool) *browseScreen {
	return &browseScreen{svc: svc, styles: styles, isDark: isDark}
}

func (b *browseScreen) Title() string { return "Browse" }

// InputActive returns true while the list's filter is active,
// suppressing global q/esc key handling.
func (b *browseScreen) InputActive() bool {
	if b.list == nil {
		return false
	}
	return b.list.FilterState() != list.Unfiltered
}

// HelpItems returns context-sensitive help items for the browse screen.
func (b *browseScreen) HelpItems() []HelpItem {
	return []HelpItem{
		{"/", "filter"},
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

		var assets []*asset.Asset
		if summary != nil && summary.Index != nil {
			assets = summary.Index.All()
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

// Update handles messages, delegating most input to the bubbles/list component.
func (b *browseScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case browseLoadedMsg:
		b.loaded = true
		b.err = msg.err
		if b.err != nil {
			return b, nil
		}

		if len(msg.assets) == 0 {
			b.noItems = true
			return b, nil
		}

		// Convert assets to list items.
		items := make([]list.Item, len(msg.assets))
		for i, a := range msg.assets {
			items[i] = assetItem{
				name:     a.Name,
				desc:     fmt.Sprintf("%s · %s", a.Type, a.SourceID),
				filterV:  fmt.Sprintf("%s %s %s", a.Name, a.Type, a.SourceID),
				deployed: msg.deployed[a.String()],
			}
		}

		// Create the list with pending dimensions (or zero if none received yet).
		delegate := list.NewDefaultDelegate()
		delegate.Styles = list.NewDefaultItemStyles(b.isDark)
		l := list.New(items, delegate, b.pendingWidth, b.pendingHeight)
		l.DisableQuitKeybindings()
		l.SetShowTitle(false)
		l.SetShowHelp(false)
		l.SetShowFilter(true)
		l.SetStatusBarItemName("asset", "assets")
		b.list = &l
		return b, nil

	case ScreenSizeMsg:
		if b.list != nil {
			b.list.SetSize(msg.Width, msg.Height)
		} else {
			b.pendingWidth = msg.Width
			b.pendingHeight = msg.Height
		}
		return b, nil

	default:
		// Delegate all other messages (key presses, filter matches, etc.) to the list.
		if b.list != nil {
			newList, cmd := b.list.Update(msg)
			b.list = &newList
			return b, cmd
		}
	}

	return b, nil
}

// View renders the asset list with cursor highlight and pagination.
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
	if b.noItems {
		return tea.NewView("  " + NoAssets())
	}
	if b.list == nil {
		return tea.NewView("  Loading assets...")
	}

	return tea.NewView(b.list.View())
}
