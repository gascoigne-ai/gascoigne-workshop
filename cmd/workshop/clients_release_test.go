//go:build release

package main

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
)

// A release build ships no example endpoints and reads no .workshop file: a
// first start shows no collection, only what the user opened before. It
// used to start with an empty unsaved collection or the client config's
// tree.
func TestReleaseBuildStartsWithNothing(t *testing.T) {
	if startWith != collection.StartWithNothing {
		t.Errorf("startWith = %v, want StartWithNothing", startWith)
	}
	sample := collection.Sample()
	if len(sample.Folders) != 0 || len(sample.Requests) != 0 {
		t.Errorf("release sample has %d folders and %d requests", len(sample.Folders), len(sample.Requests))
	}
}
