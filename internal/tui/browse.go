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
// It supports a text filter toggled with '/'.
type browseScreen struct {
	svc    Services
	styles Styles
	isDark bool

	assets   []*asset.Asset
	deployed map[string]bool
	filter   string
	filtering bool
	err      error
	loaded   bool
}

func newBrowseScreen(svc Services, styles Styles, isDark bool) *browseScreen {
	return &browseScreen{svc: svc, styles: styles, isDark: isDark}
}

func (b *browseScreen) Title() string { return "Browse" }

// InputActive returns true while the filter input is active,
// suppressing global q/esc key handling.
func (b *browseScreen) InputActive() bool { return b.filtering }

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
		return b, nil
	}

	// Not filtering: handle / to enter filter mode.
	if msg.String() == "/" {
		b.filtering = true
		return b, nil
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

	for _, a := range visible {
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

		fmt.Fprintf(&buf, "  %s  %-12s  %-24s  %s%s\n",
			marker, typePart, a.Name, srcPart, description)
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
