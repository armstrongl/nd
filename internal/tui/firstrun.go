package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/armstrongl/nd/internal/nd"
)

// hasUserSources checks whether any non-builtin sources are configured.
func hasUserSources(svc Services) bool {
	sm, err := svc.SourceManager()
	if err != nil || sm == nil {
		return false
	}
	cfg := sm.Config()
	for _, s := range cfg.Sources {
		if s.Type != nd.SourceBuiltin {
			return true
		}
	}
	return false
}

type firstRunScreen struct {
	svc       Services
	form      *huh.Form
	choice    string
	styles    Styles
	isDark    bool
	navigated bool
}

func newFirstRunScreen(svc Services, styles Styles, isDark bool) *firstRunScreen {
	f := &firstRunScreen{
		svc:    svc,
		styles: styles,
		isDark: isDark,
	}

	f.form = huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Welcome to nd").
				Description("No sources configured yet. Add a source directory to get started."),
			huh.NewSelect[string]().
				Title("What would you like to do?").
				Options(
					huh.NewOption("Add a source", "add"),
					huh.NewOption("Quit", "quit"),
				).
				Value(&f.choice),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCatppuccin))

	return f
}

func (f *firstRunScreen) Title() string    { return "Welcome" }
func (f *firstRunScreen) InputActive() bool { return !f.navigated }

func (f *firstRunScreen) Init() tea.Cmd {
	return f.form.Init()
}

func (f *firstRunScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f.navigated {
		return f, nil
	}

	model, cmd := f.form.Update(msg)
	if form, ok := model.(*huh.Form); ok {
		f.form = form
	}

	if f.form.State == huh.StateCompleted {
		f.navigated = true
		return f, f.handleSelection()
	}

	if f.form.State == huh.StateAborted {
		f.navigated = true
		return f, tea.Quit
	}

	return f, cmd
}

func (f *firstRunScreen) View() tea.View {
	return tea.NewView(f.form.View())
}

func (f *firstRunScreen) handleSelection() tea.Cmd {
	switch f.choice {
	case "add":
		screen := newSourceScreen(f.svc, f.styles, f.isDark)
		return func() tea.Msg { return NavigateMsg{Screen: screen} }
	case "quit":
		return tea.Quit
	default:
		return nil
	}
}
