package tui

import (
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

// Compile-time check: mockServices satisfies Services.
var _ Services = (*mockServices)(nil)

func TestMockServices_DefaultReturnValues(t *testing.T) {
	m := newMockServices()

	t.Run("SourceManager returns nil nil", func(t *testing.T) {
		sm, err := m.SourceManager()
		if sm != nil || err != nil {
			t.Errorf("expected (nil, nil), got (%v, %v)", sm, err)
		}
	})

	t.Run("ScanIndex returns empty summary", func(t *testing.T) {
		summary, err := m.ScanIndex()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if summary == nil {
			t.Fatal("expected non-nil ScanSummary")
		}
	})

	t.Run("AgentRegistry returns nil nil", func(t *testing.T) {
		reg, err := m.AgentRegistry()
		if reg != nil || err != nil {
			t.Errorf("expected (nil, nil), got (%v, %v)", reg, err)
		}
	})

	t.Run("DefaultAgent returns claude-code", func(t *testing.T) {
		ag, err := m.DefaultAgent()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ag == nil || ag.Name != "claude-code" {
			t.Errorf("expected agent named claude-code, got %v", ag)
		}
	})

	t.Run("DeployEngine returns nil nil", func(t *testing.T) {
		eng, err := m.DeployEngine()
		if eng != nil || err != nil {
			t.Errorf("expected (nil, nil), got (%v, %v)", eng, err)
		}
	})

	t.Run("StateStore returns nil", func(t *testing.T) {
		if s := m.StateStore(); s != nil {
			t.Errorf("expected nil, got %v", s)
		}
	})

	t.Run("ProfileManager returns nil nil", func(t *testing.T) {
		pm, err := m.ProfileManager()
		if pm != nil || err != nil {
			t.Errorf("expected (nil, nil), got (%v, %v)", pm, err)
		}
	})

	t.Run("ProfileStore returns nil nil", func(t *testing.T) {
		ps, err := m.ProfileStore()
		if ps != nil || err != nil {
			t.Errorf("expected (nil, nil), got (%v, %v)", ps, err)
		}
	})

	t.Run("OpLog returns nil", func(t *testing.T) {
		if ol := m.OpLog(); ol != nil {
			t.Errorf("expected nil, got %v", ol)
		}
	})

	t.Run("GetScope returns global", func(t *testing.T) {
		if scope := m.GetScope(); scope != nd.ScopeGlobal {
			t.Errorf("expected %q, got %q", nd.ScopeGlobal, scope)
		}
	})

	t.Run("GetConfigPath returns test path", func(t *testing.T) {
		if p := m.GetConfigPath(); p != "/tmp/nd-test/config.yaml" {
			t.Errorf("expected /tmp/nd-test/config.yaml, got %q", p)
		}
	})

	t.Run("IsDryRun returns false", func(t *testing.T) {
		if m.IsDryRun() {
			t.Error("expected false")
		}
	})
}

func TestMockServices_ResetForScope(t *testing.T) {
	m := newMockServices()

	m.ResetForScope(nd.ScopeProject, "/some/project")
	m.ResetForScope(nd.ScopeGlobal, "")

	if len(m.resetCalls) != 2 {
		t.Fatalf("expected 2 reset calls, got %d", len(m.resetCalls))
	}

	if m.resetCalls[0].Scope != nd.ScopeProject {
		t.Errorf("call 0: expected scope %q, got %q", nd.ScopeProject, m.resetCalls[0].Scope)
	}
	if m.resetCalls[0].ProjectRoot != "/some/project" {
		t.Errorf("call 0: expected project root %q, got %q", "/some/project", m.resetCalls[0].ProjectRoot)
	}

	if m.resetCalls[1].Scope != nd.ScopeGlobal {
		t.Errorf("call 1: expected scope %q, got %q", nd.ScopeGlobal, m.resetCalls[1].Scope)
	}
	if m.resetCalls[1].ProjectRoot != "" {
		t.Errorf("call 1: expected empty project root, got %q", m.resetCalls[1].ProjectRoot)
	}
}

func TestMockServices_OverrideFunctions(t *testing.T) {
	m := newMockServices()

	m.getScopeFn = func() nd.Scope { return nd.ScopeProject }
	m.isDryRunFn = func() bool { return true }
	m.getConfigPathFn = func() string { return "/custom/config.yaml" }

	if scope := m.GetScope(); scope != nd.ScopeProject {
		t.Errorf("expected %q, got %q", nd.ScopeProject, scope)
	}
	if !m.IsDryRun() {
		t.Error("expected IsDryRun() true after override")
	}
	if p := m.GetConfigPath(); p != "/custom/config.yaml" {
		t.Errorf("expected /custom/config.yaml, got %q", p)
	}
}
