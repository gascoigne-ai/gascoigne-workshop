package autosave

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Sources builds the comparable forms Keys caches; the app answers it.
type Sources interface {
	// Collections are the open collections, whose keys a rebuild takes.
	Collections() []*request.Collection
	// CollectionKey is the comparable form of one collection.
	CollectionKey(request.Collection) string
	// StateKey is the comparable form of the state to save.
	StateKey() string
	// Dirty reports whether the tab's request differs from its saved copy.
	Dirty(*tabs.Tab) bool
}

// Keys caches the comparable forms (JSON) of the collections and of the
// state to save. Building them costs time in proportion to the size of the
// collection, so the per-frame checks (the autosave timers, the window
// title) reuse them and rebuild only on a frame where input, a message or
// a window change could have changed something. Code that saves, loads or
// compares right after changing things builds a fresh one.
//
// The cache is read only when it is not stale and was built once; a read
// while stale rebuilds everything first. The dirty marks are keyed by the
// tab, one per open tab, and live as long as a build. Rejected: keying by
// the collection entry, or a field on the tab, which would put app state
// into the tab model. Revisit when a rebuild shows up in a profile or tabs
// get stable ids.
type Keys struct {
	sources     Sources
	collections map[*request.Collection]string // by open collection
	state       string
	stale       bool // a rebuild is due: something may have changed since the last one
	built       bool
	dirty       map[*tabs.Tab]bool // which open tabs differ from their saved copy
	geometry    [4]int             // window size and position at the last frame
}

// NewKeys is a cache over sources; the first read builds it.
func NewKeys(sources Sources) *Keys { return &Keys{sources: sources} }

// MarkStale says this frame may have changed the collection or the state.
func (keys *Keys) MarkStale() { keys.stale = true }

// Collection is the comparable form of col, as of the last frame that
// could have changed it.
func (keys *Keys) Collection(col *request.Collection) string {
	keys.refresh()
	if key, ok := keys.collections[col]; ok {
		return key
	}
	return keys.CollectionNow(col) // opened since the last build
}

// State is the comparable form of the state to save, as of the last
// frame that could have changed it.
func (keys *Keys) State() string {
	keys.refresh()
	return keys.state
}

// TabDirty reports whether the tab's request differs from its saved copy,
// as of the last frame that could have changed it.
func (keys *Keys) TabDirty(tab *tabs.Tab) bool {
	keys.refresh()
	if marked, ok := keys.dirty[tab]; ok {
		return marked
	}
	keys.dirty[tab] = keys.sources.Dirty(tab)
	return keys.dirty[tab]
}

// CollectionNow is a key of col built right now, for code that has
// just changed it and compares or records it at once.
func (keys *Keys) CollectionNow(col *request.Collection) string {
	keys.rebuild()
	if key, ok := keys.collections[col]; ok {
		return key
	}
	return keys.sources.CollectionKey(*col) // not open (yet): nothing caches it
}

// NoteGeometry marks the keys stale when the window moved or was resized
// (width, height, x, y), which the saved state records and no input reports.
func (keys *Keys) NoteGeometry(geometry [4]int) {
	if geometry != keys.geometry {
		keys.geometry = geometry
		keys.MarkStale()
	}
}

func (keys *Keys) refresh() {
	if keys.stale || !keys.built {
		keys.rebuild()
	}
}

// rebuild builds both forms now and forgets the dirty marks.
func (keys *Keys) rebuild() {
	keys.collections = map[*request.Collection]string{}
	for _, col := range keys.sources.Collections() {
		keys.collections[col] = keys.sources.CollectionKey(*col)
	}
	keys.state = keys.sources.StateKey()
	keys.dirty = map[*tabs.Tab]bool{}
	keys.stale, keys.built = false, true
}
