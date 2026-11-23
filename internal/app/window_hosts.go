package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
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

// The key-value tables are the kvtable component's; the App answers what it
// asks (kvtable.Deps) and keeps the few rules that need the window.

// tablePanes is the pane that shows each table, by kind: where its rows are
// and what follows a change of them.
var tablePanes = [kvlogic.KindCount]pane{
	kvlogic.Params:  paramsPane,
	kvlogic.Headers: headersPane,
	kvlogic.Vars:    varsPane,
	kvlogic.Form:    formPane,
}

// tablePane is the pane's table in the request pane's padding.
func (a *App) tablePane(panel pane, tab *reqTab, scroll *widget.ScrollState) Node {
	return a.tableView(panel, tab, scroll, w.Gutter)
}

// tableView is the pane's table with the given padding around it.
func (a *App) tableView(panel pane, tab *reqTab, scroll *widget.ScrollState, pad int) Node {
	look := kvtable.Look{Caption: panel.captionOf(a), FlagDuplicates: panel.flagDuplicates}
	return a.kv.View(a.ctx, kvlogic.Table{Tab: tab.Tab, Kind: panel.table}, look, scroll, pad)
}

// commitEdit writes a cell edit in progress into its row, before whatever
// the caller does next could change the rows or the tab.
func (a *App) commitEdit() {
	if a.kv.Commit() {
		a.treeStale = true
	}
}

// deleteRow removes the selected table row, not while text has the keyboard
// (Delete belongs to the text) or the tree does (Delete removes its node).
func (a *App) deleteRow() {
	if a.textFocused() || a.side.Tree().Focused() {
		return
	}
	a.kv.DeleteRow()
}

// kvtableHost is what the key-value tables ask of the App (kvtable.Deps).
type kvtableHost struct{ *App }

// tabOf is the widget state of table's tab. A tab that closed since has none;
// its table is still written (a bulk text typed before the close), through
// a stand-in that only carries the request.
func (host kvtableHost) tabOf(table kvlogic.Table) *reqTab {
	if tab := host.widgets[table.Tab]; tab != nil {
		return tab
	}
	return &reqTab{Tab: table.Tab}
}

// Rows are the rows of table.
func (host kvtableHost) Rows(table kvlogic.Table) *[]request.KV {
	return tablePanes[table.Kind].rows(host.App, host.tabOf(table))
}

// Changed runs the hook of the table's pane: the Params rewrite the URL.
func (host kvtableHost) Changed(table kvlogic.Table) {
	if changed := tablePanes[table.Kind].changed; changed != nil {
		changed(host.App, host.tabOf(table))
	}
}

