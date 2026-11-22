package app

import (
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/openset"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/tabs"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// saveDebounce is how long a collection must stay unchanged before it is
// saved by itself.
const saveDebounce = time.Second

// openWorkspace opens the collection in dir, or, when dir is empty, the
// collections remembered in the settings (an older file's single workspace
// is a set of one). With none of them, the build's client config, if it
// has one, replaces the built-in sample. A folder that cannot be read is
// left out and named in the status line, so a later save cannot
// overwrite what is there.
func (a *App) openWorkspace(dir string) {
	var wanted []openset.OpenCollection
	switch {
	case dir != "":
		wanted = []openset.OpenCollection{{Folder: dir, Environment: a.rememberedEnv(collections.AbsDir(dir))}}
	default:
		wanted = a.settings.RememberedCollections()
		a.setIsRemembered = true
	}
	var failed []error
	var sample *request.Collection
	a.unreadable = nil
	for _, item := range wanted {
		if item.Folder == "" {
			sample = a.includeStart(item.Environment)
			continue
		}
		if _, err := a.loadCollection(collections.AbsDir(item.Folder), item.Environment); err != nil {
			failed = append(failed, err)
			a.toastNewer(item.Folder, err)
			if dir == "" {
				a.unreadable = append(a.unreadable, item)
			}
		}
	}
	if len(failed) > 0 {
		a.status = "Workspace not loaded: " + errors.Join(failed...).Error()
	}
	if len(a.open.Entries) == 0 {
		sample = a.includeStart(a.rememberedEnv(""))
	}
	first := a.open.Entries[0].Collection
	if _, start, ok := collection.SampleStart(first); ok && first == sample { // the sample opens on its search call
		a.openRequest(start)
	} else {
		a.focusCollection(first)
	}
	for _, entry := range a.open.Entries { // a blank request added to an empty collection is not an edit
		a.markSaved(entry.Collection)
	}
}

// toastNewer tells, beyond the status line, that the collection in folder
// was not opened because a newer build saved it.
func (a *App) toastNewer(folder string, err error) {
	var newer *fileversion.NewerError
	if errors.As(err, &newer) {
		a.toast("Cannot open "+folder+": "+err.Error(), severity.Error)
	}
}

// includeStart opens the collection the program starts with when nothing
// saved is open: the client config's, else the sample, which it returns
// (nil for the config's). Only one is opened, however often a settings
// file names it.
func (a *App) includeStart(env string) (sample *request.Collection) {
	if _, open := a.open.At(""); open {
		return nil
	}
	col, isSample := a.startCollection()
	if folder, _, ok := collection.SampleStart(col); ok && isSample { // the sample opens on its search call
		a.expanded[&col.Folder] = true
		a.expanded[folder] = true
	} else {
		a.expandAll(&col.Folder)
	}
	a.include(col, "", env)
	if isSample {
		return col
	}
	return nil
}

// startCollection is the client config's tree, or the sample when there is
// no config, it is empty, or it cannot be read (the status then says why).
func (a *App) startCollection() (col *request.Collection, isSample bool) {
	if a.clients != nil {
		config, err := a.clients.Load()
		if err != nil {
			a.status = "Client config not loaded: " + err.Error()
		} else if !config.Empty() {
			configured := config.Collection()
			return &configured, false
		}
	}
	sample := startCollection(a.sample)
	return &sample, true
}

func (a *App) expandAll(folder *request.Folder) {
	a.expanded[folder] = true
	for _, sub := range folder.Folders {
		a.expandAll(sub)
	}
}

// saveWorkspace writes the focused collection now (Ctrl+S).
func (a *App) saveWorkspace() { a.saveCollection(a.focusCol()) }

// saveCollection writes col to its folder. A collection that has none yet
// gets its own new folder in the collections folder, as a new collection
// does, and the open set with it is remembered.
func (a *App) saveCollection(col *request.Collection) {
	a.commitEdit()
	entry, ok := a.open.ByCollection(col)
	if !ok {
		return
	}
	dir := entry.Dir
	if dir == "" {
		if dir = a.locate(col); dir == "" {
			return
		}
	}
	a.treeStale = true      // the tabs of col are marked saved
	snapshot := col.Clone() // written off the UI goroutine while the UI goes on editing
	a.markSaved(col)        // an error is reported once, not retried every frame
	a.markTabsSaved(col)
	a.write("collection:"+dir, serial.Job{
		Run: func() error { return a.store.Save(dir, snapshot) },
		Done: func(err error) {
			if err != nil {
				a.toast("Saving failed: "+err.Error(), severity.Error)
				return
			}
			a.status = fmt.Sprintf("Saved %s", snapshot.Name)
		},
	})
}

// locate makes the folder of an unsaved collection, named after it, in the
// collections folder, and records it; "" when there is none to make.
func (a *App) locate(col *request.Collection) string {
	parent := a.collectionsDir()
	if parent == "" {
		a.toast("No folder to save to", severity.Error)
		return ""
	}
	name, err := collections.CleanName(col.Name)
	if err != nil {
		name = "My collection" // the folder needs a name even when the collection's cannot be one
	}
	dir, err := a.store.Create(parent, name)
	if err != nil {
		a.toast("Saving failed: "+err.Error(), severity.Error)
		return ""
	}
	a.open = collections.Update(a.open, collections.Located{Collection: col, Dir: dir})
	a.rememberCollections()
	return dir
}

// workspaceTick saves each saved collection once it has been unchanged for
// saveDebounce. A collection without a folder is not saved by itself.
func (a *App) workspaceTick() {
	for _, entry := range a.open.Entries {
		if entry.Dir != "" && a.saveOf(entry.Collection).Due(a.collectionKey(entry.Collection), a.now()) {
			a.saveCollection(entry.Collection)
		}
	}
}

// FlushWorkspace saves every collection that changed since its last save,
// for the moment the window closes (the autosave pause may not have
// passed). Nothing is written for a collection without a folder.
func (a *App) FlushWorkspace() error {
	a.commitEdit()
	a.waitWrites()
	var failed []error
	for _, entry := range a.open.Entries {
		if err := a.flushCollection(entry); err != nil {
			failed = append(failed, err)
		}
	}
	return errors.Join(failed...)
}

// flushCollection writes entry's collection now if it changed since its
// last save and has a folder.
func (a *App) flushCollection(entry collections.Entry) error {
	if entry.Dir == "" {
		return nil
	}
	a.waitWrites()
	col := entry.Collection
	if !a.saveOf(col).Pending(a.collectionKeyNow(col)) {
		return nil
	}
	if err := a.store.Save(entry.Dir, *col); err != nil {
		return err
	}
	a.markSaved(col)
	// renaming another collection flushes this one too, and once it is
	// saved no autosave runs again to clear the marks of its tabs
	a.markTabsSaved(col)
	return nil
}

// markTabsSaved makes the requests of col's open tabs, as they are now,
// their saved copies, so their unsaved marks go.
func (a *App) markTabsSaved(col *request.Collection) {
	for _, t := range a.openTabs() {
		if owner, ok := a.entryFor(t.Req); ok && owner.Collection == col {
			t.MarkSaved()
		}
	}
}

// openWorkspaceAt switches to the collection in dir, which deleting the
// last collection does: pending edits of the open ones are saved, they and
// every tab are closed and the folder becomes the one remembered. A folder that cannot be read changes
// nothing and says why. The environment remembered for the folder is
// picked, else the one of that name active before, as when one collection
// was open at a time.
func (a *App) openWorkspaceAt(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	dir = collections.AbsDir(dir)
	if err := a.FlushWorkspace(); err != nil {
		a.toast("Saving the current collection failed: "+err.Error(), severity.Error)
		return
	}
	loaded, err := a.store.Load(dir)
	if err != nil {
		a.toast("Cannot open "+dir+": "+err.Error(), severity.Error)
		return
	}
	env := a.rememberedEnv(dir)
	if env == "" {
		env = a.envName()
	}
	for _, t := range a.openTabs() {
		a.disconnect(t)
	}
	a.commitEdit()
	for _, entry := range a.open.Entries {
		a.forget(entry.Collection)
	}
	a.updateTabs(tabs.Reset{})
	a.selected = request.Node{}
	a.expanded = map[*request.Folder]bool{}
	a.showOpened(a.openLoaded(loaded, dir, env))
	a.status = "Opened " + dir
}

// promptWorkspace asks for the folder of a collection to open beside the
// open ones; an empty or new folder starts a new collection there.
func (a *App) promptWorkspace() {
	entry, _ := a.focus()
	a.prompt("Open collection folder", entry.Dir, "Open", a.openCollectionAt)
}
