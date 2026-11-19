package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/ui/destpicker"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/openapi"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/postman"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// ---- what the tree shows -------------------------------------------------

// visibleRows are the tree rows shown, after the filter.
func (a *App) visibleRows() []collection.Row {
	var rows []collection.Row
	a.eachRoot(func(root *model.Folder) {
		rows = append(rows, collection.FilteredRows(root, a.expanded, a.filter.Text)...)
	})
	return rows
}

// ---- selection and the rows' menus --------------------------------------

// selectNode marks a folder or request as the one the tree acts on.
func (a *App) selectNode(node model.Node) { a.selected = node }

// collectionToken is the color of the open collection that holds node, ""
// for a node of none.
func (a *App) collectionToken(node model.Node) core.Token {
	for index, entry := range a.open.Entries {
		if entry.Collection.Folder.Contains(node) {
			return appearance.CollectionToken(index)
		}
	}
	return ""
}

// revealRequest selects req in the tree and opens the folders above it.
func (a *App) revealRequest(req *model.Request) {
	a.selectNode(model.Of(req))
	owner, ok := a.open.Owner(model.Of(req))
	if !ok {
		return
	}
	root := &owner.Collection.Folder
	a.expanded[root] = true
	for folder, found := root.Parent(req); found && folder != root; folder, found = root.Parent(folder) {
		a.expanded[folder] = true
	}
}

// selectedFolder is where a new request goes: the selected folder, the
// folder of the selected request, or the root of the focused collection;
// nil when nothing is open.
func (a *App) selectedFolder() *model.Folder {
	switch {
	case a.selected.Dir != nil:
		return a.selected.Dir
	case a.selected.Req != nil:
		if owner, ok := a.open.Owner(a.selected); ok {
			if parent, ok := owner.Collection.Folder.Parent(a.selected.Req); ok {
				return parent
			}
		}
	}
	if col := a.focusCol(); col != nil {
		return &col.Folder
	}
	return nil
}

// treeMenuItems are the right-click actions for a row.
func (a *App) treeMenuItems(row collection.Row) []MenuItem {
	node := row.Node()
	rename := MenuItem{Label: "Rename", Run: func() { a.renameNode(node) }}
	sep := MenuItem{Separator: true}
	duplicate := MenuItem{Label: "Duplicate", Run: func() { a.duplicateNode(node) }}
	moveTo := MenuItem{Label: "Move to...", Run: func() { a.openTransferPicker(node, transfer.Move) }}
	copyTo := MenuItem{Label: "Copy to...", Run: func() { a.openTransferPicker(node, transfer.Copy) }}
	if row.Req != nil {
		return []MenuItem{
			rename, duplicate, moveTo, copyTo,
			{Label: "Copy URL", Run: func() { a.copyRequestURL(row.Req) }},
			sep,
			{Label: "Delete", Run: func() { a.deleteNode(node) }},
		}
	}
	create := []MenuItem{
		{Label: "New request", Run: func() { a.newRequestIn(row.Dir) }},
		{Label: "New folder", Run: func() { a.newFolderIn(row.Dir) }},
		{Label: "Import cURL from clipboard", Run: func() { a.importFolder(row.Dir, curlKind) }},
		{Label: "Import OpenAPI from clipboard", Run: func() { a.importFolder(row.Dir, openAPIKind) }},
		{Label: "Import Postman from clipboard", Run: func() { a.importFolder(row.Dir, postmanKind) }},
		{Label: "Export as Postman...", Run: func() { a.exportPostman(row.Dir) }},
		{Label: "Variables", Run: func() { a.editVars(row.Dir) }},
		sep,
		rename,
	}
	if a.isRoot(row.Dir) {
		return append(create[:len(create)-1], a.rootMenuItems(row.Dir)...)
	}
	return append(create, duplicate, moveTo, copyTo,
		MenuItem{Label: "Delete", Run: func() { a.deleteNode(node) }},
	)
}

// ---- operations ----------------------------------------------------------

