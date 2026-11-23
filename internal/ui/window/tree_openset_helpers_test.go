package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Most tests open one collection; these reach it the way the App did when
// it held a single collection.

// col is the first open collection.
func (a *Component) col() *request.Collection { return a.cols.Set().Entries[0].Collection }

// workspace is the folder of the first open collection, "" when not saved.
func (a *Component) workspace() string { return a.cols.Set().Entries[0].Dir }

// setWorkspace makes dir the folder of the first open collection.
func (a *Component) setWorkspace(dir string) {
	a.cols.Apply(collections.Located{Collection: a.col(), Dir: dir})
}

// envIndex is the index of the focused collection's active environment.
func (a *Component) envIndex() int {
	entry, _ := a.focus()
	return entry.Env
}

// rowOf is the tree row that stands for node.
func rowOf(node request.Node) collection.Row {
	return collection.Row{Name: node.Name(), Folder: node.Dir != nil, Dir: node.Dir, Req: node.Req}
}

// holdDrag leaves the tree as a frame in the middle of a drag does: node
// pressed and the pointer over target, either of them none. The selection
// and the keyboard stay as they were, since a test sets up a drag without
// clicking.
func holdDrag(app *Component, node, target request.Node) {
	tree := app.side.Tree()
	selected, focused := tree.Selected(), tree.Focused()
	tree.Clear()
	tree.Release(false)
	if !node.None() {
		tree.Press(rowOf(node), false)
	}
	tree.Over(target)
	tree.Select(selected)
	if !focused {
		tree.Blur()
	}
}
