package components

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/state"
	"github.com/armstrongl/nd/internal/tui"
)

// RowDetail holds expanded detail fields for a table row.
type RowDetail struct {
	SourcePath     string
	TargetPath     string
	Scope          string
	Origin         string
	Pinned         bool
	Profile        string
	ErrorDetail    string
	ShadowedBy     string
	Description    string
	Tags           []string
	TargetLanguage string
	TargetProject  string
	TargetAgent    string
}

// TableRow represents one deployed asset in the table.
type TableRow struct {
	Origin     nd.DeployOrigin
	Health     state.HealthStatus
	Name       string
	Source     string
	Scope      nd.Scope
	StatusText string
	Detail     *RowDetail
	IsFailed   bool
}

// Table renders the asset list with status/origin icons, inline expand, and scrolling.
type Table struct {
	Rows     []TableRow
	Selected int
	Expanded int // -1 if none expanded
	Width    int
	Height   int
	Offset   int // scroll offset
	Styles   tui.Styles
	EmptyMsg string // shown when Rows is empty
	keys     tui.KeyMap
}

// NewTable creates a table with default settings.
func NewTable() *Table {
	return &Table{
		Selected: 0,
		Expanded: -1,
		keys:     tui.DefaultKeyMap(),
	}
}

// SetRows replaces the table rows and sorts them (issues first, then alphabetical).
func (t *Table) SetRows(rows []TableRow) {
	sort.Slice(rows, func(i, j int) bool {
		iIssue := rows[i].Health != state.HealthOK
		jIssue := rows[j].Health != state.HealthOK
		if iIssue != jIssue {
			return iIssue // issues sort to top
		}
		return rows[i].Name < rows[j].Name
	})
	t.Rows = rows
	t.Selected = 0
	t.Expanded = -1
	t.Offset = 0
}

// Update handles key input for table navigation.
func (t *Table) Update(msg tea.Msg) (*Table, tea.Cmd) {
	if len(t.Rows) == 0 {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, t.keys.Up):
			if t.Selected > 0 {
				t.Selected--
			}
			t.ensureVisible()
		case key.Matches(msg, t.keys.Down):
			if t.Selected < len(t.Rows)-1 {
				t.Selected++
			}
			t.ensureVisible()
		case key.Matches(msg, t.keys.Enter):
			if t.Expanded == t.Selected {
				t.Expanded = -1
			} else {
				t.Expanded = t.Selected
			}
		}
	}
	return t, nil
}

// ensureVisible adjusts offset so the selected row is visible.
func (t *Table) ensureVisible() {
	visibleRows := t.visibleRowCount()
	if visibleRows <= 0 {
		return
	}
	if t.Selected < t.Offset {
		t.Offset = t.Selected
	}
	if t.Selected >= t.Offset+visibleRows {
		t.Offset = t.Selected - visibleRows + 1
	}
}

// visibleRowCount returns how many rows fit in the available height.
func (t *Table) visibleRowCount() int {
	if t.Height <= 2 {
		return 1
	}
	return t.Height - 1 // reserve 1 line for header
}