// IsOpen reports whether tab is one of the open tabs.
func (host kvtableHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Visible is the editable table the request pane shows, if any: Params,
// Headers, Vars, or the form fields of the Body pane.
func (host kvtableHost) Visible() (kvlogic.Table, bool) {
	tab := host.cur()
	shown := requestPanesFor(tab)[tab.paneTab]
	switch {
	case shown.kind == paneTable:
		return kvlogic.Table{Tab: tab.Tab, Kind: shown.table}, true
	case shown.kind == paneBody && (tab.Req.BodyType == request.BodyForm || tab.Req.BodyType == request.BodyMultipart):
		return kvlogic.Table{Tab: tab.Tab, Kind: formPane.table}, true
	}
	return kvlogic.Table{}, false
}

// Open shows a context menu.
func (host kvtableHost) Open(posX, posY int, items []widget.MenuItem) {
	host.openMenu(posX, posY, items)
}

// Secret says whether row of table is a variable of the active environment
// that can be marked as a secret, and whether it is.
func (host kvtableHost) Secret(table kvlogic.Table, row int) (secret, ok bool) {
	env := host.activeEnv()
	if table.Kind != kvlogic.Vars || env == nil || row >= len(env.Vars) {
		return false, false
	}
	return env.Vars[row].Secret, true
}

// ToggleSecret marks row of the active environment's variables as a secret, or clears it.
func (host kvtableHost) ToggleSecret(row int) { host.toggleSecret(row) }

// CellStyler marks the {{variables}} of a cell of req.
func (host kvtableHost) CellStyler(req *request.Request) func(string) []widget.Span {
	vars := host.varsFor(req)
	return func(text string) []widget.Span {
		if !strings.Contains(text, "{{") {
			return nil // a cell without a variable is drawn as it is
		}
		return theme.VarSpans(text, "", vars)
	}
}

// tabstripHost is what the request tab strip asks of the App (tabstrip.Deps).
type tabstripHost struct{ *App }

// Tabs are the open tabs, in the order of the strip.
func (host tabstripHost) Tabs() []*tabs.Tab { return host.tabs.Tabs() }

// Active is the index of the current tab.
func (host tabstripHost) Active() int { return host.tabs.Active() }

// Update applies msg to the open tabs, with what a change of them drags along.
func (host tabstripHost) Update(msg tabs.Msg) { host.updateTabs(msg) }

// Dirty reports whether tab's request differs from its saved copy.
func (host tabstripHost) Dirty(tab *tabs.Tab) bool { return host.tabDirty(host.widgets[tab]) }

// Stripe is the color of the collection holding req.
func (host tabstripHost) Stripe(req *request.Request) w.Token {
	return host.collectionToken(request.Node{Req: req})
}

// OpenTabMenu opens the menu of tab index at the pointer.
func (host tabstripHost) OpenTabMenu(posX, posY, index int) {
	host.openMenu(posX, posY, host.tabMenuItems(index))
}

// NewRequest opens a new request in the selected folder.
func (host tabstripHost) NewRequest() { host.newRequest() }

// Hint is label with the shortcut of action.
func (host tabstripHost) Hint(label, action string) string { return host.hint(label, action) }

// urlbarHost is what the URL bar asks of the App (urlbar.Deps).
type urlbarHost struct{ *App }

// Current is the tab shown.
func (host urlbarHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host urlbarHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Send sends the request of tab; on a socket tab it opens or closes the connection.
func (host urlbarHost) Send(tab *tabs.Tab) { host.send(host.widgets[tab]) }

// Cancel stops the running request of tab.
func (host urlbarHost) Cancel(tab *tabs.Tab) { host.cancelSend(host.widgets[tab]) }

// Sending reports whether the request of tab is in flight.
func (host urlbarHost) Sending(tab *tabs.Tab) bool { return host.sends.Sending(tab.Req) }

// Socket reports whether tab keeps a connection open.
func (host urlbarHost) Socket(tab *tabs.Tab) bool { return isSocket(host.widgets[tab]) }

// ToggleConnection connects a disconnected socket tab and closes a connected one.
func (host urlbarHost) ToggleConnection(tab *tabs.Tab) { host.toggleConnection(host.widgets[tab]) }

// PickMethod lists the methods of the server of tab in a menu.
func (host urlbarHost) PickMethod(tab *tabs.Tab) { host.pickGRPCMethod(host.widgets[tab]) }

// Streaming reports whether tab streams instead of making single calls.
func (host urlbarHost) Streaming(tab *tabs.Tab) bool { return host.widgets[tab].grpcStream }

// ToggleStream switches tab between single calls and a stream.
func (host urlbarHost) ToggleStream(tab *tabs.Tab) { host.toggleStream(host.widgets[tab]) }

// Offered are the protocols the chip cycles through for a request on current.
func (host urlbarHost) Offered(current request.Protocol) []request.Protocol {
	return host.protocolsFor(current)
}

// Choose gives tab the protocol; a connection belongs to the protocol that
// opened it, so it is closed first, and the pane indexes start over.
func (host urlbarHost) Choose(tab *tabs.Tab, protocol request.Protocol) {
	widgets := host.widgets[tab]
	host.disconnect(widgets)
	editRequest(widgets, requestedit.ProtocolChosen{Protocol: protocol})
	widgets.paneTab, widgets.respTab = 0, 0 // the indexes point into the panes of the protocol
}

// Segments colors the {{variables}} of the URL raw of req.
func (host urlbarHost) Segments(req *request.Request, raw string) []widget.Span {
	return host.urlSegments(req, raw)
}

// Tip is what the URL of tab resolves to.
func (host urlbarHost) Tip(tab *tabs.Tab) string { return host.urlTip(host.widgets[tab]) }

// Hint is label with the shortcut of action.
func (host urlbarHost) Hint(label, action string) string { return host.hint(label, action) }

// protocolsFor are the protocols the chip cycles through for a request on
// current: the experimental ones only when settings switch them on, except
// the one the request already uses, so it never loses its protocol.
func (a *App) protocolsFor(current request.Protocol) []request.Protocol {
	var offered []request.Protocol
	for _, view := range protocolViews {
		if view.offered == nil || view.offered(a.settings) || view.protocol == current {
			offered = append(offered, view.protocol)
		}
	}
	return offered
}

// bodyHost is what the body pane asks of the App (body.Deps).
type bodyHost struct{ *App }

// Current is the tab shown.
func (host bodyHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host bodyHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// FormTable is the table of form fields of tab.
func (host bodyHost) FormTable(tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return host.tableView(formPane, host.widgets[tab], scroll, 0)
}

// GraphQL is the editor of a GraphQL body.
func (host bodyHost) GraphQL(tab *tabs.Tab) Node { return host.graphQLBody(host.widgets[tab]) }

// Message is a status line in place of the Schema pane.
func (host bodyHost) Message(text string) Node { return host.messagePane(schemaPane, text, false) }

// VarsFor are the variables req is resolved with.
func (host bodyHost) VarsFor(req *request.Request) []request.KV { return host.varsFor(req) }

// Check asks the disk whether path is a file, off the UI goroutine.
func (host bodyHost) Check(path string, answer func(found bool)) {
	files := host.folders
	host.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		found := files.IsFile(ctx, path)
		return func() { answer(found) } // on the UI goroutine
	})
}

// Pick opens the file chooser.
func (host bodyHost) Pick(title string, answer func(path string)) {
	host.pickFile(title, filedialog.OpenFile, "", answer)
}

// Describe asks for the request message of tab's gRPC method.
func (host bodyHost) Describe(tab *tabs.Tab, answer func(request.MessageSchema, error)) {
	req := tab.Req.Expanded(host.varsFor(tab.Req))
	timeout := host.timeoutFor(req) // read here: the goroutine must not touch App fields
	host.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		schema, err := host.describeGRPCInput(ctx, req)
		return func() { answer(schema, err) } // on the UI goroutine
	})
}

