package app

import (
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/environment/controller/envs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/autosave"
)

// The open collections: a.open holds them in tree order, each with the
// folder it is saved in and its own active environment (the rules are in
// collection/controller/collections). Whatever acts on a request, a folder
// or a tab acts on the collection that holds it; what has no node of its
// own (the header, the environment chip, Ctrl+S) acts on the focused one.

// focus is the collection the header, the environment chip and the
// collection commands act on: the one holding the active tab's request,
// else the tree selection, else the first open one. ok is false when
// nothing is open.
func (a *App) focus() (collections.Entry, bool) {
	if len(a.list.Tabs) > 0 {
		if entry, ok := a.open.Owner(model.Of(a.cur().Req)); ok {
			return entry, true
		}
	}
	if entry, ok := a.open.Owner(a.selected); ok {
		return entry, true
	}
	if len(a.open.Entries) == 0 {
		return collections.Entry{}, false
	}
	return a.open.Entries[0], true
}

// focusCol is the focused collection, nil when nothing is open.
func (a *App) focusCol() *model.Collection {
	entry, _ := a.focus()
	return entry.Collection
}

// headerTitle is the name of the focused collection, "" when none is open.
func (a *App) headerTitle() string {
	if col := a.focusCol(); col != nil {
		return col.Name
	}
	return ""
}

// entryFor is the collection that holds req. A request of no collection,
// such as the blank tab left after closing the last one, goes with the
// focused collection, as it did when one collection was open at a time.
func (a *App) entryFor(req *model.Request) (collections.Entry, bool) {
	if entry, ok := a.open.Owner(model.Of(req)); ok {
		return entry, true
	}
	return a.focus()
}

// isRoot reports whether dir is the root folder of an open collection,
// which is renamed and deleted as a collection, never moved.
func (a *App) isRoot(dir *model.Folder) bool {
	for _, entry := range a.open.Entries {
		if dir == &entry.Collection.Folder {
			return true
		}
	}
	return false
}

// eachRoot calls visit with the root folder of every open collection, in
// tree order.
func (a *App) eachRoot(visit func(root *model.Folder)) {
	for _, entry := range a.open.Entries {
		visit(&entry.Collection.Folder)
	}
}

// saveOf is when col was last written.
func (a *App) saveOf(col *model.Collection) *autosave.Debounce {
	debounce := a.saves[col]
	if debounce == nil {
		fresh := autosave.New(saveDebounce, "")
		debounce = &fresh
		a.saves[col] = debounce
	}
	return debounce
}

// markSaved records that col is on disk as it is now.
func (a *App) markSaved(col *model.Collection) { a.saveOf(col).Saved(a.collectionKeyNow(col)) }

// addCollection opens the collection saved in dir after the open ones. A
// folder already open changes nothing and is focused; one that cannot be
// read is not opened. The new one gets the environment remembered for its
// folder.
func (a *App) addCollection(dir string) error {
	dir = collections.AbsDir(dir)
	if entry, ok := a.open.ByDir(dir); ok {
		// its edits stay pending, and the remembered list stays as it is,
		// since the set did not change
		a.focusCollection(entry.Collection)
		return nil
	}
	col, err := a.loadCollection(dir, a.rememberedEnv(dir))
	if err != nil {
		return err
	}
	a.showOpened(col)
	return nil
}

// showOpened focuses a collection just opened and remembers the open set.
func (a *App) showOpened(col *model.Collection) {
	a.focusCollection(col)
	a.markSaved(col) // a blank request added to an empty collection is not an edit
	a.rememberCollections()
}

// loadCollection opens the collection in dir with the environment called
// env active, or gives back the one already open there.
func (a *App) loadCollection(dir, env string) (*model.Collection, error) {
	if entry, ok := a.open.ByDir(dir); ok {
		return entry.Collection, nil
	}
	loaded, err := a.store.Load(dir)
	if err != nil {
		return nil, err
	}
	return a.openLoaded(loaded, dir, env), nil
}

// openLoaded opens a collection read from dir, with its folders expanded.
func (a *App) openLoaded(loaded model.Collection, dir, env string) *model.Collection {
	col := &loaded
	a.expandAll(&col.Folder)
	a.include(col, dir, env)
	return col
}

// include puts col, saved in dir ("" for not saved), into the open set
// with the environment called env active, and counts it as saved.
func (a *App) include(col *model.Collection, dir, env string) {
	a.open = collections.Update(a.open, collections.Open{Collection: col, Dir: dir})
	selection := envs.None()
	selection.Pick(col, env)
	a.open = collections.Update(a.open, collections.ChooseEnv{Collection: col, Index: selection.Index})
	a.markSaved(col)
}

