package tui

// AppState represents the current state of the TUI application.
type AppState int

const (
	StatePicker AppState = iota
	StateMenu
	StateDashboard
	StateDetail
	StateFuzzy
	StateListPicker
	StatePrompt
	StateConfirm
	StateLoading
)

// ToastLevel indicates the severity of a toast notification.
type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)
