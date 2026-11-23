package window

import (
	"context"
	"errors"
	"fmt"
	envs "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/environments"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"image/color"
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
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/outbox"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/menus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response/respview"
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
func (a *Component) Size() (width, height int) { return a.w, a.h }

// SentStore, Toast (below), SetStatus, OpenRequest and Badge make the App
// the host of the sent-request log.
func (a *Component) SentStore() history.SentStore {
	store := a.historyStore()
	if store == nil {
		return nil
	}
	return store.Sent()
}

// Toast shows a message over the window.
func (a *Component) Toast(text string, level severity.Level) { a.toast(text, level) }

// SetStatus replaces the status line.
func (a *Component) SetStatus(text string) { a.notices.SetStatus(text) }

// OpenRequest focuses the request at path of the collection saved in
// folder and reports whether there is one. A row without a folder is of a
// collection not saved, or older than the folder in a row, when the one
// collection open was the first.
func (a *Component) OpenRequest(folder, path string) bool {
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
func (a *Component) sentCandidates(folder string) []collections.Entry {
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
func (a *Component) Badge(protocol request.Protocol, method string) Node {
	return Text(theme.RequestBadge(protocol, method), Small(), Color(theme.MethodToken(method)), W(sizes.BadgeSlot(a.ctx, sizes.BadgeW)))
}

// sidebarHost is what the sidebar asks of the App (sidebar.Deps). The store
// and window work stays in the App; the sidebar only decides and draws. A
// second type, collectionHost, answers CollectionOps, whose Rename and Delete
// take a collection where Nodes' take a node.
type sidebarHost struct{ *Component }

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
type collectionHost struct{ *Component }

func (host collectionHost) Rename(col *request.Collection)       { host.promptRenameCollection(col) }
func (host collectionHost) Move(col *request.Collection, by int) { host.cols.Shift(col, by) }
func (host collectionHost) Close(col *request.Collection)        { host.requestCloseCollection(col) }
func (host collectionHost) Delete(col *request.Collection)       { host.confirmDeleteCollection(col) }

// The prompts and confirmations of whole collections: new, rename, open a
// folder, delete and close. They stay here and not in collections.Library
// because the dialogs are the overlays' and the Library returns an
// error worded for the user, which the window shows.

// promptNewCollection asks for a name, then makes the collection and opens
// it beside the open ones.
func (a *Component) promptNewCollection() {
	a.prompt("New collection", "", "Create", a.createCollection)
}

func (a *Component) createCollection(text string) {
	dir, err := a.cols.Create(text)
	if err != nil {
		a.toastError(err)
		return
	}
	a.openCollectionAt(dir)
}

// promptRenameCollection asks for the new name of col.
func (a *Component) promptRenameCollection(col *request.Collection) {
	if !a.onDisk(col, "rename") {
		return
	}
	a.prompt("Rename collection", col.Name, "Rename", func(text string) { a.renameCollection(col, text) })
}

func (a *Component) renameCollection(col *request.Collection, text string) {
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
func (a *Component) confirmDeleteCollection(col *request.Collection) {
	if !a.onDisk(col, "delete") {
		return
	}
	title, body := collections.DeletePrompt(col.Name)
	a.confirm(title, body, "Delete", func() { a.deleteCollection(col) })
}

// deleteCollection removes col from the disk and from the window. When it
// was the only one open, the next collection of the collections folder
// opens, or a new empty one when none is left.
func (a *Component) deleteCollection(col *request.Collection) {
	a.commitEdit()
	a.writes.Wait() // a write still in flight would make the folder again
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
func (a *Component) onDisk(col *request.Collection, verb string) bool {
	if entry, ok := a.cols.Set().ByCollection(col); ok && entry.Dir != "" {
		return true
	}
	a.toast("The collection is not saved yet: save it first to "+verb+" it", severity.Error)
	return false
}

// hint is a tooltip with the action's shortcut, when it has one.
func (a *Component) hint(text, shortcutID string) string {
	if key := a.shortcutHint(shortcutID); key != "" {
		return text + " (" + key + ")"
	}
	return text
}

// shortcutHint is the key of an action for a menu item: nothing when unbound.
func (a *Component) shortcutHint(id string) string { return shortcutpage.Hint(a.over.Settings, id) }

// promptWorkspace asks for the folder of a collection to open beside the
// open ones; an empty or new folder starts a new collection there.
func (a *Component) promptWorkspace() {
	entry, _ := a.focus()
	a.prompt("Open collection folder", entry.Dir, "Open", a.openCollectionAt)
}

// requestCloseCollection closes col, first asking when closing would lose
// edits: a collection with a folder is written as it closes, as a closed
// tab keeps its edits in the collection, but one without a folder keeps
// them nowhere, so it asks the way deleting does.
func (a *Component) requestCloseCollection(col *request.Collection) {
	pending := a.saves.Pending(col, a.keys.CollectionNow(col))
	if !a.cols.AsksToClose(col, pending) {
		a.closeCollection(col)
		return
	}
	title, body := collections.ClosePrompt(col.Name)
	a.confirm(title, body, "Close", func() { a.closeCollection(col) })
}

// closeFocusedCollection closes the focused collection (the close command).
func (a *Component) closeFocusedCollection() {
	if col := a.focusCol(); col != nil {
		a.requestCloseCollection(col)
	}
}

// The key-value tables are the kvtable component's; the App answers what it
// asks (kvtable.Deps) and keeps the few rules that need the window.

// commitEdit writes a cell edit in progress into its row, before whatever
// the caller does next could change the rows or the tab.
func (a *Component) commitEdit() {
	if a.editor.Tables.Commit() {
		a.treeStale = true
	}
}

// deleteRow removes the selected table row, not while text has the keyboard
// (Delete belongs to the text) or the tree does (Delete removes its node).
func (a *Component) deleteRow() {
	if a.typing() || a.side.Tree().Focused() {
		return
	}
	a.editor.Tables.DeleteRow()
}

// kvtableHost is what the key-value tables ask of the App (kvtable.Deps),
// beyond the rows and the tabs, which the editor's pane table answers.
type kvtableHost struct{ *Component }

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
type tabstripHost struct{ *Component }

// Tabs are the open tabs, in the order of the strip.
func (host tabstripHost) Tabs() []*tabs.Tab { return host.tabs.Tabs() }

// Active is the index of the current tab.
func (host tabstripHost) Active() int { return host.tabs.Active() }

// Update applies msg to the open tabs, with what a change of them drags along.
func (host tabstripHost) Update(msg tabs.Msg) { host.updateTabs(msg) }

// Dirty reports whether tab's request differs from its saved copy.
func (host tabstripHost) Dirty(tab *tabs.Tab) bool { return host.keys.TabDirty(tab) }

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

// headerHost is what the banner asks of the App (header.Deps).
type headerHost struct{ *Component }

// Title is the name of the focused collection, "" when none is open.
func (host headerHost) Title() string { return host.headerTitle() }

// Toggle opens the settings dialog, or closes it.
func (host headerHost) Toggle() { host.over.Settings.Toggle() }

// Hint is label with the shortcut of action.
func (host headerHost) Hint(label, action string) string { return host.hint(label, action) }

// envchipHost is what the environment chip asks of the App (envchip.Deps).
type envchipHost struct{ *Component }

// Focused is the focused collection and its environment choice.
func (host envchipHost) Focused() (*request.Collection, envs.Selection, bool) {
	return host.focusedSelection()
}

// Select makes environment index the active one of the focused collection.
func (host envchipHost) Select(index int) { host.selectEnv(index) }

// Add creates an environment in the focused collection and makes it active.
func (host envchipHost) Add(name string) { host.addEnv(name) }

// Rename gives environment index of the focused collection a new name.
func (host envchipHost) Rename(index int, name string) { host.renameEnv(index, name) }

// Delete removes environment index from the focused collection.
func (host envchipHost) Delete(index int) { host.deleteEnv(index) }

// Open shows the chip's menu at the pointer.
func (host envchipHost) Open(posX, posY int, items []widget.MenuItem) {
	host.openMenu(posX, posY, items)
}

// Prompt asks for a line of text.
func (host envchipHost) Prompt(title, initial, okLabel string, onOK func(text string)) {
	host.prompt(title, initial, okLabel, onOK)
}

// Confirm asks a yes/no question.
func (host envchipHost) Confirm(title, body, okLabel string, onOK func()) {
	host.confirm(title, body, okLabel, onOK)
}

// urlbarHost is what the URL bar asks of the App (urlbar.Deps).
type urlbarHost struct{ *Component }

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
func (host urlbarHost) Socket(tab *tabs.Tab) bool { return host.editor.Socket(tab) }

// ToggleConnection connects a disconnected socket tab and closes a connected one.
func (host urlbarHost) ToggleConnection(tab *tabs.Tab) { host.toggleConnection(host.widgets[tab]) }

// PickMethod lists the methods of the server of tab in a menu.
func (host urlbarHost) PickMethod(tab *tabs.Tab) { host.editor.GRPC.PickMethod(host.ctx, tab) }

// Streaming reports whether tab streams instead of making single calls.
func (host urlbarHost) Streaming(tab *tabs.Tab) bool { return host.editor.GRPC.Streaming(tab) }

// ToggleStream switches tab between single calls and a stream.
func (host urlbarHost) ToggleStream(tab *tabs.Tab) { host.editor.GRPC.ToggleStream(tab) }

// Segments colors the {{variables}} of the URL raw of req.
func (host urlbarHost) Segments(req *request.Request, raw string) []widget.Span {
	return host.urlSegments(req, raw)
}

// Tip is what the URL of tab resolves to.
func (host urlbarHost) Tip(tab *tabs.Tab) string { return host.urlTip(host.widgets[tab]) }

// Hint is label with the shortcut of action.
func (host urlbarHost) Hint(label, action string) string { return host.hint(label, action) }

// editorHost is what the editor asks of the App beyond what each of its
// components asks: the tab shown, the connection of a tab whose protocol
// changes, the settings and the Vars pane's table.
type editorHost struct{ *Component }

// Current is the tab shown.
func (host editorHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host editorHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Disconnect closes the connection of tab.
func (host editorHost) Disconnect(tab *tabs.Tab) { host.disconnect(host.widgets[tab]) }

// ProtocolChosen starts the response side of tab over: its indexes point
// into the panes of the protocol.
func (host editorHost) ProtocolChosen(tab *tabs.Tab) { host.widgets[tab].respTab = 0 }

// Settings are the settings in effect.
func (host editorHost) Settings() settings.Settings { return host.settings }

// Rows are the rows the Vars pane edits.
func (host editorHost) Rows() *[]request.KV { return host.varsRows() }

// Caption is the caption above them.
func (host editorHost) Caption() string { return host.varsCaption() }

// bodyHost is what the body pane asks of the App (body.Deps).
type bodyHost struct{ *Component }

// Current is the tab shown.
func (host bodyHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host bodyHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

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

func (a *Component) describeGRPCInput(ctx context.Context, req request.Request) (request.MessageSchema, error) {
	if a.grpcSchema == nil {
		return request.MessageSchema{}, errors.New("no gRPC client")
	}
	return a.grpcSchema(ctx, req)
}

// CommitEdit writes a table cell edit in progress into its row.
func (host bodyHost) CommitEdit() { host.commitEdit() }

// SetStatus replaces the status line.
func (host bodyHost) SetStatus(text string) { host.notices.SetStatus(text) }

// graphqlHost is what the GraphQL component asks of the App (graphql.Deps).
type graphqlHost struct{ *Component }

// Current is the tab shown.
func (host graphqlHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host graphqlHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Dispatch sends req as the request of tab and shows the answer as its response.
func (host graphqlHost) Dispatch(tab *tabs.Tab, req request.Request) {
	host.dispatch(host.widgets[tab], &req)
}

// Open shows a menu of items at the position.
func (host graphqlHost) Open(posX, posY int, items []widget.MenuItem) {
	host.openMenu(posX, posY, items)
}

// grpcHost is what the gRPC component asks of the App (grpc.Deps).
type grpcHost struct{ *Component }

// IsOpen reports whether tab is one of the open tabs.
func (host grpcHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// List asks the gRPC client for the methods of req in the background.
func (host grpcHost) List(req request.Request, answer func(methods []string, err error)) {
	timeout := host.timeoutFor(req) // read here: the goroutine must not touch App fields
	host.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		methods, err := host.listGRPCMethods(ctx, req)
		return func() { answer(methods, err) } // on the UI goroutine
	})
}

func (a *Component) listGRPCMethods(ctx context.Context, req request.Request) ([]string, error) {
	if a.grpcMethods == nil {
		return nil, errors.New("no gRPC client")
	}
	return a.grpcMethods(ctx, req)
}

// VarsFor are the variables req is resolved with.
func (host grpcHost) VarsFor(req *request.Request) []request.KV { return host.varsFor(req) }

// Folder is the directory of the collection that owns req, "" for none.
func (host grpcHost) Folder(req *request.Request) string {
	entry, ok := host.entryFor(req)
	if !ok {
		return ""
	}
	return entry.Dir
}

// Pick opens the file chooser.
func (host grpcHost) Pick(title string, answer func(path string)) {
	host.pickFile(title, filedialog.OpenFile, "", answer)
}

// Disconnect closes the open stream of tab or abandons the connect in progress.
func (host grpcHost) Disconnect(tab *tabs.Tab) { host.disconnect(host.widgets[tab]) }

// Open shows a menu of items at the position.
func (host grpcHost) Open(posX, posY int, items []widget.MenuItem) { host.openMenu(posX, posY, items) }

// Toast shows text for a moment.
func (host grpcHost) Toast(text string, level severity.Level) { host.toast(text, level) }

// SetStatus replaces the status line.
func (host grpcHost) SetStatus(text string) { host.notices.SetStatus(text) }

// authHost is what the auth pane asks of the App (auth.Deps).
type authHost struct{ *Component }

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
func (host authHost) Has(access request.Access) bool { return host.stateFile.Trusts(access) }

// Add remembers that the user allowed access.
func (host authHost) Add(access request.Access) { host.stateFile.Trust(access) }

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
type assertionsHost struct{ *Component }

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
type responseHost struct{ *Component }

// Current is the tab shown.
func (host responseHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host responseHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Results is the Tests pane, which the assertions component draws.
func (host responseHost) Results(tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return host.editor.Assertions.ResultsView(host.ctx, tab, scroll)
}

// Widths is the column widths of the response headers table, kept with the
// widths of the other key-value tables.
func (host responseHost) Widths() *widget.TableState {
	return host.editor.Tables.Widths(kvlogic.ResponseHeaders)
}

// CellStyler marks the {{variables}} of a cell of req.
func (host responseHost) CellStyler(req *request.Request) func(string) []widget.Span {
	return kvtableHost{host.Component}.CellStyler(req)
}

// PickSave asks for the path of a file to save.
func (host responseHost) PickSave(title, suggested string, answer func(path string)) {
	host.pickFile(title, filedialog.SaveFile, suggested, answer)
}

// writeFile writes data to a file the user named through the Folders
// port, off the UI goroutine; done gets the outcome on it. Writes are
// queued by path, so two saves to one file never interleave.
func (a *Component) writeFile(path string, data []byte, done func(error)) {
	files := a.folders
	a.writes.Write("file "+path, serial.Job{
		Run:  func() error { return files.Write(context.Background(), path, data) },
		Done: done,
	})
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

// mcpHost is what the MCP component asks of the App (mcp.Deps).
type mcpHost struct{ *Component }

// IsOpen reports whether tab is one of the open tabs.
func (host mcpHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Run does work in the background and runs the answer it returns on the UI
// goroutine.
func (host mcpHost) Run(work func() (answer func())) {
	host.bus.Run(context.Background(), func(context.Context) bus.Msg {
		if answer := work(); answer != nil {
			return answer
		}
		return nil
	})
}

// Dispatch runs run on the UI goroutine.
func (host mcpHost) Dispatch(run func()) { host.bus.Dispatch(run) }

// SetComposer replaces the text of the composer of tab.
func (host mcpHost) SetComposer(tab *tabs.Tab, text string) { host.editor.WS.SetComposer(tab, text) }

// ComposeCall puts the call of tool into the composer of tab.
func (host mcpHost) ComposeCall(tab *tabs.Tab, tool request.MCPTool) {
	host.editor.WS.ComposeCall(tab, tool)
}

// Open shows state as the dialog of the window, with content as its own
// body when it has one. The returned func closes that dialog and no other.
func (host mcpHost) Open(state widget.DialogState, content func(ctx *w.Ctx) Node) func() {
	return host.over.Open(state, content, 0)
}

// Busy reports whether a dialog is open.
func (host mcpHost) Busy() bool { return host.over.Busy() }

// StateText is what the explorer says of a connection in state, the words
// of the connection bar.
func (host mcpHost) StateText(state tabs.ConnState) string { return ws.StateText(state) }

// CodeLines are the colored lines of a JSON answer, as the response shows
// them.
func (host mcpHost) CodeLines(body []byte) []widget.Line { return respview.CodeLines(body) }

// RawLines are the plain lines of an answer that is not JSON.
func (host mcpHost) RawLines(body []byte) []widget.Line { return respview.RawLines(body) }

// wsHost is what the Messages pane asks of the App (ws.Deps).
type wsHost struct{ *Component }

// Current is the tab shown.
func (host wsHost) Current() *tabs.Tab { return host.tabs.Current() }

// IsOpen reports whether tab is one of the open tabs.
func (host wsHost) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.tabs.Tabs(), tab) }

// Queue sends in order with the other sends of tab; a failure is noted in
// its log.
func (host wsHost) Queue(tab *tabs.Tab, send outbox.Send) { host.sendInOrder(host.widgets[tab], send) }

// OpenToolForm shows the form of tool; its answer fills the composer of tab.
func (host wsHost) OpenToolForm(tab *tabs.Tab, tool request.MCPTool) {
	host.editor.MCP.OpenToolForm(tab, tool)
}

// Open shows a menu at the pointer.
func (host wsHost) Open(posX, posY int, items []widget.MenuItem) { host.openMenu(posX, posY, items) }

// Toast shows a message over the window.
func (host wsHost) Toast(text string, level severity.Level) { host.toast(text, level) }

// settingsHost is what the settings dialog asks of the App
// (settingsdialog.Deps). The settings stay on the App, since the theme, the
// open tabs and the protocols follow from them; the dialog decides when
// they are written.
type settingsHost struct{ *Component }

// Settings are the settings in effect.
func (host settingsHost) Settings() settings.Settings { return host.settings }

// Edit changes the settings in effect and what follows from the change: the
// tabs already open, the protocols and the theme. The error says why the
// theme could not be built.
func (host settingsHost) Edit(edit func(*settings.Settings)) error {
	wasNetwork, wasView := host.networkSettings(), host.settings.ResponseView
	edit(&host.settings)
	if host.settings.ResponseView != wasView { // the choice applies to the tabs already open
		host.setResponseMode()
	}
	if host.networkSettings() != wasNetwork {
		host.reg = host.newRegistry() // protocols hold the proxy and certificates they were built with
	}
	return host.ctx.SetTheme(theme.ThemeWith(host.settings))
}

// Apply makes current the settings in effect, with everything that follows
// from them.
func (host settingsHost) Apply(current settings.Settings) {
	host.settings = current
	if err := host.ctx.SetTheme(theme.ThemeWith(current)); err != nil {
		host.notices.SetStatus(err.Error())
	}
	host.setResponseMode()
	host.dirty = true
}

// setResponseMode gives every open tab the response view of the settings.
func (a *Component) setResponseMode() {
	for _, t := range a.openTabs() {
		a.resp.SetMode(t.Tab, viewer.ModeOf(a.settings.ResponseView))
	}
}

// SetScale sets the interface scale while its slider is dragged.
func (host settingsHost) SetScale(percent int) { host.settings.UIScalePct, host.dirty = percent, true }

// Pointer is the pointer position in the window.
func (host settingsHost) Pointer() (int, int) { return host.ctx.In.X, host.ctx.In.Y }

// Toast shows a message over the window.
func (host settingsHost) Toast(text string, level severity.Level) { host.toast(text, level) }

// SetStatus writes the status line.
func (host settingsHost) SetStatus(text string) { host.notices.SetStatus(text) }

// OpenMenu shows a menu of items at a point of the window.
func (host settingsHost) OpenMenu(posX, posY int, items []widget.MenuItem) {
	host.openMenu(posX, posY, items)
}

// Stale says the tree built this frame no longer shows what changed.
func (host settingsHost) Stale() { host.treeStale = true }

// Redraw says the screen must be drawn again.
func (host settingsHost) Redraw() { host.dirty = true }

// Pick asks for a file and calls done with its path.
func (host settingsHost) Pick(title string, done func(path string)) (cancel func()) {
	return host.pickFile(title, filedialog.OpenFile, "", done)
}

// Open shows state as the dialog of the window. The returned func closes
// that dialog and no other.
func (host settingsHost) Open(state widget.DialogState) (close func()) {
	return host.over.Open(state, nil, 0)
}

// Dispatch runs run on the UI goroutine.
func (host settingsHost) Dispatch(run func()) { host.bus.Dispatch(run) }

// Color is the color in effect for a token.
func (host settingsHost) Color(token w.Token) color.RGBA { return host.ctx.Th.Color(token) }

// SetFonts switches the interface and code fonts.
func (host settingsHost) SetFonts(ui, code w.FontData) error { return host.ctx.SetFonts(ui, code) }

// HostOS is the operating system shortcut text follows.
func (host settingsHost) HostOS() string { return host.ctx.HostOS }

// CopyText puts text on the clipboard.
func (host settingsHost) CopyText(text string) { _ = host.ctx.Clipboard.Write(text) } // the clipboard reports its own failures

// Keymap is the binding of every shortcut action, in display order.
func (host settingsHost) Keymap() keymap.Keymap { return host.keymap() }

// ---- the overlays ------------------------------------------------------------------

// toast shows a message: an info one fades after a few seconds, an error
// stays until it is clicked.
func (a *Component) toast(text string, level severity.Level) { a.over.Toasts.Show(text, level) }

// confirm asks a yes/no question; onOK runs only on the primary button or Enter.
func (a *Component) confirm(title, body, okLabel string, onOK func()) {
	a.over.Confirm(title, body, okLabel, onOK)
}

// prompt asks for a line of text, starting with initial selected.
func (a *Component) prompt(title, initial, okLabel string, onOK func(text string)) {
	a.over.Prompt(title, initial, okLabel, onOK)
}

// dialogDefault runs the primary button: what Enter does.
func (a *Component) dialogDefault() { a.over.Default() }

// openMenu shows a context menu at the pointer position x, y.
func (a *Component) openMenu(posX, posY int, items []widget.MenuItem) {
	menus.Open(a.ctx, posX, posY, items)
}

func (a *Component) closeMenu() { menus.Close(a.ctx) }

// pickFile asks for a file, in the desktop's dialog or the in-app browser,
// and calls done with the path. The returned func takes the question back.
func (a *Component) pickFile(title string, mode filedialog.Mode, name string, done func(path string)) (cancel func()) {
	return a.over.Browser.Pick(title, mode, name, done)
}

// OpenWelcome shows the release notes and remembers that this version was
// welcomed, so the dialog opens by itself only once per version.
func (a *Component) OpenWelcome() {
	a.stateFile.Welcome(version)
	a.over.Welcome.Open()
}

// clipboardNote reports what the clipboard did: failures are error
// toasts, a successful copy is a status line.
func (a *Component) clipboardNote(msg string, isErr bool) {
	if isErr {
		a.toast(msg, severity.Error)
		return
	}
	a.notices.SetStatus(msg)
}

// overlaysHost is what the overlays ask of the App (overlays.Deps). The
// desktop dialog and the file system are read from the App on every call, so
// a port replaced after the App was built is the one that answers.
type overlaysHost struct{ *Component }

// Size is the window's size in logical pixels.
func (host overlaysHost) Size() (int, int) { return host.w, host.h }

// Stale says the tree built this frame no longer shows what changed.
func (host overlaysHost) Stale() { host.treeStale = true }

// Redraw says the screen must be drawn again.
func (host overlaysHost) Redraw() { host.dirty = true }

// Now is the time of the frame.
func (host overlaysHost) Now() time.Time { return host.now() }

// Available reports whether the desktop has a file dialog.
func (host overlaysHost) Available() bool { return host.files.Available() }

// Suggested is the program to install for native dialogs.
func (host overlaysHost) Suggested() string { return host.files.Suggested() }

// Choose shows the desktop's file dialog.
func (host overlaysHost) Choose(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error) {
	return host.files.Choose(ctx, title, mode, name)
}

// Home is the user's home folder.
func (host overlaysHost) Home() (string, error) { return host.folders.Home() }

// List is what dir holds.
func (host overlaysHost) List(dir string) ([]workspace.FolderEntry, error) {
	return host.folders.List(dir)
}

// Run starts cmd on its own goroutine; its answer runs on the UI goroutine.
func (host overlaysHost) Run(ctx context.Context, cmd bus.Cmd) { host.bus.Run(ctx, cmd) }

// The settings the dialog edits stay on the App (the theme, the open tabs and
// the protocols follow from them): these are the App's views onto its dialog.

// changeSettings is saveSettings for the settings dialog's own controls (see
// settingsdialog.Component.Change).
func (a *Component) changeSettings(edit func(*settings.Settings)) { a.over.Settings.Change(edit) }

// saveSettings edits and applies the settings and, unless auto-save is off,
// writes them (see settingsdialog.Component.Edit).
func (a *Component) saveSettings(edit func(*settings.Settings)) { a.over.Settings.Edit(edit) }

// SettingsPath is the settings file, "" when settings are not kept.
func (a *Component) SettingsPath() string { return a.over.Settings.SettingsPath() }

// networkSettings are the settings the protocols are built from.
type networkSettings struct{ proxy, caCert, clientCert, clientKey string }

func (a *Component) networkSettings() networkSettings {
	s := a.settings
	return networkSettings{s.Proxy, s.CACert, s.ClientCert, s.ClientKey}
}

// keysHost answers autosave.Sources: it builds the comparable forms the
// change-key cache keeps.
type keysHost struct{ *Component }

// Collections are the open collections.
func (host keysHost) Collections() []*request.Collection {
	entries := host.cols.Set().Entries
	open := make([]*request.Collection, len(entries))
	for i, entry := range entries {
		open[i] = entry.Collection
	}
	return open
}

// CollectionKey is the comparable form of col.
func (host keysHost) CollectionKey(col request.Collection) string { return tree.Key(col) }

// StateKey is the comparable form of the state to save.
func (host keysHost) StateKey() string { return host.stateFile.Key() }

// Dirty reports whether tab's request differs from its saved copy.
func (host keysHost) Dirty(tab *tabs.Tab) bool { return tab.Dirty() }

// stateHost answers autosave.Window and autosave.Reports: it reads the
// window's part of the remembered state off the UI and applies a remembered
// one to a freshly built app. Why it stays here and not in a component:
// what it reads (sizes, tabs, open folders, table widths) belongs to
// several components and the App's own tab widgets; revisit when those
// have one owner.
type stateHost struct{ *Component }

// Snapshot is the window's part of the state to remember.
func (host stateHost) Snapshot() state.State {
	snapshot := state.State{
		Window:    state.Window{W: host.windowPx(host.w), H: host.windowPx(host.h), X: host.winX, Y: host.winY},
		SidebarW:  host.sidebarSplit.Size,
		PaneSplit: host.paneSplit.Size,
		SettingsW: host.over.Settings.Shell.Size.W,
		SettingsH: host.over.Settings.Shell.Size.H,
	}
	expanded := host.side.Tree().Expanded()
	for _, entry := range host.cols.Set().Entries {
		snapshot.RememberExpanded(entry.Dir, host.cols.Set().ExpandedPaths(entry, func(folder *request.Folder) bool { return expanded[folder] }))
	}
	open := host.openTabs()
	remembered := make([]state.Tab, len(open))
	for i, tab := range open {
		owner, path, _ := host.cols.Set().PathOf(tab.Req)
		if path == "" {
			continue // not in the tree: Remember drops it
		}
		remembered[i] = state.Tab{Collection: owner.Dir, Path: path, Pane: host.editor.PaneOf(tab.Tab), PaneLabel: host.editor.PaneLabel(tab.Tab), Resp: tab.respTab}
	}
	snapshot.Tabs, snapshot.Active = state.Remember(remembered, host.tabs.Active())
	for kind := kvlogic.Params; kind < kvlogic.KindCount; kind++ {
		if weights := host.editor.Tables.Widths(kind).Weights; len(weights) > 0 {
			if snapshot.Columns == nil {
				snapshot.Columns = map[string][]float64{}
			}
			snapshot.Columns[kind.Key()] = append([]float64(nil), weights...)
		}
	}
	return snapshot
}

// Restore applies a remembered state to a freshly built app. Anything that
// no longer exists is skipped, and values are kept in range.
func (host stateHost) Restore(saved state.State) {
	sizes := saved.Sizes()
	host.over.Settings.Shell.Size.W, host.over.Settings.Shell.Size.H = sizes.SettingsW, sizes.SettingsH
	host.sidebarSplit.Size = sizes.SidebarW
	if sizes.HasPaneSplit {
		host.paneSplit.Size = sizes.PaneSplit
	}
	for dir, paths := range saved.ExpandedByCollection() {
		host.expandPaths(host.remembered(dir), paths)
	}
	reopened, active := saved.Reopen(func(tab state.Tab) *request.Request {
		return host.requestAt(host.remembered(tab.Collection), tab.Path)
	})
	if len(reopened) > 0 {
		host.updateTabs(tabs.Reset{})
		for _, again := range reopened {
			host.updateTabs(tabs.Open{Req: again.Req})
			tab := host.cur()
			labels := tabLabels(host.editor.PaneTabs(tab.Tab))
			host.editor.ShowPane(tab.Tab, again.Tab.PaneIn(labels, host.editor.PaneOf(tab.Tab)))
			if again.Tab.Resp >= 0 && again.Tab.Resp < len(host.responsePanesFor(tab)) {
				tab.respTab = again.Tab.Resp
			}
		}
		host.updateTabs(tabs.Activate{Index: active})
		host.side.Tree().Select(request.Node{Req: host.cur().Req})
	}
	for name, weights := range saved.Columns {
		if len(weights) == 0 {
			continue
		}
		if kind, ok := kvlogic.KindOfKey(name); ok {
			*host.editor.Tables.Widths(kind) = widget.TableState{Weights: append([]float64(nil), weights...)}
		}
	}
}

// Failed shows that the state could not be written.
func (host stateHost) Failed(err error) { host.toast(err.Error(), severity.Error) }

// ReadOnly shows that a file a newer build wrote is left as it is.
func (host stateHost) ReadOnly(err error) { host.noteReadOnly(storage.ReadOnly, err) }

// remembered is the open collection a state reference names, nil for none.
func (a *Component) remembered(ref string) *request.Collection {
	at := state.CollectionAt(ref, a.cols.Set().Dirs())
	if at < 0 {
		return nil
	}
	return a.cols.Set().Entries[at].Collection
}

// expandPaths opens the folders of col at the remembered paths.
func (a *Component) expandPaths(col *request.Collection, paths []string) {
	for _, folder := range a.cols.Set().FoldersAt(col, paths) {
		a.side.Tree().Expand(folder)
	}
}

// requestAt is the request of col at path, nil when there is none.
func (a *Component) requestAt(col *request.Collection, path string) *request.Request {
	if col == nil {
		return nil
	}
	return tree.At(&col.Folder, path)
}

// tabLabels are the labels of the tabs of a pane's strip.
func tabLabels(items []widget.TabItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

// windowPx converts a logical size to window pixels (device independent),
// which is what the window system and the remembered geometry use.
func (a *Component) windowPx(logical int) int {
	scale := a.windowScale
	if scale == 0 {
		scale = a.settings.UIScale() // before the first Layout
	}
	return int(float64(logical)*scale + 0.5)
}
