//go:build !release

package storage

import "testing"

// A build run from an IDE kept its settings and collections in the release
// build's directory, so the two overwrote each other's data.
func TestTheDefaultBuildHasItsOwnDevDirectory(t *testing.T) {
	if Name != "gascoigne-workshop-dev" {
		t.Fatalf("Name = %q, want %q", Name, "gascoigne-workshop-dev")
	}
}
