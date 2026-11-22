//go:build !release

package main

import "testing"

// A debug build is what runs from an IDE: its first start shows the
// example server, and no config file can hide it. What it remembers
// afterwards lives in its own -dev directory (storage.Name).
func TestDebugBuildReadsNoClientConfig(t *testing.T) {
	if clientConfigs() != nil {
		t.Error("a debug build reads no client config")
	}
}
