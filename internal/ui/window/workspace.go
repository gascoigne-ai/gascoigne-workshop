package window

import (
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
)

// openWorkspace opens the collection in dir, or, when dir is empty, the
// collections remembered in the settings (an older file's single workspace
// is a set of one). With none of them, the build's client config, if it
// has one, replaces the built-in sample. A folder that cannot be read is
// left out and named in the status line, so a later save cannot
// overwrite what is there.
func (a *Component) openWorkspace(dir string) {
	wanted, origin := a.settings.RememberedCollections(), collections.OriginRemembered
	if dir != "" {
		wanted = []request.OpenCollection{{Folder: dir, Environment: a.cols.RememberedEnv(collections.AbsDir(dir))}}
		origin = collections.OriginStartFolder
	}
	var started, sample *request.Collection
	failures := a.cols.Restore(wanted, origin, func(string) *request.Collection {
		var isSample bool
		if started, isSample = a.startCollection(); isSample {
			sample = started
		}
		a.expandStart(started, isSample)
		return started
	})
	var failed []error
	for _, failure := range failures {
		failed = append(failed, failure.Err)
		a.toastNewer(failure.Folder, failure.Err)
	}
	if len(failed) > 0 {
		a.notices.SetStatus("Workspace not loaded: " + errors.Join(failed...).Error())
	}
	for _, entry := range a.cols.Set().Entries {
		if entry.Collection != started { // the start collection has its own expansion
			a.side.Tree().ExpandAll(&entry.Collection.Folder)
		}
	}
	first := a.cols.Set().Entries[0].Collection
	if _, start, ok := collection.SampleStart(first); ok && first == sample { // the sample opens on its search call
		a.openRequest(start)
	} else {
		a.openRequest(collections.FirstOrBlank(first))
	}
	for _, entry := range a.cols.Set().Entries { // a blank request added to an empty collection is not an edit
		a.markSaved(entry.Collection)
	}
}

// toastNewer tells, beyond the status line, that the collection in folder
// was not opened because a newer build saved it.
func (a *Component) toastNewer(folder string, err error) {
	var newer *storage.NewerError
	if errors.As(err, &newer) {
		a.toast("Cannot open "+folder+": "+err.Error(), severity.Error)
	}
}

// expandStart opens the folders of the collection the program starts with
// when nothing saved is open: the client config's, else the sample. The
// sample opens on its search call, the config with every folder open.
func (a *Component) expandStart(col *request.Collection, isSample bool) {
	if folder, _, ok := collection.SampleStart(col); ok && isSample {
		a.side.Tree().Expand(&col.Folder)
		a.side.Tree().Expand(folder)
		return
	}
	a.side.Tree().ExpandAll(&col.Folder)
}

// startCollection is the client config's tree, or the sample when there is
// no config, it is empty, or it cannot be read (the status then says why).
func (a *Component) startCollection() (col *request.Collection, isSample bool) {
	if a.clients != nil {
		config, err := a.clients.Load()
		if err != nil {
			a.notices.SetStatus("Client config not loaded: " + err.Error())
		} else if !config.Empty() {
			configured := config.Collection()
			return &configured, false
		}
	}
	sample := startCollection(a.sample)
	return &sample, true
}

// saveWorkspace writes the focused collection now (Ctrl+S).
func (a *Component) saveWorkspace() { a.saveCollection(a.focusCol()) }

// saveCollection writes col to its folder. A collection that has none yet
// gets its own new folder in the collections folder, as a new collection
// does, and the open set with it is remembered.
func (a *Component) saveCollection(col *request.Collection) {
	a.commitEdit()
	entry, ok := a.cols.Set().ByCollection(col)
	if !ok {
		return
	}
	dir := entry.Dir
	if dir == "" {
		var err error
		if dir, err = a.cols.Locate(col); err != nil {
			a.toastError(err)
			return
		}
	}
	a.treeStale = true // the tabs of col are marked saved
	a.markSaved(col)   // an error is reported once, not retried every frame
	a.markTabsSaved(col)
	a.saves.Write(dir, col, func(name string, err error) {
		if err != nil {
			a.toast("Saving failed: "+err.Error(), severity.Error)
			return
		}
		a.notices.SetStatus(fmt.Sprintf("Saved %s", name))
	})
}

