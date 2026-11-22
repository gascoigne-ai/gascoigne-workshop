package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/severity"
	"path/filepath"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Whole collections: new, open, rename, save, close, move and delete. A collection is a
// folder on disk; the open ones are a.open (see openset.go). The rules are
// in collection/collections, the I/O behind the store port.

// collectionsDir is the folder new collections are made in: beside the
// default workspace, never inside it, since a folder with a folder.json
// below a collection is one of its folders. Empty when the program has no
// config directory.
func (a *App) collectionsDir() string {
	if a.defaultWorkspace == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(a.defaultWorkspace), "collections")
}

// collectionMenu is the menu of the Collections chip, which acts on the
// focused collection.
func (a *App) collectionMenu() []MenuItem { return a.collectionMenuOf(a.focusCol()) }

// collectionMenuOf is the collection menu acting on col; with nothing open
// (col nil) only New and Open act.
func (a *App) collectionMenuOf(col *model.Collection) []MenuItem {
	none := col == nil
	return []MenuItem{
		{Label: "New collection...", Run: a.promptNewCollection},
		{Label: "Open collection folder...", Run: a.promptWorkspace},
		menuSeparator,
		{Label: "Rename collection...", Disabled: none, Run: func() { a.promptRenameCollection(col) }},
		{Label: "Save collection", Disabled: none, Run: func() { a.saveCollection(col) }},
		menuSeparator,
		{Label: "Close collection", Disabled: none, Run: func() { a.requestCloseCollection(col) }},
		{Label: "Delete collection...", Disabled: none, Run: func() { a.confirmDeleteCollection(col) }},
	}
}

// promptNewCollection asks for a name, then makes the collection and opens
// it beside the open ones.
func (a *App) promptNewCollection() {
	a.prompt("New collection", "", "Create", a.createCollection)
}

func (a *App) createCollection(text string) {
	name, err := collections.CleanName(text)
	if err != nil {
		a.toast("Cannot create a collection: "+err.Error(), severity.Error)
		return
	}
	parent := a.collectionsDir()
	if parent == "" {
		a.toast("There is no folder for new collections", severity.Error)
		return
	}
	dir, err := a.store.Create(parent, name)
	if err != nil {
		a.toast("Cannot create the collection: "+err.Error(), severity.Error)
		return
	}
	a.openCollectionAt(dir)
}

// promptRenameCollection asks for the new name of col.
func (a *App) promptRenameCollection(col *model.Collection) {
	if !a.onDisk(col, "rename") {
		return
	}
	a.prompt("Rename collection", col.Name, "Rename", func(text string) { a.renameCollection(col, text) })
}

func (a *App) renameCollection(col *model.Collection, text string) {
	name, err := collections.CleanName(text)
	if err != nil {
		a.toast("Cannot rename: "+err.Error(), severity.Error)
		return
	}
	if err := a.FlushWorkspace(); err != nil { // the writes in flight go to the old folder first
		a.toast("Saving the collection failed: "+err.Error(), severity.Error)
		return
	}
	entry, ok := a.open.ByCollection(col)
	if !ok {
		return
	}
	dir, err := a.store.Rename(entry.Dir, name)
	if err != nil {
		a.toast("Cannot rename: "+err.Error(), severity.Error)
		return
	}
	col.Name = name
	col.MoveProtoPaths(entry.Dir, dir)
	a.open = collections.Update(a.open, collections.Located{Collection: col, Dir: dir})
	a.rememberCollections()
	a.markSaved(col)
	a.status = "Renamed to " + name
}

// confirmDeleteCollection asks, then deletes col.
func (a *App) confirmDeleteCollection(col *model.Collection) {
	if !a.onDisk(col, "delete") {
		return
	}
	title, body := collections.DeletePrompt(col.Name)
	a.confirm(title, body, "Delete", func() { a.deleteCollection(col) })
}

// deleteCollection removes col from the disk and from the window. When it
// was the only one open, the next collection of the collections folder
// opens, or a new empty one when none is left.
func (a *App) deleteCollection(col *model.Collection) {
	a.commitEdit()
	a.waitWrites() // a write still in flight would make the folder again
	entry, ok := a.open.ByCollection(col)
	if !ok {
		return
	}
	dir := entry.Dir
	if err := a.store.Delete(dir); err != nil {
		a.toast("Cannot delete the collection: "+err.Error(), severity.Error)
		return
	}
	// nothing is left to flush into the deleted folder
	a.open = collections.Update(a.open, collections.Located{Collection: col, Dir: ""})
	if len(a.open.Entries) > 1 {
		a.closeCollection(col)
		a.toast("Deleted "+filepath.Base(dir), severity.Info)
		return
	}
	listed, _ := a.store.List(a.collectionsDir())
	next := collections.NextAfterDelete(listed, dir)
	if next == "" {
		var err error
		if next, err = a.store.Create(a.collectionsDir(), "My collection"); err != nil {
			a.toast("Deleted, but no collection could be opened: "+err.Error(), severity.Error)
			return
		}
	}
	a.openWorkspaceAt(next)
	a.toast("Deleted "+filepath.Base(dir), severity.Info)
}

// onDisk is true when col has a folder; otherwise it tells the user to
// save first, since there is nothing to rename or delete.
func (a *App) onDisk(col *model.Collection, verb string) bool {
	if entry, ok := a.open.ByCollection(col); ok && entry.Dir != "" {
		return true
	}
	a.toast("The collection is not saved yet: save it first to "+verb+" it", severity.Error)
	return false
}
