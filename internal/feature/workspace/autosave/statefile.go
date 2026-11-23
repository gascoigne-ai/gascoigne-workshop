package autosave

import (
	"path/filepath"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
)

// StateDebounce is how long the remembered state must stay unchanged
// before it is written, so a drag does not write on every frame.
const StateDebounce = time.Second

// Window is the window's own part of the remembered state.
type Window interface {
	Snapshot() state.State
	Restore(state.State)
}

// Reports tells the user what went wrong with the state file.
type Reports interface {
	Failed(err error)
	ReadOnly(err error)
}

// StateFile keeps what is remembered between runs (the window's part, the
// trusted accesses and the welcomed version) in one file: it reads it once,
// writes it in the background once it stayed unchanged for StateDebounce
// and, at exit, at once. It writes nothing when it has no path or a newer
// build wrote the file.
//
// Why the trusted set and the welcomed version live here and not in the
// window: they are written with the window's part but nothing in the window
// reads them, so the window's Snapshot stays the UI's alone and this file
// adds the rest. Rejected: the App holding them and passing them in at each
// save, which keeps a copy of a fact the file owns; revisit if a second
// reader of the trusted set needs it before a save.
type StateFile struct {
	store    workspace.StateStore
	path     string
	window   Window
	reports  Reports
	writes   *serial.Writer
	access   storage.Access // read-only when a newer build wrote the file
	save     Debounce       // when the state was last written
	initial  state.Window   // the window geometry of the last run
	welcomed string         // the version whose welcome dialog was shown
	trusted  request.AccessSet
}

// NewStateFile keeps the state at path ("" keeps none) in store, writing in
// the background on writes. Load it, then Begin it once the window is built.
func NewStateFile(store workspace.StateStore, path string, window Window, reports Reports, writes *serial.Writer) *StateFile {
	return &StateFile{store: store, path: path, window: window, reports: reports, writes: writes}
}

// Load reads the file and applies it to the window; the error is the
// reason it could not be read, if any. A file a newer build wrote is
// reported through Reports.ReadOnly and left as it is.
func (file *StateFile) Load() error {
	stored, err := file.store.Load(file.path)
	file.access = storage.AccessAfter(err)
	if file.access == storage.ReadOnly {
		file.reports.ReadOnly(err)
	}
	file.initial = stored.Window
	file.trusted = request.AccessSetOf(stored.Trusted)
	file.window.Restore(stored)
	file.welcomed = stored.Welcomed
	return err
}

// Begin takes the state as it stands as the one on disk, so only a change
// from here is written. Call it after the window is fully set up.
func (file *StateFile) Begin() { file.save = New(StateDebounce, file.Key()) }

// Writable reports whether the state is written to a file: it is kept on
// disk, and no newer build wrote it.
func (file *StateFile) Writable() bool {
	return file.path != "" && file.access == storage.Writable
}

// Snapshot is the state to remember: the window's part with the trusted
// accesses and the welcomed version.
func (file *StateFile) Snapshot() state.State {
	snapshot := file.window.Snapshot()
	snapshot.Trusted = file.trusted.Keys()
	snapshot.Welcomed = file.welcomed
	return snapshot
}

// Key is the comparable form of Snapshot.
func (file *StateFile) Key() string { return state.Key(file.Snapshot()) }

// Tick writes the state, off the UI goroutine, once key, its comparable
// form (the Keys cache has it), has stayed the same for StateDebounce.
// Call it every frame.
func (file *StateFile) Tick(key string, now time.Time) {
	if !file.Writable() || !file.save.Due(key, now) {
		return
	}
	snapshot, path := file.Snapshot(), file.path
	file.save.Saved(key) // an error is reported once, not retried every frame
	file.writes.Write("state:"+path, serial.Job{
		Run: func() error { return file.store.Save(path, snapshot) },
		Done: func(err error) {
			if err != nil {
				file.reports.Failed(err)
			}
		},
	})
}

// Save waits for the background writes and writes the state now; the
// program calls it on exit.
func (file *StateFile) Save() error {
	if !file.Writable() {
		return nil
	}
	file.writes.Wait()
	snapshot := file.Snapshot()
	err := file.store.Save(file.path, snapshot)
	if err == nil {
		file.save.Saved(state.Key(snapshot))
	}
	return err
}

// Restored is the window geometry of the last run, clamped to a monitor of
// monitorW by monitorH; the zero Window when nothing was remembered.
func (file *StateFile) Restored(monitorW, monitorH int) state.Window {
	return file.initial.Clamp(monitorW, monitorH)
}

// HistoryDir is the folder of the saved responses, next to the state file;
// "" when the state is not kept on disk.
func (file *StateFile) HistoryDir() string {
	if file.path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(file.path), "history")
}

// Welcomed is the version whose welcome dialog the user has seen.
func (file *StateFile) Welcomed() string { return file.welcomed }

// Welcome records that version's welcome dialog was shown.
func (file *StateFile) Welcome(version string) { file.welcomed = version }

// Trusts reports whether the user allowed access.
func (file *StateFile) Trusts(access request.Access) bool { return file.trusted.Has(access) }

// Trust remembers that the user allowed access.
func (file *StateFile) Trust(access request.Access) { file.trusted.Add(access) }
