package autosave

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
)

// fakeStore keeps the states it was asked to save, in order.
type fakeStore struct {
	mu      sync.Mutex
	loaded  state.State
	loadErr error
	saveErr error
	saved   []state.State
	paths   []string
}

func (store *fakeStore) Load(string) (state.State, error) { return store.loaded, store.loadErr }

func (store *fakeStore) Save(path string, current state.State) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.saved = append(store.saved, current)
	store.paths = append(store.paths, path)
	return store.saveErr
}

func (store *fakeStore) saves() []state.State {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]state.State(nil), store.saved...)
}

// fakeWindow is a window whose state is a value the test sets.
type fakeWindow struct {
	current  state.State
	restored []state.State
}

func (window *fakeWindow) Snapshot() state.State { return window.current }
func (window *fakeWindow) Restore(s state.State) { window.restored = append(window.restored, s) }

// fakeReports counts what was reported.
type fakeReports struct {
	failed   []error
	readOnly []error
}

func (reports *fakeReports) Failed(err error)   { reports.failed = append(reports.failed, err) }
func (reports *fakeReports) ReadOnly(err error) { reports.readOnly = append(reports.readOnly, err) }

type stateWorld struct {
	file    *StateFile
	store   *fakeStore
	window  *fakeWindow
	reports *fakeReports
	drain   func() bool
}

func newStateWorld(path string) *stateWorld {
	queue := bus.New()
	drain := func() bool {
		return queue.Drain(func(msg bus.Msg) {
			if run, ok := msg.(func()); ok {
				run()
			}
		})
	}
	world := &stateWorld{store: &fakeStore{}, window: &fakeWindow{}, reports: &fakeReports{}, drain: drain}
	world.file = NewStateFile(world.store, path, world.window, world.reports, serial.NewWriter(queue, drain))
	return world
}

