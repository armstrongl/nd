package tui

import tea "charm.land/bubbletea/v2"

// Screen is a TUI view that the root model manages on a stack.
type Screen interface {
	tea.Model
	Title() string      // Displayed in breadcrumb context.
	InputActive() bool  // True when a text field has focus (suppresses global keys).
}

// Navigation messages — screens emit these, root model handles them.

// NavigateMsg pushes a new screen onto the navigation stack.
type NavigateMsg struct{ Screen Screen }

// BackMsg pops one level from the navigation stack.
type BackMsg struct{}

// PopToRootMsg clears the navigation stack and returns to the main menu.
type PopToRootMsg struct{}

// RefreshHeaderMsg re-queries state for header counts.
type RefreshHeaderMsg struct{}

// ScopeSwitchedMsg signals that the deployment scope changed.
// Screens caching scope-dependent data should reload when they receive this.
type ScopeSwitchedMsg struct{}
