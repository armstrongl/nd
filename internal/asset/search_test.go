package asset

import (
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func testIndex() *Index {
	return NewIndex([]Asset{
		{Identity: Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "my-skill"}, SourcePath: "/src/skills/my-skill"},
		{Identity: Identity{SourceID: "s1", Type: nd.AssetCommand, Name: "my-cmd"}, SourcePath: "/src/commands/my-cmd"},
		{Identity: Identity{SourceID: "s1", Type: nd.AssetSkill, Name: "other-skill"}, SourcePath: "/src/skills/other-skill"},
		{Identity: Identity{SourceID: "s2", Type: nd.AssetRule, Name: "My-Skill"}, SourcePath: "/src2/rules/My-Skill"},
	})
}

func TestSearchByName_Found(t *testing.T) {
	idx := testIndex()
	results := idx.SearchByName("my-skill")
	// Should find "my-skill" (skill from s1) and "My-Skill" (rule from s2, case-insensitive)
	if len(results) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(results))
	}
}

func TestSearchByName_NotFound(t *testing.T) {
	idx := testIndex()
	results := idx.SearchByName("nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(results))
	}
}

func TestSearchByName_CaseInsensitive(t *testing.T) {
	idx := testIndex()
	results := idx.SearchByName("MY-SKILL")
	if len(results) != 2 {
		t.Fatalf("expected 2 matches for uppercase query, got %d", len(results))
	}
}

func TestSearchByTypeAndName_Found(t *testing.T) {
	idx := testIndex()
	a := idx.SearchByTypeAndName(nd.AssetSkill, "my-skill")
	if a == nil {
		t.Fatal("expected to find skill 'my-skill'")
	}
	if a.SourceID != "s1" {
		t.Errorf("expected source s1, got %s", a.SourceID)
	}
}

func TestSearchByTypeAndName_WrongType(t *testing.T) {
	idx := testIndex()
	a := idx.SearchByTypeAndName(nd.AssetCommand, "my-skill")
	if a != nil {
		t.Error("expected nil for wrong type")
	}
}

func TestSearchByTypeAndName_NotFound(t *testing.T) {
	idx := testIndex()
	a := idx.SearchByTypeAndName(nd.AssetSkill, "nonexistent")
	if a != nil {
		t.Error("expected nil for nonexistent asset")
	}
}

func TestSearchByTypeAndName_CaseInsensitive(t *testing.T) {
	idx := testIndex()
	a := idx.SearchByTypeAndName(nd.AssetRule, "my-skill")
	if a == nil {
		t.Fatal("expected to find rule 'My-Skill' via case-insensitive match")
	}
	if a.Name != "My-Skill" {
		t.Errorf("expected original case 'My-Skill', got %q", a.Name)
	}
}
