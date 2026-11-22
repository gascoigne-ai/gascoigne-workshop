//go:build release

package storage

import "testing"

// The release build keeps the directory installed users already have.
func TestTheReleaseBuildKeepsTheInstalledDirectory(t *testing.T) {
	if Name != "gascoigne-workshop" {
		t.Fatalf("Name = %q, want %q", Name, "gascoigne-workshop")
	}
}
