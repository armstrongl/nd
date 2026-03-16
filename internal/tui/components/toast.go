package components

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/tui"
)

const toastDuration = 3 * time.Second

// Toast renders a temporary status message that auto-dismisses.
type Toast struct {
	Message string
	Level   tui.ToastLevel
	Visible bool
}

// NewToast creates a visible toast and returns the auto-dismiss command.
func NewToast(msg string, level tui.ToastLevel) (Toast, tea.Cmd) {
	t := Toast{
		Message: msg,
		Level:   level,
		Visible: true,
	}
	return t, t.dismissCmd()
}

// dismissCmd returns a tick command that fires a ToastDismissMsg after the duration.
func (t Toast) dismissCmd() tea.Cmd {
	return tea.Tick(toastDuration, func(_ time.Time) tea.Msg {
		return tui.ToastDismissMsg{}
	})
}

// Dismiss hides the toast.
func (t *Toast) Dismiss() {
	t.Visible = false
	t.Message = ""
}

// View renders the toast bar.
func (t Toast) View() string {
	if !t.Visible || t.Message == "" {
		return ""
	}

	prefix := ""
	switch t.Level {
	case tui.ToastSuccess:
		prefix = "OK "
	case tui.ToastWarning:
		prefix = "WARN "
	case tui.ToastError:
		prefix = "ERR "
	}

	return tui.StyleToast.Render(prefix + t.Message)
}