func (a *App) describeGRPCInput(ctx context.Context, req request.Request) (request.MessageSchema, error) {
	if a.grpcSchema == nil {
		return request.MessageSchema{}, errors.New("no gRPC client")
	}
	return a.grpcSchema(ctx, req)
}

// CommitEdit writes a table cell edit in progress into its row.
func (host bodyHost) CommitEdit() { host.commitEdit() }

// ShowBody shows the Body pane of tab.
func (host bodyHost) ShowBody(tab *tabs.Tab) {
	widgets := host.widgets[tab]
	for i, panel := range requestPanesFor(widgets) {
		if panel.kind == paneBody {
			widgets.paneTab = i
		}
	}
}

// SetStatus replaces the status line.
func (host bodyHost) SetStatus(text string) { host.notices.SetStatus(text) }

// authHost is what the auth pane asks of the App (auth.Deps).
type authHost struct{ *App }

// Current is the tab shown.
func (host authHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host authHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// VarsFor are the variables req is resolved with.
func (host authHost) VarsFor(req *request.Request) []request.KV { return host.varsFor(req) }

// Folder is the directory of the collection that owns req, "" for none.
func (host authHost) Folder(req *request.Request) string {
	entry, ok := host.entryFor(req)
	if !ok {
		return ""
	}
	return entry.Dir
}

// Has reports whether the user allowed access.
func (host authHost) Has(access request.Access) bool { return host.trusted.Has(access) }

// Add remembers that the user allowed access.
func (host authHost) Add(access request.Access) { host.trusted.Add(access) }

// Cookies are the cookies the jar holds for u. The jar is read on every
// call because Clear replaces it.
func (host authHost) Cookies(u *url.URL) []*http.Cookie { return host.jar.Cookies(u) }

// Clear replaces the jar with an empty one and rebuilds the protocols that
// hold it, so connections made before keep nothing.
func (host authHost) Clear() error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	host.jar = jar
	host.reg = host.newRegistry()
	return nil
}