// workspaceTick saves each saved collection once it has been unchanged for
// saveDebounce. A collection without a folder is not saved by itself.
func (a *Component) workspaceTick() {
	for _, entry := range a.cols.Set().Entries {
		if entry.Dir != "" && a.saves.Due(entry.Collection, a.keys.Collection(entry.Collection), a.now()) {
			a.saveCollection(entry.Collection)
		}
	}
}

// FlushWorkspace saves every collection that changed since its last save,
// for the moment the window closes (the autosave pause may not have
// passed). Nothing is written for a collection without a folder.
func (a *Component) FlushWorkspace() error {
	a.commitEdit()
	a.writes.Wait()
	var failed []error
	for _, entry := range a.cols.Set().Entries {
		if err := a.flushCollection(entry); err != nil {
			failed = append(failed, err)
		}
	}
	return errors.Join(failed...)
}

// flushCollection writes entry's collection now if it changed since its
// last save and has a folder.
func (a *Component) flushCollection(entry collections.Entry) error {
	if entry.Dir == "" {
		return nil
	}
	a.writes.Wait()
	col := entry.Collection
	key := a.keys.CollectionNow(col)
	if !a.saves.Pending(col, key) {
		return nil
	}
	if err := a.saves.WriteNow(entry.Dir, col, key); err != nil {
		return err
	}
	// renaming another collection flushes this one too, and once it is
	// saved no autosave runs again to clear the marks of its tabs
	a.markTabsSaved(col)
	return nil
}

