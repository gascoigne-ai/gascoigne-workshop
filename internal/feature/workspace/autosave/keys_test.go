package autosave

import (
	"fmt"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// fakeSources answers from a name list and counts every build.
type fakeSources struct {
	cols        []*request.Collection
	generation  int // part of every key, so a rebuild is visible in the result
	keyBuilds   int
	stateBuilds int
	compares    int
}

func (sources *fakeSources) Collections() []*request.Collection { return sources.cols }

func (sources *fakeSources) CollectionKey(col request.Collection) string {
	sources.keyBuilds++
	return fmt.Sprintf("%s#%d", col.Name, sources.generation)
}

func (sources *fakeSources) StateKey() string {
	sources.stateBuilds++
	return fmt.Sprintf("state#%d", sources.generation)
}

func (sources *fakeSources) Dirty(*tabs.Tab) bool {
	sources.compares++
	return sources.generation%2 == 1
}

func named(name string) *request.Collection {
	col := &request.Collection{}
	col.Name = name
	return col
}

func newFake() (*fakeSources, *request.Collection) {
	col := named("one")
	return &fakeSources{cols: []*request.Collection{col}}, col
}

func TestKeysAreNotRebuiltUntilMarkedStale(t *testing.T) {
	sources, col := newFake()
	keys := NewKeys(sources)
	first := keys.Collection(col)
	sources.generation = 1 // a change nobody reported
	if got := keys.Collection(col); got != first {
		t.Fatalf("key = %q, want the cached %q", got, first)
	}
	keys.MarkStale()
	if got := keys.Collection(col); got == first {
		t.Fatalf("key = %q after MarkStale, want a rebuilt one", got)
	}
}

func TestAStaleCacheRebuildsOnceOnTheNextRead(t *testing.T) {
	sources, col := newFake()
	keys := NewKeys(sources)
	keys.MarkStale()
	keys.Collection(col)
	keys.State()
	keys.Collection(col)
	if sources.keyBuilds != 1 || sources.stateBuilds != 1 {
		t.Fatalf("builds: key %d, state %d, want 1 and 1", sources.keyBuilds, sources.stateBuilds)
	}
}

func TestCollectionNowAlwaysRebuilds(t *testing.T) {
	sources, col := newFake()
	keys := NewKeys(sources)
	keys.Collection(col)
	sources.generation = 1
	if got := keys.CollectionNow(col); got != "one#1" {
		t.Fatalf("key = %q, want one#1", got)
	}
	if got := keys.Collection(col); got != "one#1" {
		t.Fatalf("the cache holds %q after a fresh build, want one#1", got)
	}
}

func TestACollectionOpenedSinceTheLastBuildGetsAFreshKey(t *testing.T) {
	sources, col := newFake()
	keys := NewKeys(sources)
	keys.Collection(col)
	opened := named("two")
	sources.cols = append(sources.cols, opened)
	if got := keys.Collection(opened); got != "two#0" {
		t.Fatalf("key = %q, want two#0", got)
	}
}

func TestTabDirtyComparesOncePerBuild(t *testing.T) {
	sources, _ := newFake()
	keys := NewKeys(sources)
	tab := &tabs.Tab{}
	for range 5 {
		keys.TabDirty(tab)
	}
	if sources.compares != 1 {
		t.Fatalf("%d comparisons over 5 reads, want 1", sources.compares)
	}
	keys.MarkStale()
	keys.TabDirty(tab)
	if sources.compares != 2 {
		t.Fatalf("%d comparisons after MarkStale, want 2", sources.compares)
	}
}

func TestAMovedOrResizedWindowMarksTheKeysStale(t *testing.T) {
	sources, col := newFake()
	keys := NewKeys(sources)
	geometry := [4]int{800, 600, 10, 20}
	keys.NoteGeometry(geometry)
	keys.Collection(col)
	builds := sources.stateBuilds
	keys.NoteGeometry(geometry)
	keys.State()
	if sources.stateBuilds != builds {
		t.Fatal("the same geometry must not mark the keys stale")
	}
	keys.NoteGeometry([4]int{900, 600, 10, 20})
	keys.State()
	if sources.stateBuilds == builds {
		t.Fatal("a resized window must mark the keys stale")
	}
}
