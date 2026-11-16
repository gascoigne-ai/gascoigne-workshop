package app

import (
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/openapi"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/postman"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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
func (a *App) selectNode(node any) { a.selected = node }

// selectedFolder is where a new request goes: the selected folder, the
// folder of the selected request, or the root of the focused collection;
// nil when nothing is open.
func (a *App) selectedFolder() *model.Folder {
	switch n := a.selected.(type) {
	case *model.Folder:
		return n
	case *model.Request:
		if owner, ok := a.open.Owner(n); ok {
			if parent, ok := owner.Collection.Folder.Parent(n); ok {
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
	rename := MenuItem{Label: "Rename", Run: func() { a.renameNode(tree.Of(node)) }}
	sep := MenuItem{Separator: true}
	if row.Req != nil {
		return []MenuItem{
			rename,
			{Label: "Duplicate", Run: func() { a.duplicateNode(tree.Of(node)) }},
			{Label: "Copy URL", Run: func() { a.copyRequestURL(row.Req) }},
			sep,
			{Label: "Delete", Run: func() { a.deleteNode(tree.Of(node)) }},
		}
	}
	create := []MenuItem{
		{Label: "New request", Run: func() { a.newRequestIn(row.Dir) }},
		{Label: "New folder", Run: func() { a.newFolderIn(row.Dir) }},
		{Label: "Import cURL from clipboard", Run: func() { a.importCurl(row.Dir) }},
		{Label: "Import OpenAPI from clipboard", Run: func() { a.importOpenAPI(row.Dir) }},
		{Label: "Import Postman from clipboard", Run: func() { a.importPostman(row.Dir) }},
		{Label: "Export as Postman...", Run: func() { a.exportPostman(row.Dir) }},
		{Label: "Variables", Run: func() { a.editVars(row.Dir) }},
		sep,
		rename,
	}
	if a.isRoot(row.Dir) {
		return append(create[:len(create)-1], a.rootMenuItems(row.Dir)...)
	}
	return append(create,
		MenuItem{Label: "Duplicate", Run: func() { a.duplicateNode(tree.Of(node)) }},
		MenuItem{Label: "Delete", Run: func() { a.deleteNode(tree.Of(node)) }},
	)
}

// ---- operations ----------------------------------------------------------

// rootMenuItems end the menu of a collection's root row: the collection
// itself is renamed, moved, closed and deleted as a collection, not as a
// folder. Moving is by menu rather than by drag, since a drag in the tree
// moves a node inside its collection.
func (a *App) rootMenuItems(root *model.Folder) []MenuItem {
	owner, _ := a.open.Owner(root)
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

// errOtherCollection refuses a drop into another collection: requests and
// folders are not moved across collections.
var errOtherCollection = errors.New("it would leave its collection")

// treeOwner is the collection a tree message changes: the one holding its
// node or folder; nil when no open collection holds it. A move onto a node
// of another collection is refused.
func (a *App) treeOwner(msg tree.Msg) (*model.Collection, error) {
	var subject, target any
	switch msg := msg.(type) {
	case tree.Rename:
		subject = msg.Node.Any()
	case tree.Duplicate:
		subject = msg.Node.Any()
	case tree.Delete:
		subject = msg.Node.Any()
	case tree.Move:
		subject, target = msg.Node.Any(), msg.Target.Any()
	case tree.NewRequest:
		subject = tree.Node{Dir: msg.Dir}.Any() // nil, not a nil *Folder, when there is no folder
	case tree.NewFolder:
		subject = tree.Node{Dir: msg.Dir}.Any()
	}
	col := a.ownerOf(subject)
	if col == nil {
		return nil, nil
	}
	if destination := a.ownerOf(target); destination != nil && destination != col {
		return nil, errOtherCollection
	}
	return col, nil
}

// ownerOf is the open collection that holds node, nil for none.
func (a *App) ownerOf(node any) *model.Collection {
	if node == nil {
		return nil
	}
	entry, _ := a.open.Owner(node)
	return entry.Collection
}

// applyTree runs a tree message against the collection that holds its
// node and takes over what it says: the selection, the folder to open,
// the request to show.
func (a *App) applyTree(msg tree.Msg) tree.Result {
	col, err := a.treeOwner(msg)
	if col == nil || err != nil {
		return tree.Result{Selected: tree.Of(a.selected), Err: err}
	}
	res := tree.Update(col, tree.Of(a.selected), msg)
	a.selected = res.Selected.Any()
	if res.Expand != nil {
		a.expanded[res.Expand] = true
	}
	if res.Open != nil {
		a.openRequest(res.Open)
	}
	if res.Removed != (tree.Node{}) {
		a.closeTabsIn(res.Removed)
	}
	return res
}

// newRequestIn adds a blank request to dir and opens it.
func (a *App) newRequestIn(dir *model.Folder) { a.applyTree(tree.NewRequest{Dir: dir}) }

func (a *App) newFolderIn(dir *model.Folder) { a.applyTree(tree.NewFolder{Dir: dir}) }

// renameNode asks for a new name and applies it; a refused name is
// explained in a toast.
func (a *App) renameNode(node tree.Node) {
	a.prompt("Rename", node.Name(), "Rename", func(text string) {
		if err := a.applyTree(tree.Rename{Node: node, Name: text}).Err; err != nil {
			a.toast("Cannot rename: "+err.Error(), true)
		}
	})
}

func (a *App) duplicateNode(node tree.Node) { a.applyTree(tree.Duplicate{Node: node}) }

// deleteNode asks, then removes the node and closes the tabs of
// everything that was in it.
func (a *App) deleteNode(node tree.Node) {
	if node == (tree.Node{}) || a.isRoot(node.Dir) {
		return
	}
	title, body := tree.DeletePrompt(node)
	a.confirm(title, body, "Delete", func() { a.applyTree(tree.Delete{Node: node}) })
}

// closeTabsIn closes the tabs whose request is node or inside it.
func (a *App) closeTabsIn(node tree.Node) {
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

// finishDrag drops the dragged node on the row it was released over.
func (a *App) finishDrag() {
	node, target, pressed := a.dragNode, a.dropTarget, a.pressedFolder
	a.dragNode, a.dropTarget, a.pressedFolder = nil, nil, nil
	if pressed != nil && target == any(pressed) { // a click on a folder row
		a.expanded[pressed] = !a.expanded[pressed]
		return
	}
	if err := a.applyTree(tree.Move{Node: tree.Of(node), Target: tree.Of(target)}).Err; err != nil {
		a.toast("Cannot move: "+err.Error(), true)
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
			if a.selected != nil {
				a.renameNode(tree.Of(a.selected))
			}
		case ebiten.KeyDelete:
			a.deleteNode(tree.Of(a.selected))
		}
	}
}

// importCurl adds the cURL command on the clipboard to dir as a new
// request and opens it.
func (a *App) importCurl(dir *model.Folder) {
	text, err := a.ctx.Clipboard.Read()
	if err != nil {
		a.toast("Cannot read the clipboard: "+err.Error(), true)
		return
	}
	req, err := model.ParseCurl(text)
	if err != nil {
		a.toast("Not a cURL command: "+err.Error(), true)
		return
	}
	req.Name = dir.UniqueName(curlName(req))
	added := dir.AddRequest(req)
	a.expanded[dir] = true
	a.openRequest(added)
	a.status = "Imported " + added.Name
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

// importOpenAPI adds the OpenAPI document (JSON) on the clipboard to dir as
// a folder of requests, and defines the baseUrl variable if the
// collection has none.
func (a *App) importOpenAPI(dir *model.Folder) {
	text, err := a.ctx.Clipboard.Read()
	if err != nil {
		a.toast("Cannot read the clipboard: "+err.Error(), true)
		return
	}
	res, err := openapi.Parse([]byte(text))
	if err != nil {
		a.toast("Not an OpenAPI document: "+err.Error(), true)
		return
	}
	res.Folder.Name = dir.UniqueName(res.Folder.Name)
	dir.Folders = append(dir.Folders, res.Folder)
	if col := a.ownerOf(dir); col != nil && !model.Defined(openapi.BaseURLVar, col.Vars) {
		col.Vars = append(col.Vars, model.KV{Name: openapi.BaseURLVar, Value: res.BaseURL, Enabled: true})
	}
	a.expanded[dir] = true
	a.expandAll(res.Folder)
	if first := tree.FirstRequest(res.Folder); first != nil {
		a.openRequest(first)
	}
	a.status = "Imported " + res.Folder.Name
	if len(res.Skipped) > 0 {
		a.toast(fmt.Sprintf("%d operations could not be read, first: %s", len(res.Skipped), res.Skipped[0]), true)
	}
}

// dragExpandTicks is how long a drag must rest over a closed folder
// before the folder opens, so something can be dropped inside it.
const dragExpandTicks = 30

// dragHover opens the closed folder a drag rests over.
func (a *App) dragHover() {
	f, ok := a.dropTarget.(*model.Folder)
	if a.dragNode == nil || !ok || a.expanded[f] || any(f) == a.dragNode {
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

// importPostman adds the Postman collection (JSON) on the clipboard to dir
// as a folder, and adds its variables that the collection does not have.
func (a *App) importPostman(dir *model.Folder) {
	text, err := a.ctx.Clipboard.Read()
	if err != nil {
		a.toast("Cannot read the clipboard: "+err.Error(), true)
		return
	}
	res, err := postman.Parse([]byte(text))
	if err != nil {
		a.toast("Not a Postman collection: "+err.Error(), true)
		return
	}
	res.Folder.Name = dir.UniqueName(res.Folder.Name)
	dir.Folders = append(dir.Folders, res.Folder)
	if col := a.ownerOf(dir); col != nil {
		for _, v := range res.Vars {
			if !model.Defined(v.Name, col.Vars) {
				col.Vars = append(col.Vars, v)
			}
		}
	}
	a.expanded[dir] = true
	a.expandAll(res.Folder)
	if first := tree.FirstRequest(res.Folder); first != nil {
		a.openRequest(first)
	}
	a.status = "Imported " + res.Folder.Name
	if len(res.Skipped) > 0 {
		a.toast(fmt.Sprintf("%d things could not be imported, first: %s", len(res.Skipped), res.Skipped[0]), true)
	}
}

// exportPostman writes dir and everything below it as a Postman v2.1
// collection to a file the user picks. Requests of kinds Postman has no
// place for are left out and counted in the message.
func (a *App) exportPostman(dir *model.Folder) {
	a.commitEdit()
	var vars []model.KV
	if col := a.ownerOf(dir); col != nil {
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
		if err := os.WriteFile(path, data, 0o644); err != nil {
			a.toast("Export failed: "+err.Error(), true)
			return
		}
		msg := "Exported to " + filepath.Base(path)
		if skipped > 0 {
			msg += fmt.Sprintf(" (%d requests of other kinds left out)", skipped)
		}
		a.toast(msg, false)
	})
}