// markTabsSaved makes the requests of col's open tabs, as they are now,
// their saved copies, so their unsaved marks go.
func (a *Component) markTabsSaved(col *request.Collection) {
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
func (a *Component) openWorkspaceAt(dir string) {
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
	env := a.cols.RememberedEnv(dir)
	if env == "" {
		env = a.envName()
	}
	for _, t := range a.openTabs() {
		a.disconnect(t)
	}
	a.commitEdit()
	for _, entry := range a.cols.Set().Entries {
		a.forget(entry.Collection)
		a.cols.Apply(collections.Close{Collection: entry.Collection})
	}
	a.updateTabs(tabs.Reset{})
	a.side.Tree().Reset()
	show := a.cols.Reopen(loaded, dir, env)
	a.side.Tree().ExpandAll(&a.cols.Set().Entries[0].Collection.Folder)
	a.openRequest(show)
	a.markSaved(a.cols.Set().Entries[0].Collection) // as in addCollection
	a.notices.SetStatus("Opened " + dir)
}

// openCollectionAt opens the collection in dir beside the open ones (an
// empty or new folder starts a new collection there), or focuses it when
// it is open already. A folder that cannot be read changes nothing and
// says why.
func (a *Component) openCollectionAt(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	if err := a.addCollection(dir); err != nil {
		a.toast("Cannot open "+dir+": "+err.Error(), severity.Error)
		return
	}
	a.notices.SetStatus("Opened " + collections.AbsDir(dir))
}

// addCollection opens the collection saved in dir after the open ones and
// shows it; a folder already open is focused, one that cannot be read is
// not opened.
func (a *Component) addCollection(dir string) error {
	opening, err := a.cols.Open(dir)
	if err != nil {
		return err
	}
	if opening.Loaded {
		a.side.Tree().ExpandAll(&opening.Collection.Folder)
	}
	a.openRequest(opening.Show)
	if opening.Loaded {
		a.markSaved(opening.Collection) // showing the request changes what the collection looks like, not what is on disk
	}
	return nil
}

// closeCollection takes col out of the window: its edits are saved first,
// its tabs close and the set left is remembered. Closing the last one
// leaves an empty tree; nothing is made on disk.
func (a *Component) closeCollection(col *request.Collection) {
	entry, ok := a.cols.Set().ByCollection(col)
	if !ok {
		return
	}
	a.commitEdit()
	if err := a.flushCollection(entry); err != nil {
		a.toast("Saving "+col.Name+" failed: "+err.Error(), severity.Error)
		return
	}
	for _, open := range a.openTabs() {
		if col.Folder.Contains(request.Node{Req: open.Req}) { // no tab is left to close its socket later
			a.disconnect(open)
		}
	}
	a.closeTabsIn(request.Node{Dir: &col.Folder})
	a.forget(col)
	a.cols.Close(col)
}

// forget drops everything that pointed into col, which is leaving the open
// set.
func (a *Component) forget(col *request.Collection) {
	if a.varsTarget.refersTo(col) {
		a.varsTarget = varsTarget{}
	}
	if col.Folder.Contains(a.side.Tree().Selected()) {
		a.side.Tree().Select(request.Node{})
	}
	a.saves.Forget(col)
	a.keys.MarkStale()
}

// The open collections: a.cols holds them in tree order, each with the
// folder it is saved in and its own active environment. Whatever acts on a
// request, a folder or a tab acts on the collection that holds it; what has
// no node of its own (the header, the environment chip, Ctrl+S) acts on the
// focused one.

// focus is the collection the header, the environment chip and the
// collection commands act on; ok is false when nothing is open.
func (a *Component) focus() (collections.Entry, bool) {
	var current *request.Request
	if len(a.tabs.Tabs()) > 0 {
		current = a.cur().Req
	}
	return a.cols.Set().Focus(current, a.side.Tree().Selected())
}

// focusCol is the focused collection, nil when nothing is open.
func (a *Component) focusCol() *request.Collection {
	entry, _ := a.focus()
	return entry.Collection
}

// headerTitle is the name of the focused collection, "" when none is open.
func (a *Component) headerTitle() string {
	if col := a.focusCol(); col != nil {
		return col.Name
	}
	return ""
}

// entryFor is the collection that holds req. A request of no collection,
// such as the blank tab left after closing the last one, goes with the
// focused collection, as it did when one collection was open at a time.
func (a *Component) entryFor(req *request.Request) (collections.Entry, bool) {
	if entry, ok := a.cols.Set().Owner(request.Node{Req: req}); ok {
		return entry, true
	}
	return a.focus()
}

// eachRoot calls visit with the root folder of every open collection, in
// tree order.
func (a *Component) eachRoot(visit func(root *request.Folder)) {
	for _, entry := range a.cols.Set().Entries {
		visit(&entry.Collection.Folder)
	}
}

// markSaved records that col is on disk as it is now.
func (a *Component) markSaved(col *request.Collection) { a.saves.Saved(col, a.keys.CollectionNow(col)) }

// librarySettings is where the open set keeps its remembered list: the
// settings (collections.Settings).
type librarySettings struct{ *Component }

func (host librarySettings) Remembered() []request.OpenCollection {
	return host.settings.RememberedCollections()
}

func (host librarySettings) Remember(list []request.OpenCollection) {
	host.saveSettings(func(prefs *settings.Settings) { prefs.RememberCollections(list) })
}

// librarySaves is the autosave bookkeeping the open set asks of the App
// (collections.Saves).
type librarySaves struct{ *Component }

func (host librarySaves) Saved(col *request.Collection) { host.markSaved(col) }
func (host librarySaves) FlushAll() error               { return host.FlushWorkspace() }

// toastError shows err, which the collections package words in lower case,
// as an error toast.
func (a *Component) toastError(err error) {
	text := err.Error()
	first, size := utf8.DecodeRuneInString(text)
	a.toast(string(unicode.ToUpper(first))+text[size:], severity.Error)
}
