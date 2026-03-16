package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the TUI.
// Each key is owned by exactly one AppState per the key ownership matrix.
type KeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding

	Enter     key.Binding
	Esc       key.Binding
	Backspace key.Binding

	Deploy   key.Binding
	Remove   key.Binding
	Sync     key.Binding
	Fix      key.Binding
	Pin      key.Binding
	Profile  key.Binding
	Snapshot key.Binding
	Retry    key.Binding

	Quit key.Binding

	Yes key.Binding
	No  key.Binding
}

// DefaultKeyMap returns the standard keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("left", "prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("right", "next tab"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Backspace: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "back"),
		),
		Deploy: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "deploy"),
		),
		Remove: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "remove"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync"),
		),
		Fix: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fix issues"),
		),
		Pin: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pin/unpin"),
		),
		Profile: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "switch profile"),
		),
		Snapshot: key.NewBinding(
			key.WithKeys("W"),
			key.WithHelp("W", "snapshots"),
		),
		Retry: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "retry failed"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
	}
}
