package app

import (
	"encoding/json"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/severity"
	"path/filepath"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// stateDebounce is how long the remembered state must stay unchanged
// before it is written, so a drag does not write on every frame.
const stateDebounce = time.Second

// holderOf is the open collection that holds req and req's remembered path in
// it; ok is false when it is in no open collection.
func (a *App) holderOf(req *model.Request) (entry collections.Entry, path string, ok bool) {
	entry, ok = a.open.Owner(model.Of(req))
	if !ok {
		return collections.Entry{}, "", false
	}
	return entry, tree.PathOf(&entry.Collection.Folder, req), true
}

// requestPath is the remembered path of req in its collection, "" when it
// is in no open collection.
func (a *App) requestPath(req *model.Request) string {
	_, path, _ := a.holderOf(req)
	return path
}

// historyKeySep joins a collection's folder and a request path in a
// history key. A key is never split again, and the folder is absolute, so
// it cannot be mistaken for a bare path.
const historyKeySep = "#"

// historyKey is what req's saved responses are kept under: its
// collection's folder and its path, since two collections can hold the
// same path. A collection not saved yet has no folder, so its key is the
// bare path. "" when req is in no open collection.
func (a *App) historyKey(req *model.Request) string {
	owner, path, ok := a.holderOf(req)
	if path == "" || !ok || owner.Dir == "" {
		return path
	}
	return owner.Dir + historyKeySep + path
}

// legacyHistoryKey is the bare path req's responses were kept under while
// one collection was open at a time, for a request of the first open
// collection, which is where that one collection is now; "" otherwise.
func (a *App) legacyHistoryKey(req *model.Request) string {
	owner, ok := a.open.Owner(model.Of(req))
	if !ok || owner.Dir == "" || a.open.Entries[0].Collection != owner.Collection {
		return ""
	}
	return a.requestPath(req)
}

// openDirs are the folders of the open collections in tree order, "" for
// one not saved, as state.CollectionAt takes them.
func (a *App) openDirs() []string {
	dirs := make([]string, len(a.open.Entries))
	for i, entry := range a.open.Entries {
		dirs[i] = entry.Dir
	}
	return dirs
}

// remembered is the open collection a state reference names, nil for none.
func (a *App) remembered(ref string) *model.Collection {
	at := state.CollectionAt(ref, a.openDirs())
	if at < 0 {
		return nil
	}
	return a.open.Entries[at].Collection
}

// snapshot is the state to remember.
func (a *App) snapshot() state.State {
	s := state.State{
		Window:    state.Window{W: a.windowPx(a.w), H: a.windowPx(a.h), X: a.winX, Y: a.winY},
		SidebarW:  a.sidebarSplit.Size,
		PaneSplit: a.paneSplit.Size,
		Active:    a.list.Active,
		SettingsW: a.prefs.Size.W,
		SettingsH: a.prefs.Size.H,
		Trusted:   a.trusted.Keys(),
		Welcomed:  a.welcomed,
	}
	for _, entry := range a.open.Entries {
		a.snapshotExpanded(&s, entry)
	}
	var kept []int // tab index -> remembered index, -1 when not in the tree
	for _, t := range a.openTabs() {
		if owner, p, _ := a.holderOf(t.Req); p != "" {
			s.Tabs = append(s.Tabs, state.Tab{Collection: owner.Dir, Path: p, Pane: t.paneTab, PaneLabel: requestPaneLabel(t), Resp: t.respTab})
			kept = append(kept, len(s.Tabs)-1)
		} else {
			kept = append(kept, -1)
		}
	}
	if a.list.Active < len(kept) {
		s.Active = max(kept[a.list.Active], 0)
	}
	for name, w := range a.colWidths {
		if len(w.Weights) > 0 {
			if s.Columns == nil {
				s.Columns = map[string][]float64{}
			}
			s.Columns[name] = append([]float64(nil), w.Weights...)
		}
	}
	return s
}

// snapshotExpanded records the open folders of entry's collection: under
// its folder, or, for the collection that is not saved, in Expanded.
func (a *App) snapshotExpanded(saved *state.State, entry collections.Entry) {
	var paths []string
	tree.WalkFolders(&entry.Collection.Folder, "", func(name string, folder *model.Folder) {
		if a.expanded[folder] {
			paths = append(paths, name)
		}
	})
	if len(paths) == 0 {
		return
	}
	if entry.Dir == "" {
		saved.Expanded = append(saved.Expanded, paths...)
		return
	}
	if saved.ExpandedIn == nil {
		saved.ExpandedIn = map[string][]string{}
	}
	saved.ExpandedIn[entry.Dir] = paths
}

// expandPaths opens the folders of col at the remembered paths.
func (a *App) expandPaths(col *model.Collection, paths []string) {
	if col == nil {
		return
	}
	for _, p := range paths {
		tree.WalkFolders(&col.Folder, "", func(path string, folder *model.Folder) {
			if path == p {
				a.expanded[folder] = true
			}
		})
	}
}

// requestPaneLabel is the label of the request pane tab shows, "" when its
// index points nowhere.
func requestPaneLabel(tab *reqTab) string {
	panes := requestPanesFor(tab)
	if tab.paneTab < 0 || tab.paneTab >= len(panes) {
		return ""
	}
	return panes[tab.paneTab].label
}

// rememberedPane is the index of the request pane a remembered tab was on.
func rememberedPane(tab *reqTab, remembered state.Tab) int {
	panes := requestPanesFor(tab)
	labels := make([]string, len(panes))
	for i, p := range panes {
		labels[i] = p.label
	}
	return remembered.PaneIn(labels, tab.paneTab)
}

// restore applies a remembered state to a freshly built app. Anything
// that no longer exists is skipped, and values are kept in range.
func (a *App) restore(stateValue state.State) {
	a.trusted = model.AccessSetOf(stateValue.Trusted)
	a.prefs.Size.W, a.prefs.Size.H = max(stateValue.SettingsW, 0), max(stateValue.SettingsH, 0)
	a.sidebarSplit.Size = max(stateValue.SidebarW, 0)
	if stateValue.PaneSplit > 0 && stateValue.PaneSplit < 1 {
		a.paneSplit.Size = stateValue.PaneSplit
	}
	a.expandPaths(a.remembered(""), stateValue.Expanded)
	for dir, paths := range stateValue.ExpandedIn {
		if dir != "" {
			a.expandPaths(a.remembered(dir), paths)
		}
	}
	var saved []state.Tab
	var requests []*model.Request
	active := 0
	for i, rt := range stateValue.Tabs {
		req := a.requestAt(a.remembered(rt.Collection), rt.Path)
		if req == nil {
			continue
		}
		if i == stateValue.Active {
			active = len(saved)
		}
		saved = append(saved, rt)
		requests = append(requests, req)
	}
	if len(saved) > 0 {
		a.updateTabs(tabs.Reset{})
		for i, rt := range saved {
			a.updateTabs(tabs.Open{Req: requests[i]})
			t := a.cur()
			t.paneTab = rememberedPane(t, rt)
			if rt.Resp >= 0 && rt.Resp < len(responsePanesFor(t)) {
				t.respTab = rt.Resp
			}
		}
		a.updateTabs(tabs.Activate{Index: active})
		a.selected = model.Of(a.cur().Req)
	}
	for name, ws := range stateValue.Columns {
		if len(ws) == 0 {
			continue
		}
		if a.colWidths == nil {
			a.colWidths = map[string]*widget.ColumnWidths{}
		}
		a.colWidths[name] = &widget.ColumnWidths{Weights: append([]float64(nil), ws...)}
	}
}

// requestAt is the request of col at path, nil when there is none.
func (a *App) requestAt(col *model.Collection, path string) *model.Request {
	if col == nil {
		return nil
	}
	return tree.At(&col.Folder, path)
}

// RestoredWindow is the window geometry of the last run, clamped to a
// monitor of mw by mh; the zero Window when nothing was remembered.
func (a *App) RestoredWindow(mw, mh int) state.Window { return a.initial.Clamp(mw, mh) }

// SaveState writes the state now; main calls it on exit.
func (a *App) SaveState() error {
	if !a.stateWritable() {
		return nil
	}
	a.waitWrites()
	s := a.snapshot()
	err := a.states.Save(a.statePath, s)
	if err == nil {
		a.stateSave.Saved(stateKey(s))
	}
	return err
}

func stateKey(saved state.State) string {
	b, _ := json.Marshal(saved)
	return string(b)
}

// stateTick writes the state, off the UI goroutine, once it has been
// unchanged for stateDebounce.
func (a *App) stateTick() {
	if !a.stateWritable() {
		return
	}
	key := a.stateNow()
	if !a.stateSave.Due(key, a.now()) {
		return
	}
	snapshot, path := a.snapshot(), a.statePath
	a.stateSave.Saved(key) // an error is reported once, not retried every frame
	a.write("state:"+path, serial.Job{
		Run: func() error { return a.states.Save(path, snapshot) },
		Done: func(err error) {
			if err != nil {
				a.toast(err.Error(), severity.Error)
			}
		},
	})
}

// stateNote joins a state load problem into the status line.
func stateNote(status string, err error) string {
	if err == nil {
		return status
	}
	return strings.TrimSpace(status + " " + err.Error())
}

// windowPx converts a logical size to window pixels (device independent),
// which is what the window system and the remembered geometry use.
func (a *App) windowPx(logical int) int {
	scale := a.windowScale
	if scale == 0 {
		scale = a.settings.UIScale() // before the first Layout
	}
	return int(float64(logical)*scale + 0.5)
}

// historyDir is the folder of the saved responses, next to the state
// file; "" when the state is not kept on disk.
func (a *App) historyDir() string {
	if a.statePath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(a.statePath), "history")
}

