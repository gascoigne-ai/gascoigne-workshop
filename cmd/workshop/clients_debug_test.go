//go:build !release

package main

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
)

// A debug build is what runs from an IDE: its first start shows the
// example server, and no config file can hide it. What it remembers
// afterwards lives in its own -dev directory (storage.Name).
func TestDebugBuildStartsWithTheExampleServer(t *testing.T) {
	if startWith != collection.StartWithSample {
		t.Error("a debug build starts with the sample")
	}
}
