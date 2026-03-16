package tuiapp

import (
	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui"
)

// Run launches the TUI with the given service dependencies.
func Run(
	deployer tui.Deployer,
	profiles tui.ProfileSwitcher,
	sources tui.SourceScanner,
	agents tui.AgentDetector,
	hasProjectDir bool,
	resolveProjectRoot func() (string, error),
) error {
	model := New(deployer, profiles, sources, agents, hasProjectDir, resolveProjectRoot)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}