// historyStore is the history folder next to the state file, nil when
// nothing is saved or no history adapter was given.
func (a *App) historyStore() rport.History {
	dir := a.historyDir()
	if dir == "" || a.histories == nil {
		return nil
	}
	return a.histories(dir)
}

// saveHistory writes the successful responses of the tab's request, the
// shown one first, so they are there after a restart. A failed write is
// shown in the status line and nothing else changes.
func (a *App) saveHistory(tab *reqTab) {
	store, key := a.historyStore(), a.historyKey(tab.Req)
	if store == nil || key == "" {
		return
	}
	var entries []rport.Entry
	if tab.Resp != nil && tab.Resp.Err == "" {
		entries = append(entries, historyEntry(tab.Resp, a.now()))
	}
	for _, earlier := range tab.Earlier {
		entries = append(entries, historyEntry(earlier.Resp, earlier.At))
	}
	a.write("history:"+key, serial.Job{ // the entries are values; the bodies are never changed
		Run: func() error { return store.Save(key, entries) },
		Done: func(err error) {
			if err != nil {
				a.toast("History is not being saved: "+err.Error(), severity.Error)
			}
		},
	})
}

// loadHistory is the saved responses of req, newest first.
func (a *App) loadHistory(req *model.Request) []tabs.PastResponse {
	store, key := a.historyStore(), a.historyKey(req)
	if store == nil || key == "" {
		return nil
	}
	saved, err := store.Load(key)
	if legacy := a.legacyHistoryKey(req); err == nil && len(saved) == 0 && legacy != "" {
		saved, err = store.Load(legacy)
	}
	if err != nil {
		a.toast("History could not be read: "+err.Error(), severity.Error)
		return nil
	}
	entries := make([]tabs.PastResponse, 0, len(saved))
	for _, entry := range saved {
		entries = append(entries, tabs.PastResponse{Resp: &model.Response{
			Status: entry.Status, StatusText: entry.StatusText, Duration: entry.Duration,
			Size: len(entry.Body), Headers: entry.Headers, Body: entry.Body,
		}, At: entry.At})
	}
	return entries
}

func historyEntry(resp *model.Response, at time.Time) rport.Entry {
	return rport.Entry{At: at, Status: resp.Status, StatusText: resp.StatusText, Duration: resp.Duration, Headers: resp.Headers, Body: resp.Body}
}

// stateWritable reports whether the state is written to a file: it is kept
// on disk, and no newer build wrote it.
func (a *App) stateWritable() bool {
	return a.statePath != "" && a.stateAccess == fileversion.Writable
}

// noteReadOnly tells once, at the start, that a file a newer build wrote is
// left as it is for this session.
func (a *App) noteReadOnly(access fileversion.Access, loadErr error) {
	if access == fileversion.ReadOnly {
		a.toast(loadErr.Error()+"; it is not changed this session", severity.Error)
	}
}