// rootMenuItems end the menu of a collection's root row: the collection
// itself is renamed, moved, closed and deleted as a collection, not as a
// folder. Reordering is by menu rather than by drag, since a drag moves or
// copies a node into a collection.
func (a *App) rootMenuItems(root *model.Folder) []MenuItem {
	owner, _ := a.open.Owner(model.Of(root))
	col := owner.Collection
	at := a.openIndex(col)
	return []MenuItem{
		{Label: "Rename collection...", Run: func() { a.promptRenameCollection(col) }},
		{Label: "Move up", Disabled: at == 0, Run: func() { a.moveCollection(col, -1) }},
		{Label: "Move down", Disabled: at == len(a.open.Entries)-1, Run: func() { a.moveCollection(col, 1) }},
		menuSeparator,
		{Label: "New collection...", Run: a.promptNewCollection},
		{Label: "Open collection folder...", Run: a.promptWorkspace},
		menuSeparator,
		{Label: "Close collection", Run: func() { a.requestCloseCollection(col) }},
		{Label: "Delete collection...", Run: func() { a.confirmDeleteCollection(col) }},
	}
}

// treeOwner is the collection a tree message changes: the one holding its
// node or folder; nil when no open collection holds it.
func (a *App) treeOwner(msg tree.Msg) *model.Collection {
	var subject model.Node
	switch msg := msg.(type) {
	case tree.Rename:
		subject = msg.Node
	case tree.Duplicate:
		subject = msg.Node
	case tree.Delete:
		subject = msg.Node
	case tree.NewRequest:
		subject = model.Of(msg.Dir) // none, not a nil *Folder, when there is no folder
	case tree.NewFolder:
		subject = model.Of(msg.Dir)
	}
	return a.ownerOf(subject)
}

// ownerOf is the open collection that holds node, nil for none.
func (a *App) ownerOf(node model.Node) *model.Collection {
	if node.None() {
		return nil
	}
	entry, _ := a.open.Owner(node)
	return entry.Collection
}

// applyTree runs a tree message against the collection that holds its
// node and takes over what it says: the selection, the folder to open,
// the request to show.
func (a *App) applyTree(msg tree.Msg) tree.Result {
	col := a.treeOwner(msg)
	if col == nil {
		return tree.Result{Selected: a.selected}
	}
	res := tree.Update(col, a.selected, msg)
	a.selected = res.Selected
	if res.Expand != nil {
		a.expanded[res.Expand] = true
	}
	if res.Open != nil {
		a.openRequest(res.Open)
	}
	if !res.Removed.None() {
		a.closeTabsIn(res.Removed)
	}
	return res
}

// newRequestIn adds a blank request to dir and opens it.
func (a *App) newRequestIn(dir *model.Folder) { a.applyTree(tree.NewRequest{Dir: dir}) }

func (a *App) newFolderIn(dir *model.Folder) { a.applyTree(tree.NewFolder{Dir: dir}) }

// renameNode asks for a new name and applies it; a refused name is
// explained in a toast.
func (a *App) renameNode(node model.Node) {
	a.prompt("Rename", node.Name(), "Rename", func(text string) {
		if err := a.applyTree(tree.Rename{Node: node, Name: text}).Err; err != nil {
			a.toast("Cannot rename: "+err.Error(), true)
		}
	})
}

func (a *App) duplicateNode(node model.Node) { a.applyTree(tree.Duplicate{Node: node}) }

// deleteNode asks, then removes the node and closes the tabs of
// everything that was in it.
func (a *App) deleteNode(node model.Node) {
	if node.None() || a.isRoot(node.Dir) {
		return
	}
	title, body := tree.DeletePrompt(node)
	a.confirm(title, body, "Delete", func() { a.applyTree(tree.Delete{Node: node}) })
}

// closeTabsIn closes the tabs whose request is node or inside it.
func (a *App) closeTabsIn(node model.Node) {
	for i := len(a.list.Tabs) - 1; i >= 0; i-- {
		if tree.Inside(node, a.list.Tabs[i].Req) {
			a.closeTab(i)
		}
	}
}

func (a *App) copyRequestURL(req *model.Request) {
	_ = a.ctx.Clipboard.Write(model.Expand(req.URL, a.varsFor(req)))
}

// ---- drag and drop ---------------------------------------------------------

