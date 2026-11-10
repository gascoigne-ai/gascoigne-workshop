package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// changeKeys caches the comparable forms (JSON) of the collection and of
// the state to save. Building them costs time in proportion to the size of
// the collection, so the per-frame checks (the autosave timers, the window
// title) reuse them and rebuild only on a frame where input, a message or
// a window change could have changed something. Code that saves, loads or
// compares right after changing things builds a fresh one.
type changeKeys struct {
	collection, state string
	stale             bool // a rebuild is due: something may have changed since the last one
	built             bool
	collectionOf      func(model.Collection) string // tree.Key; a field so a test can count the builds
	geometry          [4]int                        // window size and position at the last frame
}

// markStale says this frame may have changed the collection or the state.
func (a *App) markStale() { a.keys.stale = true }

// collectionKey is the comparable form of the collection, as of the last
// frame that could have changed it.
func (a *App) collectionKey() string {
	if a.keys.stale || !a.keys.built {
		a.freshKeys()
	}
	return a.keys.collection
}

// stateNow is the comparable form of the state to save, as of the last
// frame that could have changed it.
func (a *App) stateNow() string {
	if a.keys.stale || !a.keys.built {
		a.freshKeys()
	}
	return a.keys.state
}

// freshKeys rebuilds both forms now.
func (a *App) freshKeys() {
	build := a.keys.collectionOf
	if build == nil {
		build = tree.Key
	}
	a.keys.collection = build(a.col)
	a.keys.state = stateKey(a.snapshot())
	a.keys.stale, a.keys.built = false, true
}

// collectionKeyNow is a key built right now, for code that has just
// changed the collection and compares or records it at once.
func (a *App) collectionKeyNow() string {
	a.freshKeys()
	return a.keys.collection
}

// noteGeometry marks the keys stale when the window moved or was resized,
// which the saved state records and no input reports.
func (a *App) noteGeometry() {
	geometry := [4]int{a.w, a.h, a.winX, a.winY}
	if geometry != a.keys.geometry {
		a.keys.geometry = geometry
		a.markStale()
	}
}
