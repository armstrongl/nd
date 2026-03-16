package state_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/state"
)

func TestHealthStatusString(t *testing.T) {
	tests := []struct {
		h    state.HealthStatus
		want string
	}{
		{state.HealthOK, "ok"},
		{state.HealthBroken, "broken"},
		{state.HealthDrifted, "drifted"},
		{state.HealthOrphaned, "orphaned"},
		{state.HealthMissing, "missing"},
		{state.HealthStatus(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.h.String(); got != tt.want {
			t.Errorf("HealthStatus(%d).String() = %q, want %q", tt.h, got, tt.want)
		}
	}
}
