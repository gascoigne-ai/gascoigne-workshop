package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Size is the window's size in logical pixels; with the methods below it
// makes the App the host of the features that live in their own packages.
func (a *App) Size() (width, height int) { return a.w, a.h }

// SentStore, Toast (below), SetStatus, OpenRequest and Badge make the App
// the host of the sent-request log.
func (a *App) SentStore() history.SentStore {
	store := a.historyStore()
	if store == nil {
		return nil
	}
	return store.Sent()
}

// Toast shows a message over the window.
func (a *App) Toast(text string, level severity.Level) { a.toast(text, level) }

// SetStatus replaces the status line.
func (a *App) SetStatus(text string) { a.notices.SetStatus(text) }

// OpenRequest focuses the request at path of the collection saved in
// folder and reports whether there is one. A row without a folder is of a
// collection not saved, or older than the folder in a row, when the one
// collection open was the first.
func (a *App) OpenRequest(folder, path string) bool {
	for _, entry := range a.sentCandidates(folder) {
		if req := tree.At(&entry.Collection.Folder, path); req != nil {
			a.openRequest(req)
			return true
		}
	}
	return false
}

// sentCandidates are the open collections a sent row of folder may name,
// in the order they are searched.
func (a *App) sentCandidates(folder string) []collections.Entry {
	if folder != "" {
		entry, ok := a.cols.Set().ByDir(folder)
		if !ok {
			return nil
		}
		return []collections.Entry{entry}
	}
	var candidates []collections.Entry
	for _, entry := range a.cols.Set().Entries {
		if entry.Dir == "" {
			candidates = append(candidates, entry)
		}
	}
	if len(a.cols.Set().Entries) > 0 {
		candidates = append(candidates, a.cols.Set().Entries[0])
	}
	return candidates
}

// Badge is the method label shown before a request's name.
func (a *App) Badge(protocol request.Protocol, method string) Node {
	return Text(theme.RequestBadge(protocol, method), Small(), Color(theme.MethodToken(method)), W(sizes.BadgeSlot(a.ctx, sizes.BadgeW)))
}

// sidebarHost is what the sidebar asks of the App (sidebar.Deps). The store
// and window work stays in the App; the sidebar only decides and draws. A
// second type, collectionHost, answers CollectionOps, whose Rename and Delete
// take a collection where Nodes' take a node.
type sidebarHost struct{ *App }

// Set is the open collections.
func (host sidebarHost) Set() collections.Set { return host.cols.Set() }

// Focused is the collection the chip menu acts on.
func (host sidebarHost) Focused() *request.Collection { return host.focusCol() }

// Show opens req in a tab, as a click on its row does.
func (host sidebarHost) Show(req *request.Request) { host.openRequest(req) }

func (host sidebarHost) Rename(node request.Node)    { host.renameNode(node) }
func (host sidebarHost) Delete(node request.Node)    { host.deleteNode(node) }
func (host sidebarHost) Duplicate(node request.Node) { host.duplicateNode(node) }

// Pick opens the destination picker for node.
func (host sidebarHost) Pick(node request.Node, mode transfer.Mode) {
	host.openTransferPicker(node, mode)
}

// Drop moves or copies a node that was dropped on another row.
func (host sidebarHost) Drop(move transfer.Transfer) { host.transferNode(move) }

func (host sidebarHost) NewRequest(dir *request.Folder) { host.newRequestIn(dir) }
func (host sidebarHost) NewFolder(dir *request.Folder)  { host.newFolderIn(dir) }
func (host sidebarHost) Export(dir *request.Folder)     { host.exportPostman(dir) }

// Import adds what the clipboard holds, read as kind, to dir and opens it;
// a text of another kind is explained in a toast. Reading the text is the
// job of logic.Importers and putting it in the tree that of Imported.Attach;
// only the clipboard, the toasts and the tab are the window's.
func (host sidebarHost) Import(dir *request.Folder, kind logic.ImportKind) {
	text, err := host.ctx.Clipboard.Read()
	if err != nil {
		host.toast("Cannot read the clipboard: "+err.Error(), severity.Error)
		return
	}
	importer := logic.Importers[kind]
	got, err := importer.Parse(text)
	if err != nil {
		host.toast(importer.Refusal+err.Error(), severity.Error)
		return
	}
	name, show := got.Attach(host.side.Tree(), dir, host.ownerOf(request.Node{Dir: dir}))
	if show != nil {
		host.openRequest(show)
	}
	host.notices.SetStatus("Imported " + name)
	if skipped := got.Skipped(); len(skipped) > 0 {
		host.toast(fmt.Sprintf("%d %s, first: %s", len(skipped), importer.SkippedWhat, skipped[0]), severity.Error)
	}
}