// settle delivers the reports of the background writes.
func (world *stateWorld) settle(t *testing.T, saves int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		world.drain()
		if len(world.store.saves()) >= saves {
			world.drain()
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("saved %d times, want %d", len(world.store.saves()), saves)
}

func TestWithoutAPathNothingIsWritten(t *testing.T) {
	world := newStateWorld("")
	if err := world.file.Load(); err != nil {
		t.Fatal(err)
	}
	world.file.Begin()
	world.window.current.SidebarW = 300
	world.file.Tick(world.file.Key(), start)
	world.file.Tick(world.file.Key(), start.Add(time.Hour))
	if err := world.file.Save(); err != nil {
		t.Fatal(err)
	}
	world.drain()
	if got := world.store.saves(); len(got) != 0 {
		t.Fatalf("saved %+v without a path", got)
	}
	if world.file.Writable() || world.file.HistoryDir() != "" {
		t.Fatalf("writable %v, history dir %q: want neither without a path", world.file.Writable(), world.file.HistoryDir())
	}
}

func TestAFileANewerBuildWroteIsReadOnlyForTheSession(t *testing.T) {
	world := newStateWorld("/cfg/state.json")
	newer := &storage.NewerError{Found: 9, Newest: 1}
	world.store.loadErr = newer
	if err := world.file.Load(); err != newer {
		t.Fatalf("Load = %v, want the load error", err)
	}
	world.file.Begin()
	world.window.current.SidebarW = 300
	world.file.Tick(world.file.Key(), start)
	world.file.Tick(world.file.Key(), start.Add(time.Hour))
	if err := world.file.Save(); err != nil {
		t.Fatal(err)
	}
	world.drain()
	if len(world.store.saves()) != 0 || world.file.Writable() {
		t.Fatalf("a newer file was written over: %+v", world.store.saves())
	}
	if len(world.reports.readOnly) != 1 || world.reports.readOnly[0] != newer {
		t.Fatalf("read-only reports %v, want the load error once", world.reports.readOnly)
	}
}

func TestTickWritesOnceTheStateStayedForThePause(t *testing.T) {
	world := newStateWorld("/cfg/state.json")
	_ = world.file.Load()
	world.file.Begin()
	world.window.current.SidebarW = 300
	key := world.file.Key()
	world.file.Tick(key, start)
	world.file.Tick(key, start.Add(StateDebounce-time.Millisecond))
	if got := len(world.store.saves()); got != 0 {
		t.Fatalf("saved %d times before the pause passed", got)
	}
	world.file.Tick(key, start.Add(StateDebounce))
	world.settle(t, 1)
	world.file.Tick(key, start.Add(time.Hour))
	world.drain()
	got := world.store.saves()
	if len(got) != 1 || got[0].SidebarW != 300 || world.store.paths[0] != "/cfg/state.json" {
		t.Fatalf("saved %+v to %v, want the changed state once", got, world.store.paths)
	}
}

func TestAFailedWriteIsReportedOnceAndNotRetried(t *testing.T) {
	world := newStateWorld("/cfg/state.json")
	_ = world.file.Load()
	world.file.Begin()
	world.store.saveErr = errors.New("disk full")
	world.window.current.SidebarW = 300
	key := world.file.Key()
	world.file.Tick(key, start)
	world.file.Tick(key, start.Add(StateDebounce))
	world.settle(t, 1)
	world.file.Tick(key, start.Add(time.Hour))
	world.drain()
	if len(world.store.saves()) != 1 || len(world.reports.failed) != 1 || world.reports.failed[0] != world.store.saveErr {
		t.Fatalf("saves %d, failures %v: want one try and one report", len(world.store.saves()), world.reports.failed)
	}
}

func TestSaveWaitsForWritesThenWritesNow(t *testing.T) {
	world := newStateWorld("/cfg/state.json")
	_ = world.file.Load()
	world.file.Begin()
	world.window.current.SidebarW = 300
	key := world.file.Key()
	world.file.Tick(key, start)
	world.file.Tick(key, start.Add(StateDebounce)) // a background write starts
	world.window.current.SidebarW = 400
	if err := world.file.Save(); err != nil {
		t.Fatal(err)
	}
	got := world.store.saves()
	if len(got) != 2 || got[0].SidebarW != 300 || got[1].SidebarW != 400 {
		t.Fatalf("saved %+v, want the background write then the one at exit", got)
	}
	world.file.Tick(world.file.Key(), start.Add(time.Hour))
	world.drain()
	if len(world.store.saves()) != 2 {
		t.Fatal("the state written at exit is written again")
	}
}

func TestLoadRestoresTheWindowAndKeepsWelcomedAndTrusted(t *testing.T) {
	world := newStateWorld("/cfg/state.json")
	allowed := request.Access{Kind: request.AccessCommand, Subject: "/bin/true --flag"}
	world.store.loaded = state.State{
		Window:   state.Window{W: 900, H: 600, X: 10, Y: 20},
		Welcomed: "v1.2",
		Trusted:  []string{allowed.Key()},
	}
	if err := world.file.Load(); err != nil {
		t.Fatal(err)
	}
	if len(world.window.restored) != 1 || world.window.restored[0].Welcomed != "v1.2" {
		t.Fatalf("restored %+v, want the stored state once", world.window.restored)
	}
	if world.file.Welcomed() != "v1.2" || !world.file.Trusts(allowed) {
		t.Fatalf("welcomed %q, trusts %v", world.file.Welcomed(), world.file.Trusts(allowed))
	}
	if got := world.file.Restored(1920, 1080); got != (state.Window{W: 900, H: 600, X: 10, Y: 20}) {
		t.Fatalf("Restored = %+v", got)
	}
	world.file.Welcome("v1.3")
	other := request.Access{Kind: request.AccessFile, Subject: "/srv/body.json"}
	world.file.Trust(other)
	saved := world.file.Snapshot()
	if saved.Welcomed != "v1.3" || len(saved.Trusted) != 2 {
		t.Fatalf("snapshot welcomed %q, trusted %v: want both kept beside the window's part", saved.Welcomed, saved.Trusted)
	}
}

func TestTheFirstSnapshotIsNotPending(t *testing.T) {
	world := newStateWorld("/cfg/state.json")
	world.window.current.SidebarW = 250
	_ = world.file.Load()
	world.file.Begin()
	world.file.Tick(world.file.Key(), start)
	world.file.Tick(world.file.Key(), start.Add(time.Hour))
	world.drain()
	if got := world.store.saves(); len(got) != 0 {
		t.Fatalf("saved %+v: the state as it started is already on disk", got)
	}
}

func TestHistoryDirSitsBesideTheStateFile(t *testing.T) {
	path := filepath.Join("cfg", "state.json")
	world := newStateWorld(path)
	if got, want := world.file.HistoryDir(), filepath.Join("cfg", "history"); got != want {
		t.Fatalf("HistoryDir = %q, want %q", got, want)
	}
}
