package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
)

// --- Glyph module ---

func TestNewGlyphsNonEmpty(t *testing.T) {
	glyphs := map[string]string{
		"GlyphDeployed":   GlyphDeployed,
		"GlyphNew":        GlyphNew,
		"GlyphActive":     GlyphActive,
		"GlyphPinned":     GlyphPinned,
		"GlyphScrollUp":   GlyphScrollUp,
		"GlyphScrollDown": GlyphScrollDown,
		"GlyphDryRun":     GlyphDryRun,
		"GlyphWarning":    GlyphWarning,
	}
	for name, val := range glyphs {
		if val == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestStyleGlyphHelpers_Unstyled(t *testing.T) {
	s := testStyles() // empty styles: Render returns its input unchanged
	cases := []struct {
		name, got, want string
	}{
		{"Deployed", s.Deployed(), GlyphDeployed},
		{"NewBadge", s.NewBadge(), GlyphNew},
		{"Active", s.Active(), GlyphActive},
		{"Pinned", s.Pinned(), GlyphPinned},
	}
	for _, c := range cases {
		if !strings.Contains(c.got, c.want) {
			t.Errorf("%s() = %q, want to contain %q", c.name, c.got, c.want)
		}
	}
}

func TestStyleGlyphHelpers_Styled(t *testing.T) {
	// Even when a colour is applied, the glyph text must remain present.
	s := NewStyles(true)
	if !strings.Contains(s.Deployed(), GlyphDeployed) {
		t.Errorf("styled Deployed() should contain %q, got %q", GlyphDeployed, s.Deployed())
	}
	if !strings.Contains(s.NewBadge(), GlyphNew) {
		t.Errorf("styled NewBadge() should contain %q, got %q", GlyphNew, s.NewBadge())
	}
}

// --- Recency window ---

func TestRecencyWindow(t *testing.T) {
	cases := []struct {
		days int
		want time.Duration
	}{
		{0, 7 * 24 * time.Hour},  // unset -> default 7 days
		{-3, 7 * 24 * time.Hour}, // negative -> default 7 days
		{1, 24 * time.Hour},
		{14, 14 * 24 * time.Hour},
	}
	for _, c := range cases {
		if got := recencyWindow(c.days); got != c.want {
			t.Errorf("recencyWindow(%d) = %v, want %v", c.days, got, c.want)
		}
	}
}

// --- isNew / isNewAt ---

func TestIsNew_EmptyPathAndNil(t *testing.T) {
	a := &asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "n"}}
	if isNew(a, 7*24*time.Hour) {
		t.Error("asset with empty SourcePath must not be new")
	}
	if isNew(nil, 7*24*time.Hour) {
		t.Error("nil asset must not be new")
	}
}

func TestIsNew_StatError(t *testing.T) {
	a := &asset.Asset{SourcePath: filepath.Join(t.TempDir(), "does-not-exist.md")}
	if isNew(a, 7*24*time.Hour) {
		t.Error("asset with an unstat-able SourcePath must not be new")
	}
}

func TestIsNew_RecentAndOld(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &asset.Asset{SourcePath: p}

	if !isNew(a, 7*24*time.Hour) {
		t.Error("freshly modified file should be new within a 7-day window")
	}

	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	if isNew(a, 7*24*time.Hour) {
		t.Error("file modified 30 days ago should not be new in a 7-day window")
	}
	if !isNew(a, 60*24*time.Hour) {
		t.Error("file modified 30 days ago should be new in a 60-day window")
	}
}

func TestIsNewAt_Boundary(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	mtime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(p, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	a := &asset.Asset{SourcePath: p}
	window := 7 * 24 * time.Hour

	// now exactly one window after mtime: inclusive boundary counts as new.
	if !isNewAt(a, window, mtime.Add(window)) {
		t.Error("mtime exactly at the window boundary should count as new")
	}
	// One second beyond the window: no longer new.
	if isNewAt(a, window, mtime.Add(window).Add(time.Second)) {
		t.Error("mtime just past the window should not be new")
	}
	// Comfortably inside the window: new.
	if !isNewAt(a, window, mtime.Add(24*time.Hour)) {
		t.Error("mtime one day ago should be new in a 7-day window")
	}
}

// --- Browse indicators + sort ---

func TestBrowseScreen_NewBadgeShown(t *testing.T) {
	p := filepath.Join(t.TempDir(), "recent.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw := []*asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "fresh"}, SourcePath: p},
	}
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: raw})

	v := s.View()
	if !strings.Contains(v.Content, GlyphNew) {
		t.Errorf("recently modified asset should show the new badge %q, got: %q", GlyphNew, v.Content)
	}
}

