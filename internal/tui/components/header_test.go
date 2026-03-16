package components_test

import (
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/tui/components"
)

func TestHeaderViewGlobal(t *testing.T) {
	h := components.Header{
		Scope:   nd.ScopeGlobal,
		Agent:   "Claude Code",
		Profile: "default",
		Width:   120,
		Styles:  tui.DefaultStyles(),
	}
	view := h.View()
	if !strings.Contains(view, "nd") {
		t.Error("header should contain 'nd'")
	}
	if !strings.Contains(view, "Global") {
		t.Error("header should contain 'Global'")
	}
	if !strings.Contains(view, "Claude Code") {
		t.Error("header should contain agent name")
	}
	if !strings.Contains(view, "profile: default") {
		t.Error("header should contain profile")
	}
}

func TestHeaderViewProject(t *testing.T) {
	h := components.Header{
		Scope: nd.ScopeProject,
		Width: 120,
		Styles: tui.DefaultStyles(),
	}
	view := h.View()
	if !strings.Contains(view, "Project") {
		t.Error("header should contain 'Project'")
	}
}

func TestHeaderViewIssues(t *testing.T) {
	h := components.Header{
		Scope:      nd.ScopeGlobal,
		IssueCount: 3,
		Width:      120,
		Styles:     tui.DefaultStyles(),
	}
	view := h.View()
	if !strings.Contains(view, "3 issues") {
		t.Error("header should show issue count")
	}
}

func TestHeaderViewSingleIssue(t *testing.T) {
	h := components.Header{
		Scope:      nd.ScopeGlobal,
		IssueCount: 1,
		Width:      120,
		Styles:     tui.DefaultStyles(),
	}
	view := h.View()
	if !strings.Contains(view, "1 issue") {
		t.Error("header should show singular 'issue'")
	}
	if strings.Contains(view, "1 issues") {
		t.Error("should not pluralize for 1 issue")
	}
}

func TestHeaderViewSourceWarning(t *testing.T) {
	h := components.Header{
		Scope:      nd.ScopeGlobal,
		SourceWarn: 2,
		Width:      120,
		Styles:     tui.DefaultStyles(),
	}
	view := h.View()
	if !strings.Contains(view, "2 sources unavailable") {
		t.Error("header should show source warning")
	}
}

func TestHeaderViewNarrow(t *testing.T) {
	h := components.Header{
		Scope:      nd.ScopeGlobal,
		Agent:      "Claude Code",
		Profile:    "default",
		IssueCount: 5,
		SourceWarn: 2,
		Width:      30,
		Styles:     tui.DefaultStyles(),
	}
	view := h.View()
	// Should truncate but not panic
	if view == "" {
		t.Error("narrow header should not be empty")
	}
}
