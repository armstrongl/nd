package components_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/tui/components"
)

func sampleRows() []components.TableRow {
	return []components.TableRow{
		{Name: "my-skill", Source: "local", Scope: nd.ScopeGlobal, Health: state.HealthOK, Origin: nd.OriginManual, StatusText: "ok"},
		{Name: "broken-agent", Source: "remote", Scope: nd.ScopeProject, Health: state.HealthBroken, Origin: nd.OriginPinned, StatusText: "broken"},
		{Name: "drifted-cmd", Source: "local", Scope: nd.ScopeGlobal, Health: state.HealthDrifted, Origin: nd.OriginProfile("dev"), StatusText: "drifted"},
	}
}

func TestTableSetRowsSortsIssuesFirst(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.SetRows(sampleRows())

	// Issues (broken, drifted) should sort before healthy
	if tbl.Rows[0].Health == state.HealthOK {
		t.Error("healthy rows should not be first")
	}
	// broken-agent and drifted-cmd should be before my-skill
	if tbl.Rows[len(tbl.Rows)-1].Health != state.HealthOK {
		t.Error("healthy rows should be last")
	}
}

func TestTableSetRowsResetsState(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Selected = 5
	tbl.Expanded = 2
	tbl.Offset = 3

	tbl.SetRows(sampleRows())
	if tbl.Selected != 0 {
		t.Errorf("Selected should reset to 0, got %d", tbl.Selected)
	}
	if tbl.Expanded != -1 {
		t.Errorf("Expanded should reset to -1, got %d", tbl.Expanded)
	}
	if tbl.Offset != 0 {
		t.Errorf("Offset should reset to 0, got %d", tbl.Offset)
	}
}

func TestTableViewWide(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 120
	tbl.Height = 20
	tbl.SetRows(sampleRows())

	view := tbl.View()
	if !strings.Contains(view, "Source") {
		t.Error("wide view should show Source column header")
	}
	if !strings.Contains(view, "my-skill") {
		t.Error("view should contain row names")
	}
}

func TestTableViewNarrow(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 50
	tbl.Height = 20
	tbl.SetRows(sampleRows())

	view := tbl.View()
	if strings.Contains(view, "Source") {
		t.Error("narrow view should hide Source column")
	}
}

func TestTableViewMedium(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 70
	tbl.Height = 20
	tbl.SetRows(sampleRows())

	view := tbl.View()
	// Should have Scope but not Source header
	if strings.Contains(view, "Source") {
		t.Error("medium view should hide Source column")
	}
}

func TestTableNavigation(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 120
	tbl.Height = 20
	tbl.SetRows(sampleRows())

	// Move down
	tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if tbl.Selected != 1 {
		t.Errorf("after down: expected 1, got %d", tbl.Selected)
	}

	// Move up
	tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if tbl.Selected != 0 {
		t.Errorf("after up: expected 0, got %d", tbl.Selected)
	}

	// Don't go below 0
	tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if tbl.Selected != 0 {
		t.Errorf("should stay at 0, got %d", tbl.Selected)
	}
}

func TestTableExpandCollapse(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 120
	tbl.Height = 20
	rows := sampleRows()
	rows[0].Detail = &components.RowDetail{
		SourcePath: "/src/skills/my-skill",
		TargetPath: "/home/.claude/skills/my-skill",
		Scope:      "global",
		Origin:     "manual",
	}
	tbl.SetRows(rows)

	// Expand first row (issues sort first, so find the one with detail)
	// Navigate to the row with detail (my-skill is last since it's healthy)
	tbl.Selected = len(tbl.Rows) - 1
	tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if tbl.Expanded != tbl.Selected {
		t.Error("enter should expand selected row")
	}

	view := tbl.View()
	if !strings.Contains(view, "Source:") {
		t.Error("expanded view should show detail")
	}

	// Collapse
	tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if tbl.Expanded != -1 {
		t.Error("second enter should collapse")
	}
}

func TestTableEmptyState(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 120
	tbl.Height = 20

	view := tbl.View()
	if !strings.Contains(view, "No assets deployed") {
		t.Error("empty table should show default empty message")
	}
}

func TestTableCustomEmptyMsg(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.EmptyMsg = "No skills deployed.\nPress d to deploy a skill."

	view := tbl.View()
	if !strings.Contains(view, "No skills deployed") {
		t.Error("should show custom empty message")
	}
}

func TestTableScrolling(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 120
	tbl.Height = 4 // Only 3 visible rows (minus 1 for header)

	// Create 10 rows
	rows := make([]components.TableRow, 10)
	for i := range rows {
		rows[i] = components.TableRow{
			Name:       strings.Repeat("row", 1),
			Health:     state.HealthOK,
			Origin:     nd.OriginManual,
			StatusText: "ok",
		}
	}
	tbl.SetRows(rows)

	// Navigate down past visible area
	for i := 0; i < 5; i++ {
		tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}

	if tbl.Offset == 0 {
		t.Error("offset should have changed after scrolling down")
	}

	view := tbl.View()
	if !strings.Contains(view, "of 10") {
		t.Error("should show scroll indicator")
	}
}

func TestTableEmptyUpdateNoPanic(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	// Should not panic on empty table
	tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	tbl, _ = tbl.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = tbl
}

func TestTableOriginIcons(t *testing.T) {
	tbl := components.NewTable()
	tbl.Styles = tui.DefaultStyles()
	tbl.Width = 120
	tbl.Height = 20
	tbl.SetRows(sampleRows())

	view := tbl.View()
	// Pinned origin should show P
	if !strings.Contains(view, "P") {
		t.Error("pinned origin should show P icon")
	}
	// Profile origin should show @
	if !strings.Contains(view, "@") {
		t.Error("profile origin should show @ icon")
	}
}
