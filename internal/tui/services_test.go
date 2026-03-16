package tui_test

import (
	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/tui"
)

// Compile-time interface assertions.
// These verify that deploy.Engine satisfies tui.Deployer.
var _ tui.Deployer = (*deploy.Engine)(nil)

// profileAdapter is tested in profileadapter_test.go; its compile-time
// assertion lives there since the adapter is unexported.
