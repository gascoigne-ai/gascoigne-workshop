package ui

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// An idle frame (no input, no message, no window change) must not build
// the JSON form of the collection: that costs time in proportion to its
// size, every frame, for ever.
func TestIdleFramesDoNotRebuildTheChangeKeys(t *testing.T) {
	a, _ := newTestApp(t)
	a.workspace = t.TempDir()
	var calls int
	a.keys.collectionOf = func(col model.Collection) string { calls++; return tree.Key(col) }
	for range 30 {
		a.step(core.Input{})
	}
	if calls > 1 {
		t.Fatalf("%d key builds over 30 idle frames, want at most one", calls)
	}
	before := calls
	a.step(core.Input{X: 10, Y: 10, Click: true})
	if calls == before {
		t.Fatal("a frame with input must check the collection")
	}
}
