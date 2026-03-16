package nd_test

import (
	"testing"

	"github.com/armstrongl/nd/internal/nd"
)

func TestOriginProfile(t *testing.T) {
	o := nd.OriginProfile("go-backend")
	if o != "profile:go-backend" {
		t.Errorf("got %q", o)
	}
	if !o.IsProfile() {
		t.Error("should be a profile origin")
	}
	if o.ProfileName() != "go-backend" {
		t.Errorf("got %q", o.ProfileName())
	}
}

func TestOriginManualIsNotProfile(t *testing.T) {
	if nd.OriginManual.IsProfile() {
		t.Error("manual should not be a profile origin")
	}
	if nd.OriginManual.ProfileName() != "" {
		t.Error("manual profile name should be empty")
	}
}