func TestBrowseScreen_OldAssetNoNewBadge(t *testing.T) {
	p := filepath.Join(t.TempDir(), "old.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	raw := []*asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "stale"}, SourcePath: p},
	}
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: raw})

	v := s.View()
	if strings.Contains(v.Content, GlyphNew) {
		t.Errorf("stale asset should not show the new badge, got: %q", v.Content)
	}
}

func TestBrowseScreen_DeployedAndNewSimultaneously(t *testing.T) {
	p := filepath.Join(t.TempDir(), "recent.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &asset.Asset{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "live-and-fresh"}, SourcePath: p}
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{
		assets:   []*asset.Asset{a},
		deployed: map[string]bool{a.String(): true},
	})

	v := s.View()
	if !strings.Contains(v.Content, GlyphDeployed) {
		t.Errorf("asset should show deployed glyph, got: %q", v.Content)
	}
	if !strings.Contains(v.Content, GlyphNew) {
		t.Errorf("asset should also show new badge, got: %q", v.Content)
	}
}

func TestBrowseScreen_SortOrder(t *testing.T) {
	p := filepath.Join(t.TempDir(), "new.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw := []*asset.Asset{
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "zed-deployed"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "bob-undeployed"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "amy-undeployed"}},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "new-one"}, SourcePath: p},
		{Identity: asset.Identity{SourceID: "s", Type: nd.AssetSkill, Name: "kai-deployed"}},
	}
	deployed := map[string]bool{
		"s:skills/zed-deployed": true,
		"s:skills/kai-deployed": true,
	}
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: raw, deployed: deployed})

	var order []string
	for _, a := range s.assets {
		order = append(order, a.Name)
	}
	want := []string{"new-one", "amy-undeployed", "bob-undeployed", "kai-deployed", "zed-deployed"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("sort order = %v, want %v", order, want)
	}
}

func TestBrowseScreen_SortStable(t *testing.T) {
	// Two undeployed, non-new assets with identical names but different sources.
	a1 := &asset.Asset{Identity: asset.Identity{SourceID: "aaa", Type: nd.AssetSkill, Name: "dup"}}
	a2 := &asset.Asset{Identity: asset.Identity{SourceID: "bbb", Type: nd.AssetSkill, Name: "dup"}}
	s := newBrowseScreen(newMockServices(), NewStyles(true), true)
	s.Update(browseLoadedMsg{assets: []*asset.Asset{a1, a2}})

	if s.assets[0].SourceID != "aaa" || s.assets[1].SourceID != "bbb" {
		t.Errorf("stable sort should preserve input order for equal keys, got %q,%q",
			s.assets[0].SourceID, s.assets[1].SourceID)
	}
}

// --- Deploy picker labels ---

func TestAssetOptionLabel_NewBadge(t *testing.T) {
	p := filepath.Join(t.TempDir(), "recent.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &asset.Asset{Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "fresh"}, SourcePath: p}

	label := assetOptionLabel(testStyles(), a, recencyWindow(0))
	if !strings.Contains(label, GlyphNew) {
		t.Errorf("recent asset label should include the new badge, got %q", label)
	}
	if !strings.Contains(label, "fresh") || !strings.Contains(label, "local") {
		t.Errorf("label should still include name and source, got %q", label)
	}
}

func TestAssetOptionLabel_NoBadgeKeepsDescription(t *testing.T) {
	a := &asset.Asset{
		Identity: asset.Identity{SourceID: "local", Type: nd.AssetSkill, Name: "old"},
		Meta:     &asset.ContextMeta{Description: "does things"},
	}

	label := assetOptionLabel(testStyles(), a, recencyWindow(0))
	if strings.Contains(label, GlyphNew) {
		t.Errorf("asset with empty SourcePath should not show a new badge, got %q", label)
	}
	if !strings.Contains(label, "does things") {
		t.Errorf("label should preserve the description, got %q", label)
	}
}