// Confirm asks the user to confirm.
func (host authHost) Confirm(title, body, action string, run func()) {
	host.confirm(title, body, action, run)
}

// Toast shows text for a moment.
func (host authHost) Toast(text string, level severity.Level) { host.toast(text, level) }

// assertionsHost is what the assertions component asks of the App
// (assertions.Deps).
type assertionsHost struct{ *App }

// Current is the tab shown.
func (host assertionsHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host assertionsHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// SetVar keeps a captured value as a variable of the request's collection.
func (host assertionsHost) SetVar(req *request.Request, name, value string) {
	host.setVariable(req, name, value)
}

// Status is the status line.
func (host assertionsHost) Status() string { return host.notices.Status() }

// SetStatus replaces the status line.
func (host assertionsHost) SetStatus(text string) { host.notices.SetStatus(text) }

// DefaultTimeout is how long a request without its own timeout may run.
func (host assertionsHost) DefaultTimeout() time.Duration { return host.timeoutFor(request.Request{}) }

// responseHost is what the response component asks of the App
// (response.Deps).
type responseHost struct{ *App }

// Current is the tab shown.
func (host responseHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host responseHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Results is the Tests pane, which the assertions component draws.
func (host responseHost) Results(tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return host.assertions.ResultsView(host.ctx, tab, scroll)
}

// Widths is the column widths of the response headers table, kept with the
// widths of the other key-value tables.
func (host responseHost) Widths() *widget.TableState { return host.kv.Widths(kvlogic.ResponseHeaders) }

// CellStyler marks the {{variables}} of a cell of req.
func (host responseHost) CellStyler(req *request.Request) func(string) []widget.Span {
	return kvtableHost{host.App}.CellStyler(req)
}

// PickSave asks for the path of a file to save.
func (host responseHost) PickSave(title, suggested string, answer func(path string)) {
	host.pickFile(title, filedialog.SaveFile, suggested, answer)
}

// Write saves body at path off the UI goroutine.
func (host responseHost) Write(path string, body []byte, done func(error)) {
	host.writeFile(path, body, done)
}

// Home is the home folder of the user.
func (host responseHost) Home() (string, error) { return host.folders.Home() }

// Curl is the cURL command of the request of tab with its variables filled in,
// "" when the app has no formatter.
func (host responseHost) Curl(tab *tabs.Tab) string {
	if host.curl == nil {
		return ""
	}
	return host.curl(tab.Req.Expanded(host.varsFor(tab.Req)))
}

// Preview serves body for the browser to open.
func (host responseHost) Preview(ctx context.Context, body []byte, contentType string) error {
	return host.previews.Show(ctx, body, contentType)
}

// CommitEdit writes a table cell edit in progress into its row.
func (host responseHost) CommitEdit() { host.commitEdit() }

// ShowResponse selects the Response pane of tab.
func (host responseHost) ShowResponse(tab *tabs.Tab) { host.widgets[tab].respTab = 0 }

// Open shows a context menu at the pointer position.
func (host responseHost) Open(posX, posY int, items []widget.MenuItem) {
	host.openMenu(posX, posY, items)
}

// Hint is label with the shortcut of action.
func (host responseHost) Hint(label, action string) string { return host.hint(label, action) }

// Shortcut is the key of action, "" when it has none.
func (host responseHost) Shortcut(action string) string { return host.shortcutHint(action) }

// ResponseView is the setting that picks the mode of a new response view.
func (host responseHost) ResponseView() string { return host.settings.ResponseView }

// Now is the time of the frame.
func (host responseHost) Now() time.Time { return host.now() }
