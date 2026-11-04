package ui

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/state"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/vml"
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
		Window:    state.Window{W: a.w, H: a.h, X: a.winX, Y: a.winY},
		SidebarW:  a.sidebarSplit.Size,
		PaneSplit: a.paneSplit.Size,
		Active:    a.active,
	}
	walkFolders(&a.col.Folder, "", func(p string, f *model.Folder) {
		if a.expanded[f] {
			s.Expanded = append(s.Expanded, p)
		}
	})
	var kept []int // tab index -> remembered index, -1 when not in the tree
	for _, t := range a.tabs {
		if p := a.requestPath(t.req); p != "" {
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
		t := newTab(req)
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
		a.selected = tabs[active].req
	}
	for name, ws := range s.Columns {
		if len(ws) == 0 {
			continue
		}
		if a.colWidths == nil {
			a.colWidths = map[string]*vml.ColumnWidths{}
		}
		a.colWidths[name] = &vml.ColumnWidths{Weights: append([]float64(nil), ws...)}
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
	err := s.Save(a.statePath)
	if err == nil {
		a.stateSaved = stateKey(s)
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
	if key != a.stateSeen {
		a.stateSeen, a.stateAt = key, a.now()
		return
	}
	if key == a.stateSaved || a.now().Sub(a.stateAt) < stateDebounce {
		return
	}
	if err := a.SaveState(); err != nil {
		a.stateSaved = key // report once, do not retry every frame
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
