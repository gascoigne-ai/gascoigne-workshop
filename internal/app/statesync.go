package app

import (
	"encoding/json"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"path/filepath"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// stateDebounce is how long the remembered state must stay unchanged
// before it is written, so a drag does not write on every frame.
const stateDebounce = time.Second

// requestPath is the remembered path of req in its collection, "" when it
// is in no open collection.
func (a *App) requestPath(req *request.Request) string {
	_, path, _ := a.cols.Set().PathOf(req)
	return path
}

// historyKey is what req's saved responses are kept under: the open set's
// key for it.
func (a *App) historyKey(req *request.Request) string { return a.cols.Set().HistoryKey(req) }

// remembered is the open collection a state reference names, nil for none.
func (a *App) remembered(ref string) *request.Collection {
	at := state.CollectionAt(ref, a.cols.Set().Dirs())
	if at < 0 {
		return nil
	}
	return a.cols.Set().Entries[at].Collection
}

// snapshot is the state to remember.
func (a *App) snapshot() state.State {
	s := state.State{
		Window:    state.Window{W: a.windowPx(a.w), H: a.windowPx(a.h), X: a.winX, Y: a.winY},
		SidebarW:  a.sidebarSplit.Size,
		PaneSplit: a.paneSplit.Size,
		Active:    a.tabs.Active(),
		SettingsW: a.over.Settings.Shell.Size.W,
		SettingsH: a.over.Settings.Shell.Size.H,
		Trusted:   a.trusted.Keys(),
		Welcomed:  a.welcomed,
	}
	for _, entry := range a.cols.Set().Entries {
		a.snapshotExpanded(&s, entry)
	}
	var kept []int // tab index -> remembered index, -1 when not in the tree
	for _, t := range a.openTabs() {
		if owner, p, _ := a.cols.Set().PathOf(t.Req); p != "" {
			s.Tabs = append(s.Tabs, state.Tab{Collection: owner.Dir, Path: p, Pane: t.paneTab, PaneLabel: requestPaneLabel(t), Resp: t.respTab})
			kept = append(kept, len(s.Tabs)-1)
		} else {
			kept = append(kept, -1)
		}
	}
	if a.tabs.Active() < len(kept) {
		s.Active = max(kept[a.tabs.Active()], 0)
	}
	for kind := kvlogic.Params; kind < kvlogic.KindCount; kind++ {
		if weights := a.kv.Widths(kind).Weights; len(weights) > 0 {
			if s.Columns == nil {
				s.Columns = map[string][]float64{}
			}
			s.Columns[kind.Key()] = append([]float64(nil), weights...)
		}
	}
	return s
}

// snapshotExpanded records the open folders of entry's collection: under
// its folder, or, for the collection that is not saved, in Expanded.
func (a *App) snapshotExpanded(saved *state.State, entry collections.Entry) {
	paths := a.cols.Set().ExpandedPaths(entry, func(folder *request.Folder) bool { return a.side.Tree().Expanded()[folder] })
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
func (a *App) expandPaths(col *request.Collection, paths []string) {
	for _, folder := range a.cols.Set().FoldersAt(col, paths) {
		a.side.Tree().Expand(folder)
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
	a.trusted = request.AccessSetOf(stateValue.Trusted)
	a.over.Settings.Shell.Size.W, a.over.Settings.Shell.Size.H = max(stateValue.SettingsW, 0), max(stateValue.SettingsH, 0)
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
	var requests []*request.Request
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
			if rt.Resp >= 0 && rt.Resp < len(a.responsePanesFor(t)) {
				t.respTab = rt.Resp
			}
		}
		a.updateTabs(tabs.Activate{Index: active})
		a.side.Tree().Select(request.Node{Req: a.cur().Req})
	}
	for name, ws := range stateValue.Columns {
		if len(ws) == 0 {
			continue
		}
		if kind, ok := kvlogic.KindOfKey(name); ok {
			*a.kv.Widths(kind) = widget.TableState{Weights: append([]float64(nil), ws...)}
		}
	}
}

// requestAt is the request of col at path, nil when there is none.
func (a *App) requestAt(col *request.Collection, path string) *request.Request {
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
	a.writes.Wait()
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
	key := a.keys.State()
	if !a.stateSave.Due(key, a.now()) {
		return
	}
	snapshot, path := a.snapshot(), a.statePath
	a.stateSave.Saved(key) // an error is reported once, not retried every frame
	a.writes.Write("state:"+path, serial.Job{
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
func (a *App) historyStore() history.History {
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
	var entries []history.Entry
	if tab.Resp != nil && tab.Resp.Err == "" {
		entries = append(entries, historyEntry(tab.Resp, a.now()))
	}
	for _, earlier := range tab.Earlier {
		entries = append(entries, historyEntry(earlier.Resp, earlier.At))
	}
	a.writes.Write("history:"+key, serial.Job{ // the entries are values; the bodies are never changed
		Run: func() error { return store.Save(key, entries) },
		Done: func(err error) {
			if err != nil {
				a.toast("History is not being saved: "+err.Error(), severity.Error)
			}
		},
	})
}

// loadHistory is the saved responses of req, newest first.
func (a *App) loadHistory(req *request.Request) []tabs.PastResponse {
	store, key := a.historyStore(), a.historyKey(req)
	if store == nil || key == "" {
		return nil
	}
	saved, err := store.Load(key)
	if legacy := a.cols.Set().LegacyHistoryKey(req); err == nil && len(saved) == 0 && legacy != "" {
		saved, err = store.Load(legacy)
	}
	if err != nil {
		a.toast("History could not be read: "+err.Error(), severity.Error)
		return nil
	}
	entries := make([]tabs.PastResponse, 0, len(saved))
	for _, entry := range saved {
		entries = append(entries, tabs.PastResponse{Resp: &request.Response{
			Status: entry.Status, StatusText: entry.StatusText, Duration: entry.Duration,
			Size: len(entry.Body), Headers: entry.Headers, Body: entry.Body,
		}, At: entry.At})
	}
	return entries
}

func historyEntry(resp *request.Response, at time.Time) history.Entry {
	return history.Entry{At: at, Status: resp.Status, StatusText: resp.StatusText, Duration: resp.Duration, Headers: resp.Headers, Body: resp.Body}
}

// stateWritable reports whether the state is written to a file: it is kept
// on disk, and no newer build wrote it.
func (a *App) stateWritable() bool {
	return a.statePath != "" && a.stateAccess == storage.Writable
}

// noteReadOnly tells once, at the start, that a file a newer build wrote is
// left as it is for this session.
func (a *App) noteReadOnly(access storage.Access, loadErr error) {
	if access == storage.ReadOnly {
		a.toast(loadErr.Error()+"; it is not changed this session", severity.Error)
	}
}
