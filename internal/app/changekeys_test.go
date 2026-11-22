package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// An idle frame (no input, no message, no window change) must not build
// the JSON form of the collection: that costs time in proportion to its
// size, every frame, for ever.
func TestIdleFramesDoNotRebuildTheChangeKeys(t *testing.T) {
	a := testApp(t)
	a.setWorkspace(t.TempDir())
	var calls int
	a.keys.collectionOf = func(col request.Collection) string { calls++; return tree.Key(col) }
	for range 30 {
		a.Step(core.Input{})
	}
	if calls > 1 {
		t.Fatalf("%d key builds over 30 idle frames, want at most one", calls)
	}
	before := calls
	a.Step(core.Input{X: 10, Y: 10, Click: true})
	if calls == before {
		t.Fatal("a frame with input must check the collection")
	}
}

// The dirty mark of a tab compares its request with the saved copy, which
// clones and walks the request: not for every tab on every frame.
func TestIdleFramesDoNotRecompareTheTabsWithTheirSavedCopies(t *testing.T) {
	a := testApp(t)
	var calls int
	a.keys.dirtyOf = func(tab *reqTab) bool { calls++; return dirty(tab) }
	a.Step(core.Input{})
	before := calls
	for range 30 {
		a.Step(core.Input{})
	}
	if calls != before {
		t.Fatalf("%d comparisons over 30 idle frames", calls-before)
	}
	a.Step(core.Input{X: 10, Y: 10, Click: true})
	if calls == before {
		t.Fatal("a frame with input must compare again")
	}
}
