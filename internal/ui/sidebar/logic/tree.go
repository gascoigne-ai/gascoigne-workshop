// Package logic is what the sidebar decides without drawing: which node is
// selected, which folders are open, who has the keyboard and what a drag
// means. It imports no toolkit, so it is tested without a window.
package logic

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Tree is the state of the collection tree: the selected node, the open
// folders, whether the tree has the keyboard and the drag in progress.
//
// A folder is identified by its pointer, since two folders may share a
// name and open tabs point into the same folders. The open set is a map
// because every visible row asks "is it open" on every frame; a slice of
// open folders would make that a scan per row. Rejected: a field per row,
// which would have to follow every rename, move and reload.
type Tree struct {
	selected request.Node
	expanded map[*request.Folder]bool
	focused  bool
	drag     Drag
}

// New returns a tree with nothing selected and every folder closed.
func New() *Tree { return &Tree{expanded: map[*request.Folder]bool{}} }

// Selected is the node last clicked or moved to, none when nothing is.
func (tree *Tree) Selected() request.Node { return tree.selected }

// Select marks node as the one the tree acts on.
func (tree *Tree) Select(node request.Node) { tree.selected = node }

// Expanded is the set of open folders. The map is the tree's own: a row
// list is built from it every frame (collection.FilteredRows).
func (tree *Tree) Expanded() map[*request.Folder]bool { return tree.expanded }

// Expand opens folder.
func (tree *Tree) Expand(folder *request.Folder) { tree.expanded[folder] = true }

// Toggle opens a closed folder and closes an open one.
func (tree *Tree) Toggle(folder *request.Folder) { tree.expanded[folder] = !tree.expanded[folder] }

// ExpandAll opens folder and everything below it.
func (tree *Tree) ExpandAll(folder *request.Folder) {
	tree.expanded[folder] = true
	for _, sub := range folder.Folders {
		tree.ExpandAll(sub)
	}
}

// Reset forgets the selection and every open folder, as when the whole open
// set is replaced.
func (tree *Tree) Reset() {
	tree.selected = request.Node{}
	tree.expanded = map[*request.Folder]bool{}
}

// Reveal selects req and opens root and every folder between it and req.
// A nil root only selects.
func (tree *Tree) Reveal(root *request.Folder, req *request.Request) {
	tree.selected = request.Node{Req: req}
	if root == nil {
		return
	}
	tree.expanded[root] = true
	for folder, found := root.Parent(request.Node{Req: req}); found && folder != root; folder, found = root.Parent(request.Node{Dir: folder}) {
		tree.expanded[folder] = true
	}
}

// Focus gives the tree the keyboard.
func (tree *Tree) Focus() { tree.focused = true }

// Blur takes the keyboard from the tree.
func (tree *Tree) Blur() { tree.focused = false }

// Focused reports whether the tree has the keyboard.
func (tree *Tree) Focused() bool { return tree.focused }

// Step moves the selection by one row (by is -1 or 1), stopping at the first
// and the last. With nothing selected the first row is chosen; with no
// rows nothing changes.
func (tree *Tree) Step(rows []collection.Row, by int) {
	if len(rows) == 0 {
		return
	}
	current := -1
	for index, row := range rows {
		if row.Node() == tree.selected {
			current = index
		}
	}
	current = min(max(current+by, 0), len(rows)-1)
	tree.selected = rows[current].Node()
}