// focusCollection shows col: its first request, or a new blank one in its
// root when it has none.
func (a *App) focusCollection(col *model.Collection) {
	first := tree.FirstRequest(&col.Folder)
	if first == nil {
		first = col.AddRequest(model.Request{Name: "New Request", Protocol: model.ProtocolREST, Method: "GET"})
	}
	a.openRequest(first)
}

// openCollectionAt opens the collection in dir beside the open ones (an
// empty or new folder starts a new collection there), or focuses it when
// it is open already. A folder that cannot be read changes nothing and
// says why.
func (a *App) openCollectionAt(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	if err := a.addCollection(dir); err != nil {
		a.toast("Cannot open "+dir+": "+err.Error(), true)
		return
	}
	a.status = "Opened " + collections.AbsDir(dir)
}

// requestCloseCollection closes col, first asking when closing would lose
// edits: a collection with a folder is written as it closes, as a closed
// tab keeps its edits in the collection, but one without a folder keeps
// them nowhere, so it asks the way deleting does.
func (a *App) requestCloseCollection(col *model.Collection) {
	entry, ok := a.open.ByCollection(col)
	if !ok {
		return
	}
	if entry.Dir != "" || !a.saveOf(col).Pending(a.collectionKeyNow(col)) {
		a.closeCollection(col)
		return
	}
	title, body := collections.ClosePrompt(col.Name)
	a.confirm(title, body, "Close", func() { a.closeCollection(col) })
}

// closeFocusedCollection closes the focused collection (the close command).
func (a *App) closeFocusedCollection() {
	if col := a.focusCol(); col != nil {
		a.requestCloseCollection(col)
	}
}

// moveCollection moves col one place up the tree (by -1) or down (by 1);
// at an end it stays.
func (a *App) moveCollection(col *model.Collection, by int) {
	moved := collections.Update(a.open, collections.Shift{Collection: col, By: by})
	if slices.Equal(moved.Entries, a.open.Entries) {
		return
	}
	a.open = moved
	a.rememberCollections()
}

// openIndex is the place of col in the tree, -1 when it is not open.
func (a *App) openIndex(col *model.Collection) int {
	return slices.IndexFunc(a.open.Entries, func(entry collections.Entry) bool { return entry.Collection == col })
}

// closeCollection takes col out of the window: its edits are saved first,
// its tabs close and the set left is remembered. Closing the last one
// leaves an empty tree; nothing is made on disk.
func (a *App) closeCollection(col *model.Collection) {
	entry, ok := a.open.ByCollection(col)
	if !ok {
		return
	}
	a.commitEdit()
	if err := a.flushCollection(entry); err != nil {
		a.toast("Saving "+col.Name+" failed: "+err.Error(), true)
		return
	}
	for _, open := range a.openTabs() {
		if col.Folder.Contains(model.Of(open.Req)) { // no tab is left to close its socket later
			a.disconnect(open)
		}
	}
	a.closeTabsIn(model.Of(&col.Folder))
	a.forget(col)
	a.rememberCollections()
}

// forget drops col from the open set and from everything that pointed
// into it.
func (a *App) forget(col *model.Collection) {
	if a.varsTarget.refersTo(col) {
		a.varsTarget = varsTarget{}
	}
	if col.Folder.Contains(a.selected) {
		a.selected = model.Node{}
	}
	a.open = collections.Update(a.open, collections.Close{Collection: col})
	delete(a.saves, col)
	a.markStale()
}

// rememberCollections writes the open set to the settings, which then no
// longer carry the older single workspace and environment.
func (a *App) rememberCollections() {
	remembered := collections.Remembered(a.open, a.unreadable)
	a.saveSettings(func(prefs *settings.Settings) { prefs.RememberCollections(remembered) })
	a.setIsRemembered = true
}

// rememberEnvironments writes the active environment of each open
// collection to the settings. When the open set did not come from the
// remembered list (a folder given at start with -workspace), only the
// entries of the list that name an open collection change: the list is
// what the next ordinary start opens and such a run must not replace it.
func (a *App) rememberEnvironments() {
	if a.setIsRemembered {
		a.rememberCollections()
		return
	}
	remembered, changed := collections.WithEnvironments(a.open, a.settings.RememberedCollections())
	if changed {
		a.saveSettings(func(prefs *settings.Settings) { prefs.RememberCollections(remembered) })
	}
}

// rememberedEnv is the environment remembered for the collection in dir.
func (a *App) rememberedEnv(dir string) string {
	for _, item := range a.settings.RememberedCollections() {
		if item.Folder == dir || (item.Folder != "" && collections.AbsDir(item.Folder) == dir) {
			return item.Environment
		}
	}
	return ""
}