// finishDrag drops the dragged node on the row it was released over: it
// moves there, or is copied when Alt is held at the release.
func (a *App) finishDrag() {
	node, target, pressed := a.dragNode, a.dropTarget, a.pressedFolder
	a.dragNode, a.dropTarget, a.pressedFolder = model.Node{}, model.Node{}, nil
	if pressed != nil && target == model.Of(pressed) { // a click on a folder row
		a.expanded[pressed] = !a.expanded[pressed]
		return
	}
	from, to := a.ownerOf(node), a.ownerOf(target)
	if from == nil || to == nil || target == node {
		return
	}
	mode := transfer.Move
	if a.ctx.In.Alt {
		mode = transfer.Copy
	}
	a.transferNode(transfer.Drop(from, to, node, target, mode))
}

// transferHost is the destination picker's view of the App: where a pick
// goes and what closes it.
type transferHost struct {
	*App
	node model.Node
	mode transfer.Mode
}

// Pick closes the picker and moves or copies the node to the destination.
func (host transferHost) Pick(destination transfer.Destination) {
	host.picker = nil
	host.transferNode(transfer.Transfer{
		Node: host.node, From: host.ownerOf(host.node),
		To: destination.Collection, Into: destination.Folder, Mode: host.mode,
	})
}

// Dismiss closes the picker.
func (host transferHost) Dismiss() { host.picker = nil }

// openTransferPicker asks where node goes. A collection's root and nothing
// cannot be moved or copied, so no picker opens for them.
func (a *App) openTransferPicker(node model.Node, mode transfer.Mode) {
	if node.None() || a.isRoot(node.Dir) || a.ownerOf(node) == nil {
		return
	}
	stripe := func(col *model.Collection) core.Token { return a.collectionToken(model.Of(&col.Folder)) }
	a.picker = destpicker.New(node, mode, transfer.Destinations(a.open, node, mode), stripe)
	a.pickerHost = transferHost{App: a, node: node, mode: mode}
}

// pickerKeys hands the picker the frame's keys and takes them from the
// input, so the field under it does not act on them too.
func (a *App) pickerKeys(in core.Input) {
	in.Enter = in.Enter || in.Pressed(ebiten.KeyEnter)
	a.ctx.In.Consume(ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyArrowDown)
	a.picker.Keys(in, a.pickerHost)
}

func (a *App) closePicker() { a.picker = nil }

// maxNamedVars is how many missing variables a toast names before it counts
// the rest.
const maxNamedVars = 5

// missingVarsText says which variables a moved or copied node uses that
// collection does not define, naming at most maxNamedVars of them.
func missingVarsText(missing []string, name string) string {
	shown := missing
	if len(shown) > maxNamedVars {
		shown = shown[:maxNamedVars]
	}
	text := "Not defined in " + name + ": " + strings.Join(shown, ", ")
	if rest := len(missing) - len(shown); rest > 0 {
		text += fmt.Sprintf(" and %d more", rest)
	}
	return text
}

// transferNode moves or copies a node to the place t names, in this or
// another open collection, and tells what happened: a refusal, a rename and
// the variables the node uses that its new collection lacks. A move between
// collections writes the target first, so a failed write never loses the
// node from disk.
func (a *App) transferNode(t transfer.Transfer) {
	a.commitEdit()
	if entry, ok := a.open.ByCollection(t.To); ok {
		t.ToEnv = entry.Environment()
	}
	verb := "move"
	if t.Mode == transfer.Copy {
		verb = "copy"
	}
	out := transfer.Apply(t)
	if out.Err != nil {
		a.toast("Cannot "+verb+": "+out.Err.Error(), true)
		return
	}
	a.selectNode(out.Node)
	a.expanded[&t.To.Folder] = true
	a.expanded[t.Into] = true
	if t.Mode == transfer.Move && a.varsTarget.kind == varsFolder && !out.Node.None() && out.Node.Dir != nil && out.Node.Dir.Contains(model.Of(a.varsTarget.folder)) {
		a.varsTarget = varsTarget{}
	}
	if out.Renamed {
		a.toast("Renamed to "+out.Name+", the name was taken", false)
	}
	if len(out.Missing) > 0 {
		a.toast(missingVarsText(out.Missing, t.To.Name), false)
	}
	if t.Mode == transfer.Move && t.From != t.To {
		a.flushMoved(t.From, t.To)
	}
}

