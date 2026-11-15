package ui

import (
	"path/filepath"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Whole collections: new, open, rename, save and delete. A collection is a
// folder on disk and one is open at a time (a.workspace); the rules are in
// collection/controller/collections, the I/O behind the store port.

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

// collectionMenu is the menu of the Collections chip and of the
// collection's own row in the tree.
func (a *App) collectionMenu() []MenuItem {
	return []MenuItem{
		{Label: "New collection...", Run: a.promptNewCollection},
		{Label: "Open collection folder...", Run: a.promptWorkspace},
		menuSeparator,
		{Label: "Rename collection...", Run: a.promptRenameCollection},
		{Label: "Save collection", Run: a.saveWorkspace},
		menuSeparator,
		{Label: "Delete collection...", Run: a.confirmDeleteCollection},
	}
}

// promptNewCollection asks for a name, then makes the collection and opens it.
func (a *App) promptNewCollection() {
	a.prompt("New collection", "", "Create", a.createCollection)
}

func (a *App) createCollection(text string) {
	name, err := collections.CleanName(text)
	if err != nil {
		a.toast("Cannot create a collection: "+err.Error(), true)
		return
	}
	parent := a.collectionsDir()
	if parent == "" {
		a.toast("There is no folder for new collections", true)
		return
	}
	if err := a.FlushWorkspace(); err != nil {
		a.toast("Saving the current collection failed: "+err.Error(), true)
		return
	}
	dir, err := a.store.Create(parent, name)
	if err != nil {
		a.toast("Cannot create the collection: "+err.Error(), true)
		return
	}
	a.openWorkspaceAt(dir)
}

// promptRenameCollection asks for the new name of the open collection.
func (a *App) promptRenameCollection() {
	if !a.onDisk("rename") {
		return
	}
	a.prompt("Rename collection", a.col.Name, "Rename", a.renameCollection)
}

func (a *App) renameCollection(text string) {
	name, err := collections.CleanName(text)
	if err != nil {
		a.toast("Cannot rename: "+err.Error(), true)
		return
	}
	if err := a.FlushWorkspace(); err != nil { // the writes in flight go to the old folder first
		a.toast("Saving the collection failed: "+err.Error(), true)
		return
	}
	dir, err := a.store.Rename(a.workspace, name)
	if err != nil {
		a.toast("Cannot rename: "+err.Error(), true)
		return
	}
	a.col.Name = name
	a.workspace = dir
	a.saveSettings(func(s *settings.Settings) { s.Workspace = dir })
	a.colSave.Saved(a.collectionKeyNow())
	a.status = "Renamed to " + name
}

// confirmDeleteCollection asks, then deletes the open collection.
func (a *App) confirmDeleteCollection() {
	if !a.onDisk("delete") {
		return
	}
	title, body := collections.DeletePrompt(a.col.Name)
	a.confirm(title, body, "Delete", a.deleteCollection)
}

// deleteCollection removes the open collection from the disk and opens the
// next one of the collections folder, or a new empty one when none is left.
func (a *App) deleteCollection() {
	a.commitEdit()
	a.waitWrites() // a write still in flight would make the folder again
	dir := a.workspace
	if err := a.store.Delete(dir); err != nil {
		a.toast("Cannot delete the collection: "+err.Error(), true)
		return
	}
	a.workspace = "" // nothing is left to flush into the deleted folder
	listed, _ := a.store.List(a.collectionsDir())
	next := collections.NextAfterDelete(listed, dir)
	if next == "" {
		var err error
		if next, err = a.store.Create(a.collectionsDir(), "My collection"); err != nil {
			a.toast("Deleted, but no collection could be opened: "+err.Error(), true)
			return
		}
	}
	a.openWorkspaceAt(next)
	a.toast("Deleted "+filepath.Base(dir), false)
}

// onDisk is true when the open collection has a folder; otherwise it tells
// the user to save first, since there is nothing to rename or delete.
func (a *App) onDisk(verb string) bool {
	if a.workspace != "" {
		return true
	}
	a.toast("The collection is not saved yet: save it first to "+verb+" it", true)
	return false
}