func (host sidebarHost) CopyURL(req *request.Request) { host.copyRequestURL(req) }
func (host sidebarHost) EditVars(dir *request.Folder) { host.editVars(dir) }

func (host sidebarHost) NewCollection()               { host.promptNewCollection() }
func (host sidebarHost) OpenFolder()                  { host.promptWorkspace() }
func (host sidebarHost) Save(col *request.Collection) { host.saveCollection(col) }
func (host sidebarHost) Open(posX, posY int, items []widget.MenuItem) {
	host.openMenu(posX, posY, items)
}

// collectionHost is the sidebar's CollectionOps.
type collectionHost struct{ *App }

func (host collectionHost) Rename(col *request.Collection)       { host.promptRenameCollection(col) }
func (host collectionHost) Move(col *request.Collection, by int) { host.cols.Shift(col, by) }
func (host collectionHost) Close(col *request.Collection)        { host.requestCloseCollection(col) }
func (host collectionHost) Delete(col *request.Collection)       { host.confirmDeleteCollection(col) }

// The prompts and confirmations of whole collections: new, rename, open a
// folder, delete and close. They stay here and not in collections.Library
// because the dialogs are overlays of the window, not components yet; the
// Library returns an error worded for the user and the window shows it.

// promptNewCollection asks for a name, then makes the collection and opens
// it beside the open ones.
func (a *App) promptNewCollection() {
	a.prompt("New collection", "", "Create", a.createCollection)
}

func (a *App) createCollection(text string) {
	dir, err := a.cols.Create(text)
	if err != nil {
		a.toastError(err)
		return
	}
	a.openCollectionAt(dir)
}

// promptRenameCollection asks for the new name of col.
func (a *App) promptRenameCollection(col *request.Collection) {
	if !a.onDisk(col, "rename") {
		return
	}
	a.prompt("Rename collection", col.Name, "Rename", func(text string) { a.renameCollection(col, text) })
}

func (a *App) renameCollection(col *request.Collection, text string) {
	name, err := a.cols.Rename(col, text)
	if errors.Is(err, collections.ErrNotOpen) {
		return
	}
	if err != nil {
		a.toastError(err)
		return
	}
	a.notices.SetStatus("Renamed to " + name)
}

// confirmDeleteCollection asks, then deletes col.
func (a *App) confirmDeleteCollection(col *request.Collection) {
	if !a.onDisk(col, "delete") {
		return
	}
	title, body := collections.DeletePrompt(col.Name)
	a.confirm(title, body, "Delete", func() { a.deleteCollection(col) })
}

// deleteCollection removes col from the disk and from the window. When it
// was the only one open, the next collection of the collections folder
// opens, or a new empty one when none is left.
func (a *App) deleteCollection(col *request.Collection) {
	a.commitEdit()
	a.WaitWrites() // a write still in flight would make the folder again
	entry, ok := a.cols.Set().ByCollection(col)
	if !ok {
		return
	}
	next, err := a.cols.Delete(col)
	if errors.Is(err, collections.ErrNotOpen) {
		return
	}
	if err != nil {
		a.toastError(err)
		return
	}
	if next == "" {
		a.closeCollection(col)
	} else {
		a.openWorkspaceAt(next)
	}
	a.toast("Deleted "+filepath.Base(entry.Dir), severity.Info)
}

// onDisk is true when col has a folder; otherwise it tells the user to
// save first, since there is nothing to rename or delete.
func (a *App) onDisk(col *request.Collection, verb string) bool {
	if entry, ok := a.cols.Set().ByCollection(col); ok && entry.Dir != "" {
		return true
	}
	a.toast("The collection is not saved yet: save it first to "+verb+" it", severity.Error)
	return false
}

// promptWorkspace asks for the folder of a collection to open beside the
// open ones; an empty or new folder starts a new collection there.
func (a *App) promptWorkspace() {
	entry, _ := a.focus()
	a.prompt("Open collection folder", entry.Dir, "Open", a.openCollectionAt)
}

// requestCloseCollection closes col, first asking when closing would lose
// edits: a collection with a folder is written as it closes, as a closed
// tab keeps its edits in the collection, but one without a folder keeps
// them nowhere, so it asks the way deleting does.
func (a *App) requestCloseCollection(col *request.Collection) {
	pending := a.saveOf(col).Pending(a.collectionKeyNow(col))
	if !a.cols.AsksToClose(col, pending) {
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