// flushMoved writes the target of a move between collections, then the
// source; when the target cannot be written the source keeps the node on
// disk and an error says so. A target that has no folder yet is not written,
// and then the source is not either, so the node is never on disk nowhere.
func (a *App) flushMoved(from, to *model.Collection) {
	target, okTarget := a.open.ByCollection(to)
	source, okSource := a.open.ByCollection(from)
	if !okTarget || !okSource || target.Dir == "" {
		return
	}
	if err := a.flushCollection(target); err != nil {
		a.toast("Saving "+to.Name+" failed, "+from.Name+" still has the node on disk: "+err.Error(), true)
		return
	}
	if err := a.flushCollection(source); err != nil {
		a.toast("Saving "+from.Name+" failed: "+err.Error(), true)
	}
}

// ---- keyboard ---------------------------------------------------------------

// textFocused reports whether a text widget has the keyboard.
func (a *App) textFocused() bool {
	t := a.cur()
	focused := a.filter.Focused || t.url.Focused || t.find.Focused || t.timeout.Focused || t.body.Focused ||
		t.tests.Focused || t.gqlVars.Focused || t.file.Focused || t.msg.Focused || a.edit != nil ||
		(a.bulk != nil && a.bulk.area.Focused)
	for _, in := range t.auth {
		focused = focused || in.Focused
	}
	return focused
}

// sidebarWidth is the current width of the sidebar in pixels.
func (a *App) sidebarWidth() int {
	if a.sidebarSplit.Size > 0 {
		return int(a.sidebarSplit.Size)
	}
	return sidebarW - 1
}

// treeKeys handles Up, Down, Enter, F2 and Delete while the tree has the
// keyboard and nothing is being typed.
func (a *App) treeKeys(in core.Input) {
	if a.textFocused() || in.Click && in.X > a.sidebarWidth() {
		a.treeFocus = false
	}
	if !a.treeFocus {
		return
	}
	rows := a.visibleRows()
	if len(rows) == 0 {
		return
	}
	current := -1
	for i, r := range rows {
		if r.Node() == a.selected {
			current = i
		}
	}
	for _, k := range in.Keys {
		switch k {
		case ebiten.KeyArrowDown:
			current = min(current+1, len(rows)-1)
			a.selectNode(rows[current].Node())
		case ebiten.KeyArrowUp:
			current = max(current-1, 0)
			a.selectNode(rows[current].Node())
		case ebiten.KeyEnter:
			if current >= 0 {
				a.clickTree(rows[current])
			}
		case ebiten.KeyF2:
			if !a.selected.None() {
				a.renameNode(a.selected)
			}
		case ebiten.KeyDelete:
			a.deleteNode(a.selected)
		}
	}
}

// imported is what a clipboard importer read: a folder or a single request,
// the collection variables it brings and the parts it left out.
type imported struct {
	folder  *model.Folder
	request *model.Request
	vars    []model.KV
	skipped []string
}

// importKind is one row of the importers: how a clipboard text is read and
// how a refusal and the parts left out are worded.
type importKind struct {
	refusal     string // starts the toast of a text that is not of this kind
	skippedWhat string // what the parts left out could not be ("parts could not be read")
	parse       func(text string) (imported, error)
}

var (
	curlKind = importKind{"Not a cURL command: ", "", func(text string) (imported, error) {
		req, err := model.ParseCurl(text)
		if err != nil {
			return imported{}, err
		}
		req.Name = curlName(req)
		return imported{request: &req}, nil
	}}
	openAPIKind = importKind{"Not an OpenAPI document: ", "parts could not be read", func(text string) (imported, error) {
		res, err := openapi.Parse([]byte(text))
		if err != nil {
			return imported{}, err
		}
		baseURL := model.KV{Name: openapi.BaseURLVar, Value: res.BaseURL, Enabled: true}
		return imported{folder: res.Folder, vars: []model.KV{baseURL}, skipped: res.Skipped}, nil
	}}
	postmanKind = importKind{"Not a Postman collection: ", "things could not be imported", func(text string) (imported, error) {
		res, err := postman.Parse([]byte(text))
		if err != nil {
			return imported{}, err
		}
		return imported{folder: res.Folder, vars: res.Vars, skipped: res.Skipped}, nil
	}}
)

