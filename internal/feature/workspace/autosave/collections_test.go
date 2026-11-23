package autosave

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
)

// fakeSaver keeps what it was given and, when gate is set, waits for it.
type fakeSaver struct {
	gate  chan struct{}
	err   error
	mu    sync.Mutex
	saved []savedCollection
}

type savedCollection struct {
	dir  string
	name string
}

func (saver *fakeSaver) Save(dir string, col request.Collection) error {
	if saver.gate != nil {
		<-saver.gate
	}
	saver.mu.Lock()
	defer saver.mu.Unlock()
	saver.saved = append(saver.saved, savedCollection{dir, col.Name})
	return saver.err
}

func (saver *fakeSaver) calls() []savedCollection {
	saver.mu.Lock()
	defer saver.mu.Unlock()
	return append([]savedCollection(nil), saver.saved...)
}

// newSaves is Collections over saver, with the drain the UI loop would run.
func newSaves(saver Saver) (*Collections, func() bool) {
	queue := bus.New()
	drain := func() bool {
		return queue.Drain(func(msg bus.Msg) {
			if run, ok := msg.(func()); ok {
				run()
			}
		})
	}
	return NewCollections(saver, serial.NewWriter(queue, drain)), drain
}

func TestACollectionIsDueOnceItsKeyStayedForThePause(t *testing.T) {
	saves, _ := newSaves(&fakeSaver{})
	col := named("one")
	saves.Saved(col, "a")
	if saves.Due(col, "b", start) {
		t.Fatal("a change starts the pause, it is not due yet")
	}
	if saves.Due(col, "b", start.Add(SaveDebounce-time.Millisecond)) {
		t.Fatal("the pause has not passed")
	}
	if !saves.Due(col, "b", start.Add(SaveDebounce)) {
		t.Fatal("unchanged for the pause: due")
	}
}

func TestASavedCollectionIsNotPending(t *testing.T) {
	saves, _ := newSaves(&fakeSaver{})
	col := named("one")
	saves.Saved(col, "a")
	if saves.Pending(col, "a") {
		t.Fatal("the key written is not pending")
	}
	if !saves.Pending(col, "b") {
		t.Fatal("a key that differs from the one written is pending")
	}
}

func TestForgetDropsTheDebounce(t *testing.T) {
	saves, _ := newSaves(&fakeSaver{})
	col := named("one")
	saves.Saved(col, "a")
	saves.Forget(col)
	if !saves.Pending(col, "a") {
		t.Fatal("a forgotten collection starts again with nothing written")
	}
}

func TestWriteSavesACopyTakenAtTheCall(t *testing.T) {
	saver := &fakeSaver{gate: make(chan struct{})}
	saves, drain := newSaves(saver)
	col := named("before")
	var gotName string
	var gotErr error
	reported := false
	saves.Write("/w/one", col, func(name string, err error) { gotName, gotErr, reported = name, err, true })
	col.Name = "after" // the UI goes on editing while the write waits
	close(saver.gate)
	deadline := time.Now().Add(2 * time.Second)
	for !reported && time.Now().Before(deadline) {
		drain()
		time.Sleep(time.Millisecond)
	}
	got := saver.calls()
	if len(got) != 1 || got[0] != (savedCollection{"/w/one", "before"}) {
		t.Fatalf("saved %+v, want the collection as it was at the call", got)
	}
	if !reported || gotName != "before" || gotErr != nil {
		t.Fatalf("done got %q, %v (reported %v)", gotName, gotErr, reported)
	}
}

func TestWriteNowSavesAtOnce(t *testing.T) {
	saver := &fakeSaver{}
	saves, _ := newSaves(saver)
	col := named("one")
	if err := saves.WriteNow("/w/one", col, "a"); err != nil {
		t.Fatal(err)
	}
	if got := saver.calls(); len(got) != 1 || got[0].dir != "/w/one" {
		t.Fatalf("saved %+v, want one save before WriteNow returned", got)
	}
	if saves.Pending(col, "a") {
		t.Fatal("a collection written now is not pending")
	}

	failing := &fakeSaver{err: errors.New("disk full")}
	saves, _ = newSaves(failing)
	if err := saves.WriteNow("/w/one", col, "a"); err == nil {
		t.Fatal("the failure is returned")
	}
	if !saves.Pending(col, "a") {
		t.Fatal("a failed write leaves the collection pending")
	}
}
