package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// changeKeys caches the comparable forms (JSON) of the collections and of
// the state to save. Building them costs time in proportion to the size of
// the collection, so the per-frame checks (the autosave timers, the window
// title) reuse them and rebuild only on a frame where input, a message or
// a window change could have changed something. Code that saves, loads or
// compares right after changing things builds a fresh one.
type changeKeys struct {
	collections  map[*request.Collection]string // by open collection
	state        string
	stale        bool // a rebuild is due: something may have changed since the last one
	built        bool
	collectionOf func(request.Collection) string // tree.Key; a field so a test can count the builds
	dirtyOf      func(*reqTab) bool              // dirty; a field so a test can count the comparisons
	dirty        map[*reqTab]bool                // which open tabs differ from their saved copy
	geometry     [4]int                          // window size and position at the last frame
}

// markStale says this frame may have changed the collection or the state.
func (a *App) markStale() { a.keys.stale = true }

// collectionKey is the comparable form of col, as of the last frame that
// could have changed it.
func (a *App) collectionKey(col *request.Collection) string {
	if a.keys.stale || !a.keys.built {
		a.freshKeys()
	}
	if key, ok := a.keys.collections[col]; ok {
		return key
	}
	return a.collectionKeyNow(col) // opened since the last build
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
	a.keys.collections = map[*request.Collection]string{}
	for _, entry := range a.open.Entries {
		a.keys.collections[entry.Collection] = build(*entry.Collection)
	}
	a.keys.state = stateKey(a.snapshot())
	a.keys.dirty = map[*reqTab]bool{}
	a.keys.stale, a.keys.built = false, true
}

// tabDirty reports whether the tab's request differs from its saved copy,
// as of the last frame that could have changed it.
func (a *App) tabDirty(tab *reqTab) bool {
	if a.keys.stale || !a.keys.built {
		a.freshKeys()
	}
	if marked, ok := a.keys.dirty[tab]; ok {
		return marked
	}
	compare := a.keys.dirtyOf
	if compare == nil {
		compare = dirty
	}
	a.keys.dirty[tab] = compare(tab)
	return a.keys.dirty[tab]
}

// collectionKeyNow is a key of col built right now, for code that has
// just changed it and compares or records it at once.
func (a *App) collectionKeyNow(col *request.Collection) string {
	a.freshKeys()
	if key, ok := a.keys.collections[col]; ok {
		return key
	}
	return tree.Key(*col) // not open (yet): nothing caches it
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
