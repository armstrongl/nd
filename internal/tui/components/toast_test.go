package components_test

import (
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/tui/components"
)

func TestNewToast(t *testing.T) {
	toast, cmd := components.NewToast("Deployed successfully", tui.ToastSuccess)
	if !toast.Visible {
		t.Error("new toast should be visible")
	}
	if toast.Message != "Deployed successfully" {
		t.Errorf("toast message: got %q", toast.Message)
	}
	if toast.Level != tui.ToastSuccess {
		t.Error("toast level should be success")
	}
	if cmd == nil {
		t.Error("toast should return a dismiss command")
	}
}

func TestToastViewSuccess(t *testing.T) {
	toast := components.Toast{
		Message: "All good",
		Level:   tui.ToastSuccess,
		Visible: true,
	}
	view := toast.View()
	if !strings.Contains(view, "OK") {
		t.Error("success toast should contain OK prefix")
	}
	if !strings.Contains(view, "All good") {
		t.Error("toast should contain message")
	}
}

func TestToastViewError(t *testing.T) {
	toast := components.Toast{
		Message: "Something failed",
		Level:   tui.ToastError,
		Visible: true,
	}
	view := toast.View()
	if !strings.Contains(view, "ERR") {
		t.Error("error toast should contain ERR prefix")
	}
}

func TestToastViewWarning(t *testing.T) {
	toast := components.Toast{
		Message: "Watch out",
		Level:   tui.ToastWarning,
		Visible: true,
	}
	view := toast.View()
	if !strings.Contains(view, "WARN") {
		t.Error("warning toast should contain WARN prefix")
	}
}

func TestToastViewInfo(t *testing.T) {
	toast := components.Toast{
		Message: "FYI",
		Level:   tui.ToastInfo,
		Visible: true,
	}
	view := toast.View()
	if !strings.Contains(view, "FYI") {
		t.Error("info toast should contain message")
	}
}

func TestToastViewHidden(t *testing.T) {
	toast := components.Toast{
		Message: "Hidden",
		Level:   tui.ToastInfo,
		Visible: false,
	}
	view := toast.View()
	if view != "" {
		t.Error("hidden toast should render empty")
	}
}

func TestToastDismiss(t *testing.T) {
	toast := components.Toast{
		Message: "Will go away",
		Level:   tui.ToastInfo,
		Visible: true,
	}
	toast.Dismiss()
	if toast.Visible {
		t.Error("dismissed toast should not be visible")
	}
	if toast.Message != "" {
		t.Error("dismissed toast message should be cleared")
	}
}
