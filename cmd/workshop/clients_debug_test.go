//go:build !release

package main

import "testing"

// A debug build is what runs from an IDE: it always starts from the
// example server, whatever an earlier run saved or a config says.
func TestDebugBuildStartsFromTheSampleOnly(t *testing.T) {
	if clientConfigs() != nil {
		t.Error("a debug build reads no client config")
	}
	if !ignoreRememberedWorkspace {
		t.Error("a debug build must not reopen a remembered workspace")
	}
}