// View renders the table.
func (t Table) View() string {
	if len(t.Rows) == 0 {
		return t.emptyView()
	}

	var b strings.Builder

	// Column header
	b.WriteString(t.headerLine())
	b.WriteString("\n")

	// Visible rows
	visible := t.visibleRowCount()
	end := t.Offset + visible
	if end > len(t.Rows) {
		end = len(t.Rows)
	}

	for i := t.Offset; i < end; i++ {
		row := t.Rows[i]
		line := t.renderRow(row, i == t.Selected)
		b.WriteString(line)
		b.WriteString("\n")

		// Inline detail expansion
		if i == t.Expanded && row.Detail != nil {
			b.WriteString(t.renderDetail(row.Detail))
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(t.Rows) > visible {
		b.WriteString(fmt.Sprintf("  [%d-%d of %d]", t.Offset+1, end, len(t.Rows)))
	}

	return b.String()
}

// headerLine renders the column header.
func (t Table) headerLine() string {
	if t.Width > 0 && t.Width < 60 {
		return "  St Name"
	}
	if t.Width > 0 && t.Width < 80 {
		return "  St Name                          Scope   Status"
	}
	return "  St Name                          Source          Scope   Status"
}

// renderRow renders a single table row.
func (t Table) renderRow(row TableRow, selected bool) string {
	origin := t.originIcon(row.Origin)
	status := t.statusIcon(row.Health)

	name := row.Name
	if len(name) > 30 {
		name = name[:27] + "..."
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("%s %s %-30s", origin, status, name))

	// Add source column if width allows
	if t.Width == 0 || t.Width >= 80 {
		source := row.Source
		if len(source) > 15 {
			source = source[:12] + "..."
		}
		parts = append(parts, fmt.Sprintf("%-15s", source))
	}

	// Add scope column if width allows
	if t.Width == 0 || t.Width >= 60 {
		scope := "global"
		if row.Scope == nd.ScopeProject {
			scope = "project"
		}
		parts = append(parts, fmt.Sprintf("%-7s", scope))
	}

	// Status text
	parts = append(parts, row.StatusText)

	line := strings.Join(parts, " ")

	if selected {
		return tui.StyleTableSelected.Render(line)
	}
	return line
}

// originIcon returns the origin indicator character.
func (t Table) originIcon(origin nd.DeployOrigin) string {
	if origin == nd.OriginPinned {
		return t.Styles.OriginPinned.Render("P")
	}
	if strings.HasPrefix(string(origin), "profile:") {
		return t.Styles.OriginProfile.Render("@")
	}
	return " "
}

// statusIcon returns the health status indicator.
func (t Table) statusIcon(health state.HealthStatus) string {
	switch health {
	case state.HealthOK:
		return t.Styles.StatusOK.Render("*")
	case state.HealthBroken:
		return t.Styles.StatusBroken.Render("!")
	case state.HealthDrifted:
		return t.Styles.StatusDrifted.Render("~")
	default:
		return "-"
	}
}

// renderDetail renders the inline detail expansion.
func (t Table) renderDetail(d *RowDetail) string {
	var lines []string

	lines = append(lines, fmt.Sprintf("Source: %s", d.SourcePath))
	lines = append(lines, fmt.Sprintf("Target: %s", d.TargetPath))
	lines = append(lines, fmt.Sprintf("Scope:  %s", d.Scope))
	lines = append(lines, fmt.Sprintf("Origin: %s", d.Origin))

	if d.Pinned {
		lines = append(lines, "Pinned: yes")
	}
	if d.Profile != "" {
		lines = append(lines, fmt.Sprintf("Profile: %s", d.Profile))
	}
	if d.ErrorDetail != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", d.ErrorDetail))
	}
	if d.ShadowedBy != "" {
		lines = append(lines, fmt.Sprintf("Shadowed by: %s", d.ShadowedBy))
	}
	if d.Description != "" {
		lines = append(lines, fmt.Sprintf("Description: %s", d.Description))
	}
	if len(d.Tags) > 0 {
		lines = append(lines, fmt.Sprintf("Tags: %s", strings.Join(d.Tags, ", ")))
	}
	if d.TargetLanguage != "" {
		lines = append(lines, fmt.Sprintf("Language: %s", d.TargetLanguage))
	}
	if d.TargetProject != "" {
		lines = append(lines, fmt.Sprintf("Project: %s", d.TargetProject))
	}
	if d.TargetAgent != "" {
		lines = append(lines, fmt.Sprintf("Agent: %s", d.TargetAgent))
	}

	return tui.StyleDetailBox.Render(strings.Join(lines, "\n"))
}

// emptyView renders the empty state message.
func (t Table) emptyView() string {
	msg := t.EmptyMsg
	if msg == "" {
		msg = "No assets deployed.\nPress d to deploy assets from your sources."
	}
	return t.Styles.Empty.Render(msg)
}
