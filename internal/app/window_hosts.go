package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar"
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
		entry, ok := a.open.ByDir(folder)
		if !ok {
			return nil
		}
		return []collections.Entry{entry}
	}
	var candidates []collections.Entry
	for _, entry := range a.open.Entries {
		if entry.Dir == "" {
			candidates = append(candidates, entry)
		}
	}
	if len(a.open.Entries) > 0 {
		candidates = append(candidates, a.open.Entries[0])
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
func (host sidebarHost) Set() collections.Set { return host.open }

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

// importKinds maps what the sidebar names to the importer that reads it.
var importKinds = map[sidebar.ImportKind]importKind{
	sidebar.ImportCurl:    curlKind,
	sidebar.ImportOpenAPI: openAPIKind,
	sidebar.ImportPostman: postmanKind,
}

// Import adds what the clipboard holds, read as kind, to dir.
func (host sidebarHost) Import(dir *request.Folder, kind sidebar.ImportKind) {
	host.importFolder(dir, importKinds[kind])
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

func (host collectionHost) Rename(col *request.Collection) { host.promptRenameCollection(col) }
func (host collectionHost) Move(col *request.Collection, by int) {
	host.moveCollection(col, by)
}
func (host collectionHost) Close(col *request.Collection)  { host.requestCloseCollection(col) }
func (host collectionHost) Delete(col *request.Collection) { host.confirmDeleteCollection(col) }
