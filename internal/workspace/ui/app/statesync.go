package ui

import (
	"encoding/json"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"path/filepath"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// stateDebounce is how long the remembered state must stay unchanged
// before it is written, so a drag does not write on every frame.
const stateDebounce = time.Second

// pathSep joins folder and request names in remembered paths.
const pathSep = "/"

// walkFolders calls fn for every folder below root, with its path.
func walkFolders(f *model.Folder, prefix string, fn func(path string, f *model.Folder)) {
	for _, sub := range f.Folders {
		p := prefix + sub.Name
		fn(p, sub)
		walkFolders(sub, p+pathSep, fn)
	}
}

// walkRequests calls fn for every request below root, with its path.
func walkRequests(f *model.Folder, prefix string, fn func(path string, r *model.Request)) {
	for _, r := range f.Requests {
		fn(prefix+r.Name, r)
	}
	for _, sub := range f.Folders {
		walkRequests(sub, prefix+sub.Name+pathSep, fn)
	}
}

// requestPath is the remembered path of req, "" when it is not in the tree.
func (a *App) requestPath(req *model.Request) string {
	found := ""
	walkRequests(&a.col.Folder, "", func(p string, r *model.Request) {
		if r == req && found == "" {
			found = p
		}
	})
	return found
}

// snapshot is the state to remember.
func (a *App) snapshot() state.State {
	s := state.State{
		Window:    state.Window{W: a.windowPx(a.w), H: a.windowPx(a.h), X: a.winX, Y: a.winY},
		SidebarW:  a.sidebarSplit.Size,
		PaneSplit: a.paneSplit.Size,
		Active:    a.active,
		SettingsW: a.prefs.Size.W,
		SettingsH: a.prefs.Size.H,
	}
	walkFolders(&a.col.Folder, "", func(p string, f *model.Folder) {
		if a.expanded[f] {
			s.Expanded = append(s.Expanded, p)
		}
	})
	var kept []int // tab index -> remembered index, -1 when not in the tree
	for _, t := range a.tabs {
		if p := a.requestPath(t.Req); p != "" {
			s.Tabs = append(s.Tabs, state.Tab{Path: p, Pane: t.paneTab, Resp: t.respTab})
			kept = append(kept, len(s.Tabs)-1)
		} else {
			kept = append(kept, -1)
		}
	}
	if a.active < len(kept) {
		s.Active = max(kept[a.active], 0)
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

// restore applies a remembered state to a freshly built app. Anything
// that no longer exists is skipped, and values are kept in range.
func (a *App) restore(s state.State) {
	a.prefs.Size.W, a.prefs.Size.H = max(s.SettingsW, 0), max(s.SettingsH, 0)
	a.sidebarSplit.Size = max(s.SidebarW, 0)
	if s.PaneSplit > 0 && s.PaneSplit < 1 {
		a.paneSplit.Size = s.PaneSplit
	}
	for _, p := range s.Expanded {
		walkFolders(&a.col.Folder, "", func(path string, f *model.Folder) {
			if path == p {
				a.expanded[f] = true
			}
		})
	}
	byPath := map[string]*model.Request{}
	walkRequests(&a.col.Folder, "", func(p string, r *model.Request) { byPath[p] = r })
	var tabs []*reqTab
	active := 0
	for i, rt := range s.Tabs {
		req := byPath[rt.Path]
		if req == nil {
			continue
		}
		t := a.makeTab(req)
		if rt.Pane >= 0 && rt.Pane < len(requestPanes) {
			t.paneTab = rt.Pane
		}
		if rt.Resp >= 0 && rt.Resp < len(responsePanes) {
			t.respTab = rt.Resp
		}
		if i == s.Active {
			active = len(tabs)
		}
		tabs = append(tabs, t)
	}
	if len(tabs) > 0 {
		a.tabs, a.active = tabs, active
		a.selected = tabs[active].Req
	}
	for name, ws := range s.Columns {
		if len(ws) == 0 {
			continue
		}
		if a.colWidths == nil {
			a.colWidths = map[string]*view.ColumnWidths{}
		}
		a.colWidths[name] = &view.ColumnWidths{Weights: append([]float64(nil), ws...)}
	}
}

// RestoredWindow is the window geometry of the last run, clamped to a
// monitor of mw by mh; the zero Window when nothing was remembered.
func (a *App) RestoredWindow(mw, mh int) state.Window { return a.initial.Clamp(mw, mh) }

// SaveState writes the state now; main calls it on exit.
func (a *App) SaveState() error {
	if a.statePath == "" {
		return nil
	}
	s := a.snapshot()
	err := a.states.Save(a.statePath, s)
	if err == nil {
		a.stateSave.Saved(stateKey(s))
	}
	return err
}

func stateKey(s state.State) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// stateTick writes the state once it has been unchanged for stateDebounce.
func (a *App) stateTick() {
	if a.statePath == "" {
		return
	}
	key := stateKey(a.snapshot())
	if !a.stateSave.Due(key, a.now()) {
		return
	}
	if err := a.SaveState(); err != nil {
		a.stateSave.Saved(key) // report once, do not retry every frame
		a.toast(err.Error(), true)
	}
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
func (a *App) windowPx(logical int) int { return int(float64(logical)*a.settings.UIScale() + 0.5) }

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
func (a *App) saveHistory(t *reqTab) {
	store, key := a.historyStore(), a.requestPath(t.Req)
	if store == nil || key == "" {
		return
	}
	var entries []rport.Entry
	if t.Resp != nil && t.Resp.Err == "" {
		entries = append(entries, historyEntry(t.Resp, a.now()))
	}
	for _, earlier := range t.Earlier {
		entries = append(entries, historyEntry(earlier.Resp, earlier.At))
	}
	if err := store.Save(key, entries); err != nil {
		a.toast("History is not being saved: "+err.Error(), true)
	}
}

// loadHistory is the saved responses of req, newest first.
func (a *App) loadHistory(req *model.Request) []tabs.PastResponse {
	store, key := a.historyStore(), a.requestPath(req)
	if store == nil || key == "" {
		return nil
	}
	saved, err := store.Load(key)
	if err != nil {
		a.toast("History could not be read: "+err.Error(), true)
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
