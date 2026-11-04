package ui

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/store"
)

// saveDebounce is how long the collection must stay unchanged before
// it is saved by itself.
const saveDebounce = time.Second

// openWorkspace loads the collection from dir, or from the remembered
// workspace when dir is empty; with neither, the built-in sample stays.
// A folder that cannot be read leaves the sample in place and no
// workspace set, so a later save cannot overwrite what is there.
func (a *App) openWorkspace(dir string) {
	if dir == "" {
		dir = a.settings.Workspace
	}
	if dir != "" {
		col, err := store.Load(dir)
		if err != nil {
			a.status = "Workspace not loaded: " + err.Error()
		} else {
			a.col, a.workspace = col, dir
		}
	}
	if a.workspace == "" { // the sample opens on its search call
		a.expanded[&a.col.Folder] = true
		a.expanded[a.col.Folders[1]] = true
		a.openRequest(a.col.Folders[1].Requests[0])
	} else {
		a.expandAll(&a.col.Folder)
		first := firstRequest(&a.col.Folder)
		if first == nil {
			first = a.col.AddRequest(model.Request{Name: "New Request", Protocol: "rest", Method: "GET"})
		}
		a.openRequest(first)
	}
	a.pickEnv()
	a.colSaved = collectionKey(a.col)
	a.colSeen = a.colSaved
}

func (a *App) expandAll(f *model.Folder) {
	a.expanded[f] = true
	for _, sub := range f.Folders {
		a.expandAll(sub)
	}
}

// firstRequest is the first request in tree order, or nil.
func firstRequest(f *model.Folder) *model.Request {
	if len(f.Requests) > 0 {
		return f.Requests[0]
	}
	for _, sub := range f.Folders {
		if r := firstRequest(sub); r != nil {
			return r
		}
	}
	return nil
}

// collectionKey is a comparable form of the collection.
func collectionKey(c model.Collection) string {
	b, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// saveWorkspace writes the collection to the workspace. Without one it
// uses the default folder and remembers it.
func (a *App) saveWorkspace() {
	a.commitEdit()
	dir := a.workspace
	if dir == "" {
		dir = a.defaultWorkspace
		if dir == "" {
			a.toast("No folder to save to", true)
			return
		}
	}
	if err := store.Save(dir, a.col); err != nil {
		a.colSaved = collectionKey(a.col) // report once, do not retry every frame
		a.toast("Saving failed: "+err.Error(), true)
		return
	}
	if a.workspace == "" {
		a.workspace = dir
		a.saveSettings(func(s *settings.Settings) { s.Workspace = dir })
	}
	for _, t := range a.tabs {
		t.saved = t.req.Clone()
	}
	a.colSaved = collectionKey(a.col)
	a.colSeen = a.colSaved
	a.status = fmt.Sprintf("Saved %s", a.col.Name)
}

// workspaceTick saves the collection once it has been unchanged for
// saveDebounce. Nothing is saved until a workspace is set.
func (a *App) workspaceTick() {
	if a.workspace == "" {
		return
	}
	key := collectionKey(a.col)
	if key != a.colSeen {
		a.colSeen, a.colAt = key, a.now()
		return
	}
	if key == a.colSaved || a.now().Sub(a.colAt) < saveDebounce {
		return
	}
	a.saveWorkspace()
}