// importFolder adds what the clipboard holds, read as kind, to dir and
// opens it; a text of another kind is explained in a toast.
func (a *App) importFolder(dir *model.Folder, kind importKind) {
	text, err := a.ctx.Clipboard.Read()
	if err != nil {
		a.toast("Cannot read the clipboard: "+err.Error(), true)
		return
	}
	got, err := kind.parse(text)
	if err != nil {
		a.toast(kind.refusal+err.Error(), true)
		return
	}
	name := a.attachImport(dir, got)
	a.status = "Imported " + name
	if len(got.skipped) > 0 {
		a.toast(fmt.Sprintf("%d %s, first: %s", len(got.skipped), kind.skippedWhat, got.skipped[0]), true)
	}
}

// attachImport puts got into dir under a name free there and returns it.
func (a *App) attachImport(dir *model.Folder, got imported) string {
	a.expanded[dir] = true
	if got.request != nil {
		got.request.Name = dir.UniqueName(got.request.Name)
		a.openRequest(dir.AddRequest(*got.request))
		return got.request.Name
	}
	got.folder.Name = dir.UniqueName(got.folder.Name)
	dir.Folders = append(dir.Folders, got.folder)
	if col := a.ownerOf(model.Of(dir)); col != nil {
		for _, v := range got.vars {
			if !model.Defined(v.Name, col.Vars) {
				col.Vars = append(col.Vars, v)
			}
		}
	}
	a.expandAll(got.folder)
	if first := tree.FirstRequest(got.folder); first != nil {
		a.openRequest(first)
	}
	return got.folder.Name
}

// curlName names an imported request after its method and path.
func curlName(request model.Request) string {
	path := request.URL
	if _, rest, ok := strings.Cut(path, "://"); ok {
		path = rest
	}
	path, _, _ = strings.Cut(path, "?")
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[i:]
	} else {
		path = "/"
	}
	return request.Method + " " + path
}

// dragExpandTicks is how long a drag must rest over a closed folder
// before the folder opens, so something can be dropped inside it.
const dragExpandTicks = 30

// dragHover opens the closed folder a drag rests over.
func (a *App) dragHover() {
	f := a.dropTarget.Dir
	if a.dragNode.None() || f == nil || a.expanded[f] || a.dragNode.Dir == f {
		a.hoverFolder, a.hoverTicks = nil, 0
		return
	}
	if a.hoverFolder != f {
		a.hoverFolder, a.hoverTicks = f, 0
	}
	if a.hoverTicks++; a.hoverTicks >= dragExpandTicks {
		a.expanded[f] = true
		a.hoverFolder, a.hoverTicks = nil, 0
	}
}

// exportPostman writes dir and everything below it as a Postman v2.1
// collection to a file the user picks. Requests of kinds Postman has no
// place for are left out and counted in the message.
func (a *App) exportPostman(dir *model.Folder) {
	a.commitEdit()
	var vars []model.KV
	if col := a.ownerOf(model.Of(dir)); col != nil {
		vars = col.Vars
	}
	data, err := postman.Export(dir, vars)
	if err != nil {
		a.toast("Export failed: "+err.Error(), true)
		return
	}
	skipped := postman.Skipped(dir)
	name := dir.Name
	if name == "" {
		name = "collection"
	}
	a.pickFile("Export as Postman collection", true, name+".postman_collection.json", func(path string) {
		a.writeFile(path, data, func(err error) {
			if err != nil {
				a.toast("Export failed: "+err.Error(), true)
				return
			}
			msg := "Exported to " + filepath.Base(path)
			if skipped > 0 {
				msg += fmt.Sprintf(" (%d requests of other kinds left out)", skipped)
			}
			a.toast(msg, false)
		})
	})
}
