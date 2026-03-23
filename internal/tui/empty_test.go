package tui

import (
	"strings"
	"testing"
)

func TestEmptyFunctionsNonEmpty(t *testing.T) {
	cases := []struct {
		name string
		fn   func() string
	}{
		{"NoSources", NoSources},
		{"NoAssets", NoAssets},
		{"NothingDeployed", NothingDeployed},
		{"NoProfiles", NoProfiles},
		{"NoSnapshots", NoSnapshots},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn()
			if got == "" {
				t.Errorf("%s() returned empty string", tc.name)
			}
		})
	}
}

func TestEmptyFunctionsContainHint(t *testing.T) {
	// Each function must contain an actionable hint: at least one sentence
	// describing what the user should do next.
	cases := []struct {
		name string
		fn   func() string
		hint string // substring that must appear in the hint portion
	}{
		{"NoSources", NoSources, "nd source add"},
		{"NoAssets", NoAssets, "nd source add"},
		{"NothingDeployed", NothingDeployed, "Deploy"},
		{"NoProfiles", NoProfiles, "nd profile create"},
		{"NoSnapshots", NoSnapshots, "nd snapshot save"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn()
			if !strings.Contains(got, tc.hint) {
				t.Errorf("%s() should contain hint %q, got %q", tc.name, tc.hint, got)
			}
		})
	}
}

func TestAllDeployedContainsTypeName(t *testing.T) {
	got := AllDeployed("skills")
	if got == "" {
		t.Fatal("AllDeployed(\"skills\") returned empty string")
	}
	if !strings.Contains(got, "skills") {
		t.Errorf("AllDeployed(\"skills\") should contain \"skills\", got %q", got)
	}
}

func TestAllDeployedNonEmpty(t *testing.T) {
	got := AllDeployed("agents")
	if got == "" {
		t.Error("AllDeployed(\"agents\") returned empty string")
	}
}

func TestNoSourcesMentionsSource(t *testing.T) {
	got := NoSources()
	if !strings.Contains(got, "source") {
		t.Errorf("NoSources() should mention \"source\", got %q", got)
	}
}

func TestNothingDeployedMentionsDeploy(t *testing.T) {
	got := NothingDeployed()
	if !strings.Contains(got, "Deploy") {
		t.Errorf("NothingDeployed() should mention \"Deploy\", got %q", got)
	}
}
