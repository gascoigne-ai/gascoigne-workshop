package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/openapi"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/postman"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// ---- what the tree shows -------------------------------------------------

// visibleRows are the tree rows shown, after the filter.
func (a *App) visibleRows() []collection.Row {
	return collection.FilteredRows(&a.col.Folder, a.expanded, a.filter.Text)
}

// ---- selection and the rows' menus --------------------------------------

// selectNode marks a folder or request as the one the tree acts on.
func (a *App) selectNode(node any) { a.selected = node }

// selectedFolder is where a new request goes: the selected folder, the
// folder of the selected request, or the root.
func (a *App) selectedFolder() *model.Folder {
	switch n := a.selected.(type) {
	case *model.Folder:
		return n
	case *model.Request:
		if parent, ok := a.col.Folder.Parent(n); ok {
			return parent
		}
	}
	return &a.col.Folder
}

// treeMenuItems are the right-click actions for a row.
func (a *App) treeMenuItems(row collection.Row) []menuItem {
	node := row.Node()
	rename := menuItem{label: "Rename", run: func() { a.renameNode(node) }}
	sep := menuItem{separator: true}
	if row.Req != nil {
		return []menuItem{
			rename,
			{label: "Duplicate", run: func() { a.duplicateNode(node) }},
			{label: "Copy URL", run: func() { a.copyRequestURL(row.Req) }},
			sep,
			{label: "Delete", run: func() { a.deleteNode(node) }},
		}
	}
	create := []menuItem{
		{label: "New request", run: func() { a.newRequestIn(row.Dir) }},
		{label: "New folder", run: func() { a.newFolderIn(row.Dir) }},
		{label: "Import cURL from clipboard", run: func() { a.importCurl(row.Dir) }},
		{label: "Import OpenAPI from clipboard", run: func() { a.importOpenAPI(row.Dir) }},
		{label: "Import Postman from clipboard", run: func() { a.importPostman(row.Dir) }},
		{label: "Export as Postman...", run: func() { a.exportPostman(row.Dir) }},
		{label: "Variables", run: func() { a.editVars(row.Dir) }},
		sep,
		rename,
	}
	if row.Dir == &a.col.Folder {
		return create // the collection itself is not duplicated or deleted
	}
	return append(create,
		menuItem{label: "Duplicate", run: func() { a.duplicateNode(node) }},
		menuItem{label: "Delete", run: func() { a.deleteNode(node) }},
	)
}

// ---- operations ----------------------------------------------------------

// newRequestIn adds a blank request to dir and opens it.
func (a *App) newRequestIn(dir *model.Folder) {
	req := dir.AddRequest(model.Request{Name: "New Request", Protocol: "rest", Method: "GET"})
	a.expanded[dir] = true
	a.openRequest(req)
}

func (a *App) newFolderIn(dir *model.Folder) {
	sub := dir.AddFolder("New Folder")
	a.expanded[dir] = true
	a.selectNode(sub)
}

// renameNode asks for a new name and applies it; a refused name is
// explained in a toast.
func (a *App) renameNode(node any) {
	field := *nodeName(node)
	a.prompt("Rename", field, "Rename", func(text string) {
		if err := a.col.Folder.Rename(node, text); err != nil {
			a.toast("Cannot rename: "+err.Error(), true)
		}
	})
}

func nodeName(node any) *string {
	switch n := node.(type) {
	case *model.Request:
		return &n.Name
	case *model.Folder:
		return &n.Name
	}
	empty := ""
	return &empty
}

func (a *App) duplicateNode(node any) {
	if cp := a.col.Folder.Duplicate(node); cp != nil {
		a.selectNode(cp)
	}
}

// deleteNode asks, then removes the node and closes the tabs of
// everything that was in it.
func (a *App) deleteNode(node any) {
	if node == nil || node == any(&a.col.Folder) {
		return
	}
	body := "This cannot be undone."
	if _, isFolder := node.(*model.Folder); isFolder {
		body = "The folder and everything in it will be removed."
	}
	a.confirm("Delete "+*nodeName(node)+"?", body, "Delete", func() {
		a.closeTabsIn(node)
		a.col.Folder.Delete(node)
		if a.selected == node {
			a.selected = nil
		}
	})
}

// closeTabsIn closes the tabs whose request is node or inside it.
func (a *App) closeTabsIn(node any) {
	for i := len(a.tabs) - 1; i >= 0; i-- {
		req := a.tabs[i].req
		inside := any(req) == node
		if f, ok := node.(*model.Folder); ok {
			inside = f.Contains(req)
		}
		if inside {
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
	if node == nil || target == nil || target == node {
		return
	}
	var dir *model.Folder
	var before any
	switch t := target.(type) {
	case *model.Folder:
		dir = t
	case *model.Request:
		dir, _ = a.col.Folder.Parent(t)
		before = t // a request lands before the one it was dropped on
	}
	if _, movingFolder := node.(*model.Folder); movingFolder {
		before = nil
	}
	if err := a.col.Folder.Move(node, dir, before); err != nil {
		a.toast("Cannot move: "+err.Error(), true)
		return
	}
	a.expanded[dir] = true
}

// ---- keyboard ---------------------------------------------------------------

// textFocused reports whether a text widget has the keyboard.
func (a *App) textFocused() bool {
	t := a.cur()
	focused := a.filter.Focused || t.url.Focused || t.find.Focused || t.timeout.Focused || t.body.Focused || a.edit != nil ||
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
				a.renameNode(a.selected)
			}
		case ebiten.KeyDelete:
			a.deleteNode(a.selected)
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
	req, err := proto.ParseCurl(text)
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
func curlName(r model.Request) string {
	path := r.URL
	if _, rest, ok := strings.Cut(path, "://"); ok {
		path = rest
	}
	path, _, _ = strings.Cut(path, "?")
	if i := strings.Index(path, "/"); i >= 0 {
		path = path[i:]
	} else {
		path = "/"
	}
	return r.Method + " " + path
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
	if !model.Defined(openapi.BaseURLVar, a.col.Vars) {
		a.col.Vars = append(a.col.Vars, model.KV{Name: openapi.BaseURLVar, Value: res.BaseURL, Enabled: true})
	}
	a.expanded[dir] = true
	a.expandAll(res.Folder)
	if first := firstRequest(res.Folder); first != nil {
		a.openRequest(first)
	}
	a.status = "Imported " + res.Folder.Name
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
	for _, v := range res.Vars {
		if !model.Defined(v.Name, a.col.Vars) {
			a.col.Vars = append(a.col.Vars, v)
		}
	}
	a.expanded[dir] = true
	a.expandAll(res.Folder)
	if first := firstRequest(res.Folder); first != nil {
		a.openRequest(first)
	}
	a.status = "Imported " + res.Folder.Name
}

// exportPostman writes dir and everything below it as a Postman v2.1
// collection to a file the user picks. Requests of kinds Postman has no
// place for are left out and counted in the message.
func (a *App) exportPostman(dir *model.Folder) {
	a.commitEdit()
	data, err := postman.Export(dir, a.col.Vars)
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
