package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
)

// saveDebounce is how long the collection must stay unchanged before
// it is saved by itself.
const saveDebounce = time.Second

// openWorkspace loads the collection from dir, or from the remembered
// workspace when dir is empty; with neither, the build's client config, if it
// has one, replaces the built-in sample.
// A folder that cannot be read leaves the sample in place and no
// workspace set, so a later save cannot overwrite what is there.
func (a *App) openWorkspace(dir string) {
	if dir == "" {
		dir = a.settings.Workspace
	}
	if dir != "" {
		col, err := a.store.Load(dir)
		if err != nil {
			a.status = "Workspace not loaded: " + err.Error()
		} else {
			a.col, a.workspace = col, dir
		}
	}
	sample := a.workspace == "" && !a.loadClients()
	if sample && len(a.col.Folders) > 1 { // the sample opens on its search call
		a.expanded[&a.col.Folder] = true
		a.expanded[a.col.Folders[1]] = true
		a.openRequest(a.col.Folders[1].Requests[0])
	} else {
		a.expandAll(&a.col.Folder)
		first := tree.FirstRequest(&a.col.Folder)
		if first == nil {
			first = a.col.AddRequest(model.Request{Name: "New Request", Protocol: "rest", Method: "GET"})
		}
		a.openRequest(first)
	}
	a.pickEnv()
	a.colSave.Saved(a.collectionKeyNow())
}

// loadClients replaces the collection with the client config's tree. It
// reports false when there is no config or it is empty, and when it cannot
// be read, which the status then says.
func (a *App) loadClients() bool {
	if a.clients == nil {
		return false
	}
	config, err := a.clients.Load()
	if err != nil {
		a.status = "Client config not loaded: " + err.Error()
		return false
	}
	if config.Empty() {
		return false
	}
	a.col = config.Collection()
	return true
}

func (a *App) expandAll(f *model.Folder) {
	a.expanded[f] = true
	for _, sub := range f.Folders {
		a.expandAll(sub)
	}
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
	snapshot := a.col.Clone()             // written off the UI goroutine while the UI goes on editing
	a.colSave.Saved(a.collectionKeyNow()) // an error is reported once, not retried every frame
	for _, t := range a.openTabs() {
		t.MarkSaved()
	}
	a.write("collection:"+dir, serial.Job{
		Run: func() error { return a.store.Save(dir, snapshot) },
		Done: func(err error) {
			if err != nil {
				a.toast("Saving failed: "+err.Error(), true)
				return
			}
			if a.workspace == "" {
				a.workspace = dir
				a.saveSettings(func(s *settings.Settings) { s.Workspace = dir })
			}
			a.status = fmt.Sprintf("Saved %s", snapshot.Name)
		},
	})
}

// workspaceTick saves the collection once it has been unchanged for
// saveDebounce. Nothing is saved until a workspace is set.
func (a *App) workspaceTick() {
	if a.workspace == "" {
		return
	}
	if a.colSave.Due(a.collectionKey(), a.now()) {
		a.saveWorkspace()
	}
}

// FlushWorkspace saves the collection if it changed since the last save,
// for the moment the window closes (the autosave pause may not have
// passed). Nothing is written without a workspace.
func (a *App) FlushWorkspace() error {
	if a.workspace == "" {
		return nil
	}
	a.commitEdit()
	a.waitWrites()
	if !a.colSave.Pending(a.collectionKeyNow()) {
		return nil
	}
	if err := a.store.Save(a.workspace, a.col); err != nil {
		return err
	}
	a.colSave.Saved(a.collectionKeyNow())
	return nil
}

// openWorkspaceAt switches to the collection in dir: pending edits of
// the current one are saved, every tab is closed and the folder becomes
// the remembered workspace. A folder that cannot be read changes nothing
// and says why.
func (a *App) openWorkspaceAt(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	if err := a.FlushWorkspace(); err != nil {
		a.toast("Saving the current collection failed: "+err.Error(), true)
		return
	}
	col, err := a.store.Load(dir)
	if err != nil {
		a.toast("Cannot open "+dir+": "+err.Error(), true)
		return
	}
	for _, t := range a.openTabs() {
		a.disconnect(t)
	}
	a.commitEdit()
	a.col, a.workspace = col, dir
	a.updateTabs(tabs.Reset{})
	a.selected = nil
	a.expanded = map[*model.Folder]bool{}
	a.expandAll(&a.col.Folder)
	first := tree.FirstRequest(&a.col.Folder)
	if first == nil {
		first = a.col.AddRequest(model.Request{Name: "New Request", Protocol: "rest", Method: "GET"})
	}
	a.openRequest(first)
	a.pickEnv()
	a.colSave.Saved(a.collectionKeyNow())
	a.saveSettings(func(s *settings.Settings) { s.Workspace = dir })
	a.status = "Opened " + dir
}

// promptWorkspace asks for the folder of a collection to open; an empty or
// new folder starts a new collection there.
func (a *App) promptWorkspace() {
	a.prompt("Open collection folder", a.workspace, "Open", a.openWorkspaceAt)
}
