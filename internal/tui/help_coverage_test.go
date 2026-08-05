package tui

import "testing"

// TestScreens_ImplementHelpInterface asserts every screen constructor's result
// satisfies HelpProvider or FullHelpProvider, so pressing '?' always yields a
// populated help overlay. This enforces the "every registered screen provides
// help" acceptance criterion.
func TestScreens_ImplementHelpInterface(t *testing.T) {
	svc := newMockServices()
	styles := NewStyles(true)

	constructors := map[string]func() Screen{
		"mainMenu": func() Screen { return newMainMenuScreen(svc, styles, true) },
		"deploy":   func() Screen { return newDeployScreen(svc, styles, true) },
		"browse":   func() Screen { return newBrowseScreen(svc, styles, true) },
		"pin":      func() Screen { return newPinScreen(svc, styles, true) },
		"remove":   func() Screen { return newRemoveScreen(svc, styles, true) },
		"profile":  func() Screen { return newProfileScreen(svc, styles, true) },
		"settings": func() Screen { return newSettingsScreen(svc, styles, true) },
		"snapshot": func() Screen { return newSnapshotScreen(svc, styles, true) },
		"scope":    func() Screen { return newScopeScreen(svc, styles, true) },
		"status":   func() Screen { return newStatusScreen(svc, styles, true) },
		"doctor":   func() Screen { return newDoctorScreen(svc, styles, true) },
		"source":   func() Screen { return newSourceScreen(svc, styles, true) },
		"firstRun": func() Screen { return newFirstRunScreen(svc, styles, true) },
	}

	for name, ctor := range constructors {
		t.Run(name, func(t *testing.T) {
			screen := ctor()
			_, isHelp := screen.(HelpProvider)
			_, isFull := screen.(FullHelpProvider)
			if !isHelp && !isFull {
				t.Errorf("%s screen implements neither HelpProvider nor FullHelpProvider", name)
			}

			// Whatever the overlay derives must be non-empty.
			if items := defaultHelp(screen); len(items) == 0 {
				t.Errorf("%s screen produced empty defaultHelp", name)
			}
		})
	}
}
