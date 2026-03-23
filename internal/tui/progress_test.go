package tui

import (
	"strings"
	"testing"
)

func TestProgressBar_ZeroTotal(t *testing.T) {
	p := newProgressBar(40)
	s := NewStyles(true)
	if got := p.View(s); got != "" {
		t.Errorf("expected empty view for zero total, got %q", got)
	}
}

func TestProgressBar_Update(t *testing.T) {
	p := newProgressBar(40)
	p = p.Update(progressMsg{completed: 5, total: 10, name: "skills/greeting"})

	if p.completed != 5 {
		t.Errorf("expected completed=5, got %d", p.completed)
	}
	if p.total != 10 {
		t.Errorf("expected total=10, got %d", p.total)
	}
	if p.name != "skills/greeting" {
		t.Errorf("expected name=%q, got %q", "skills/greeting", p.name)
	}
}

func TestProgressBar_ViewShowsCounter(t *testing.T) {
	p := newProgressBar(40)
	p = p.Update(progressMsg{completed: 8, total: 20, name: "agents/code-reviewer"})
	s := NewStyles(true)

	got := p.View(s)
	if !strings.Contains(got, "8/20") {
		t.Errorf("expected counter '8/20' in view, got %q", got)
	}
}

func TestProgressBar_ViewShowsName(t *testing.T) {
	p := newProgressBar(40)
	p = p.Update(progressMsg{completed: 1, total: 5, name: "skills/greeting"})
	s := NewStyles(true)

	got := p.View(s)
	if !strings.Contains(got, "skills/greeting") {
		t.Errorf("expected name 'skills/greeting' in view, got %q", got)
	}
}

func TestProgressBar_ViewNoNameWhenEmpty(t *testing.T) {
	p := newProgressBar(40)
	p = p.Update(progressMsg{completed: 1, total: 5, name: ""})
	s := NewStyles(true)

	got := p.View(s)
	// Should not have a second line.
	if strings.Count(got, "\n") > 0 {
		t.Errorf("expected no newline when name is empty, got %q", got)
	}
}

func TestProgressBar_Complete(t *testing.T) {
	p := newProgressBar(40)
	p = p.Update(progressMsg{completed: 10, total: 10, name: "done"})
	s := NewStyles(true)

	got := p.View(s)
	if !strings.Contains(got, "10/10") {
		t.Errorf("expected counter '10/10' in view, got %q", got)
	}
}
