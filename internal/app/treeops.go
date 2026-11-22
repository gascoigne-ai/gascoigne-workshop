package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/hajimehoshi/ebiten/v2"
)

// ---- what the tree shows -------------------------------------------------

// visibleRows are the tree rows shown, after the filter.
func (a *App) visibleRows() []collection.Row {
	var rows []collection.Row
	a.eachRoot(func(root *request.Folder) {
		rows = append(rows, collection.FilteredRows(root, a.expanded, a.filter.Text)...)
	})
	return rows
}

// ---- selection and the rows' menus --------------------------------------

// selectNode marks a folder or request as the one the tree acts on.
func (a *App) selectNode(node request.Node) { a.selected = node }

// collectionToken is the color of the open collection that holds node, ""
// for a node of none.
func (a *App) collectionToken(node request.Node) core.Token {
	for index, entry := range a.open.Entries {
		if entry.Collection.Folder.Contains(node) {
			return theme.CollectionToken(index)
		}
	}
	return ""
}

// revealRequest selects req in the tree and opens the folders above it.
func (a *App) revealRequest(req *request.Request) {
	a.selectNode(request.Node{Req: req})
	owner, ok := a.open.Owner(request.Node{Req: req})
	if !ok {
		return
	}
	root := &owner.Collection.Folder
	a.expanded[root] = true
	for folder, found := root.Parent(request.Node{Req: req}); found && folder != root; folder, found = root.Parent(request.Node{Dir: folder}) {
		a.expanded[folder] = true
	}
}

// selectedFolder is where a new request goes: the selected folder, the
// folder of the selected request, or the root of the focused collection;
// nil when nothing is open.
func (a *App) selectedFolder() *request.Folder {
	switch {
	case a.selected.Dir != nil:
		return a.selected.Dir
	case a.selected.Req != nil:
		if owner, ok := a.open.Owner(a.selected); ok {
			if parent, ok := owner.Collection.Folder.Parent(a.selected); ok {
				return parent
			}
		}
	}
	if col := a.focusCol(); col != nil {
		return &col.Folder
	}
	return nil
}

// treeOwner is the collection a tree message changes: the one holding its
// node or folder; nil when no open collection holds it.
func (a *App) treeOwner(msg tree.Msg) *request.Collection {
	var subject request.Node
	switch msg := msg.(type) {
	case tree.Rename:
		subject = msg.Node
	case tree.Duplicate:
		subject = msg.Node
	case tree.Delete:
		subject = msg.Node
	case tree.NewRequest:
		subject = request.Node{Dir: msg.Dir} // none, not a nil *Folder, when there is no folder
	case tree.NewFolder:
		subject = request.Node{Dir: msg.Dir}
	}
	return a.ownerOf(subject)
}

// ownerOf is the open collection that holds node, nil for none.
func (a *App) ownerOf(node request.Node) *request.Collection {
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
func (a *App) newRequestIn(dir *request.Folder) { a.applyTree(tree.NewRequest{Dir: dir}) }

func (a *App) newFolderIn(dir *request.Folder) { a.applyTree(tree.NewFolder{Dir: dir}) }

// renameNode asks for a new name and applies it; a refused name is
// explained in a toast.
func (a *App) renameNode(node request.Node) {
	a.prompt("Rename", node.Name(), "Rename", func(text string) {
		if err := a.applyTree(tree.Rename{Node: node, Name: text}).Err; err != nil {
			a.toast("Cannot rename: "+err.Error(), severity.Error)
		}
	})
}

func (a *App) duplicateNode(node request.Node) { a.applyTree(tree.Duplicate{Node: node}) }

// deleteNode asks, then removes the node and closes the tabs of
// everything that was in it.
func (a *App) deleteNode(node request.Node) {
	if node.None() || a.isRoot(node.Dir) {
		return
	}
	title, body := tree.DeletePrompt(node)
	a.confirm(title, body, "Delete", func() { a.applyTree(tree.Delete{Node: node}) })
}

// closeTabsIn closes the tabs whose request is node or inside it.
func (a *App) closeTabsIn(node request.Node) {
	for i := len(a.list.Tabs) - 1; i >= 0; i-- {
		if tree.Inside(node, a.list.Tabs[i].Req) {
			a.closeTab(i)
		}
	}
}

func (a *App) copyRequestURL(req *request.Request) {
	_ = a.ctx.Clipboard.Write(request.Expand(req.URL, a.varsFor(req)))
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
