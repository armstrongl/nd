package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

// exportScreen is a static notice shown when the user selects "Export plugin"
// from the main menu. Export has no interactive TUI flow — it runs as the
// nd export command-line action — so this screen tells the user where to run
// it and returns to the main menu on enter/esc.
type exportScreen struct {
	styles Styles
}

// newExportScreen builds the export notice screen. It takes the same
// (svc, styles, isDark) signature as the other screen constructors so the main
// menu can wire it uniformly; only styles is needed to render the notice.
func newExportScreen(svc Services, styles Styles, isDark bool) *exportScreen {
	return &exportScreen{styles: styles}
}

func (s *exportScreen) Title() string    { return "Export" }
func (s *exportScreen) InputActive() bool { return false }

func (s *exportScreen) Init() tea.Cmd { return nil }

func (s *exportScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "enter", "esc":
			return s, func() tea.Msg { return PopToRootMsg{} }
		}
	}
	return s, nil
}

func (s *exportScreen) View() tea.View {
	return tea.NewView(fmt.Sprintf("  %s\n\n  %s",
		ExportCLIOnly(),
		s.styles.Subtle.Render("Press enter to return.")))
}
