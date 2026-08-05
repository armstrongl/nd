package tui

import (
	"testing"

	"charm.land/huh/v2"
)

// hasHelpItem reports whether items contains an entry with the given key and desc.
func hasHelpItem(items []HelpItem, key, desc string) bool {
	for _, it := range items {
		if it.Key == key && it.Desc == desc {
			return true
		}
	}
	return false
}

// hasHelpKey reports whether items contains any entry with the given key.
func hasHelpKey(items []HelpItem, key string) bool {
	for _, it := range items {
		if it.Key == key {
			return true
		}
	}
	return false
}

// TestHelpSteps_PinSelect_ShowsToggleNotSelect checks the huh MultiSelect step
// advertises "x/space toggle" and never the misleading "enter select".
func TestHelpSteps_PinSelect_ShowsToggleNotSelect(t *testing.T) {
	s := newPinScreen(newMockServices(), NewStyles(true), true)
	s.step = pinSelect
	items := s.FullHelpItems()
	if !hasHelpItem(items, "x/space", "toggle") {
		t.Errorf("pinSelect help should include {x/space, toggle}; got %v", items)
	}
	if hasHelpItem(items, "enter", "select") {
		t.Errorf("pinSelect (MultiSelect) help must not include {enter, select}; got %v", items)
	}
}

// TestHelpSteps_ConfirmSteps_ShowYesNo checks every huh Confirm step advertises
// "h/l yes/no" (never "j/k" as the toggle for confirm).
func TestHelpSteps_ConfirmSteps_ShowYesNo(t *testing.T) {
	svc := newMockServices()
	styles := NewStyles(true)

	pin := newPinScreen(svc, styles, true)
	pin.step = pinConfirm
	if !hasHelpItem(pin.FullHelpItems(), "h/l", "yes/no") {
		t.Errorf("pinConfirm help should include {h/l, yes/no}; got %v", pin.FullHelpItems())
	}

	doc := newDoctorScreen(svc, styles, true)
	doc.step = doctorConfirm
	if !hasHelpItem(doc.FullHelpItems(), "h/l", "yes/no") {
		t.Errorf("doctorConfirm help should include {h/l, yes/no}; got %v", doc.FullHelpItems())
	}
	if !hasHelpItem(doc.FullHelpItems(), "j/k", "scroll") {
		t.Errorf("doctorConfirm help should include {j/k, scroll}; got %v", doc.FullHelpItems())
	}

	src := newSourceScreen(svc, styles, true)
	src.step = sourceRemoveConfirm
	if !hasHelpItem(src.FullHelpItems(), "h/l", "yes/no") {
		t.Errorf("sourceRemoveConfirm help should include {h/l, yes/no}; got %v", src.FullHelpItems())
	}
}

// TestHelpSteps_TextInput_NoEnterSelect checks that text-input steps submit rather
// than "select", so they never advertise the menu-style "enter select".
func TestHelpSteps_TextInput_NoEnterSelect(t *testing.T) {
	svc := newMockServices()
	styles := NewStyles(true)

	src := newSourceScreen(svc, styles, true)
	for _, step := range []sourceStep{sourceAddLocalInput, sourceAddGitInput} {
		src.step = step
		items := src.FullHelpItems()
		if hasHelpItem(items, "enter", "select") {
			t.Errorf("source input step %d must not include {enter, select}; got %v", step, items)
		}
		if !hasHelpItem(items, "enter", "submit") {
			t.Errorf("source input step %d should include {enter, submit}; got %v", step, items)
		}
	}

	snap := newSnapshotScreen(svc, styles, true)
	snap.step = snapshotSaveName
	if hasHelpItem(snap.FullHelpItems(), "enter", "select") {
		t.Errorf("snapshotSaveName (input) must not include {enter, select}; got %v", snap.FullHelpItems())
	}

	prof := newProfileScreen(svc, styles, true)
	prof.step = profileCreateName
	if hasHelpItem(prof.FullHelpItems(), "enter", "select") {
		t.Errorf("profileCreateName (input) must not include {enter, select}; got %v", prof.FullHelpItems())
	}
}

// TestHelpSteps_SnapshotRestoreSelect_ConfirmPhase checks the two-phase restore step:
// a huh Select until confirmForm is built, then a huh Confirm.
func TestHelpSteps_SnapshotRestoreSelect_ConfirmPhase(t *testing.T) {
	s := newSnapshotScreen(newMockServices(), NewStyles(true), true)
	s.step = snapshotRestoreSelect

	// Select phase: confirmForm nil -> navigation/select help.
	if !hasHelpItem(s.FullHelpItems(), "enter", "select") {
		t.Errorf("restoreSelect (select phase) should include {enter, select}; got %v", s.FullHelpItems())
	}

	// Confirm phase: confirmForm set -> yes/no help, never "enter select".
	s.confirmForm = huh.NewForm(huh.NewGroup(huh.NewConfirm().Value(&s.confirmed)))
	items := s.FullHelpItems()
	if !hasHelpItem(items, "h/l", "yes/no") {
		t.Errorf("restoreSelect (confirm phase) should include {h/l, yes/no}; got %v", items)
	}
	if hasHelpItem(items, "enter", "select") {
		t.Errorf("restoreSelect (confirm phase) must not include {enter, select}; got %v", items)
	}
}

// TestHelpSteps_MainMenu_RootHasNoBack checks the root main menu advertises "? help"
// and omits any "esc" hint (there is no screen to go back to).
func TestHelpSteps_MainMenu_RootHasNoBack(t *testing.T) {
	m := newMainMenuScreen(newMockServices(), NewStyles(true), true)
	items := m.FullHelpItems()
	if hasHelpKey(items, "esc") {
		t.Errorf("main menu (root) must not include an 'esc' hint; got %v", items)
	}
	if !hasHelpItem(items, "?", "help") {
		t.Errorf("main menu help should include {?, help}; got %v", items)
	}
}
