package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/hajimehoshi/ebiten/v2"
)

// ---- selection and the rows' menus --------------------------------------

// collectionToken is the color of the open collection that holds node, ""
// for a node of none.
func (a *App) collectionToken(node request.Node) core.Token {
	for index, entry := range a.cols.Set().Entries {
		if entry.Collection.Folder.Contains(node) {
			return theme.CollectionToken(index)
		}
	}
	return ""
}

// revealRequest selects req in the tree and opens the folders above it.
func (a *App) revealRequest(req *request.Request) {
	owner, ok := a.cols.Set().Owner(request.Node{Req: req})
	if !ok {
		a.side.Tree().Reveal(nil, req)
		return
	}
	a.side.Tree().Reveal(&owner.Collection.Folder, req)
}

// selectedFolder is where a new request goes: the selected folder, the
// folder of the selected request, or the root of the focused collection;
// nil when nothing is open.
func (a *App) selectedFolder() *request.Folder {
	selected := a.side.Tree().Selected()
	switch {
	case selected.Dir != nil:
		return selected.Dir
	case selected.Req != nil:
		if owner, ok := a.cols.Set().Owner(selected); ok {
			if parent, ok := owner.Collection.Folder.Parent(selected); ok {
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
	entry, _ := a.cols.Set().Owner(node)
	return entry.Collection
}

// applyTree runs a tree message against the collection that holds its
// node and takes over what it says: the selection, the folder to open,
// the request to show.
func (a *App) applyTree(msg tree.Msg) tree.Result {
	selected := a.side.Tree().Selected()
	col := a.treeOwner(msg)
	if col == nil {
		return tree.Result{Selected: selected}
	}
	res := tree.Update(col, selected, msg)
	a.side.Tree().Select(res.Selected)
	if res.Expand != nil {
		a.side.Tree().Expand(res.Expand)
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
	if node.None() || a.cols.Set().IsRoot(node.Dir) {
		return
	}
	title, body := tree.DeletePrompt(node)
	a.confirm(title, body, "Delete", func() { a.applyTree(tree.Delete{Node: node}) })
}

// closeTabsIn closes the tabs whose request is node or inside it.
func (a *App) closeTabsIn(node request.Node) {
	for i := len(a.tabs.Tabs()) - 1; i >= 0; i-- {
		if tree.Inside(node, a.tabs.Tabs()[i].Req) {
			a.closeTab(i)
		}
	}
}

func (a *App) copyRequestURL(req *request.Request) {
	_ = a.ctx.Clipboard.Write(request.Expand(req.URL, a.varsFor(req)))
}

// ---- drag and drop, and the destination picker ---------------------------------

// transferHost is the destination picker's view of the App: where a pick
// goes and what closes it.
type transferHost struct {
	*App
	node request.Node
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
func (a *App) openTransferPicker(node request.Node, mode transfer.Mode) {
	if node.None() || a.cols.Set().IsRoot(node.Dir) || a.ownerOf(node) == nil {
		return
	}
	stripe := func(col *request.Collection) core.Token { return a.collectionToken(request.Node{Dir: &col.Folder}) }
	a.picker = destpicker.New(node, mode, transfer.Destinations(a.cols.Set(), node, mode), stripe)
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

// transferNode moves or copies a node to the place t names, in this or
// another open collection, and tells what happened: a refusal, a rename and
// the variables the node uses that its new collection lacks. A move between
// collections writes the target first, so a failed write never loses the
// node from disk.
func (a *App) transferNode(t transfer.Transfer) {
	a.commitEdit()
	if entry, ok := a.cols.Set().ByCollection(t.To); ok {
		t.ToEnv = entry.Environment()
	}
	verb := "move"
	if t.Mode == transfer.Copy {
		verb = "copy"
	}
	out := transfer.Apply(t)
	if out.Err != nil {
		a.toast("Cannot "+verb+": "+out.Err.Error(), severity.Error)
		return
	}
	a.side.Tree().Select(out.Node)
	a.side.Tree().Expand(&t.To.Folder)
	a.side.Tree().Expand(t.Into)
	if t.Mode == transfer.Move && a.varsTarget.kind == varsFolder && !out.Node.None() && out.Node.Dir != nil && out.Node.Dir.Contains(request.Node{Dir: a.varsTarget.folder}) {
		a.varsTarget = varsTarget{}
	}
	if out.Renamed {
		a.toast("Renamed to "+out.Name+", the name was taken", severity.Info)
	}
	if len(out.Missing) > 0 {
		a.toast(transfer.MissingText(out.Missing, t.To.Name), severity.Info)
	}
	if t.Mode == transfer.Move && t.From != t.To {
		a.flushMoved(t.From, t.To)
	}
}

// flushMoved writes the target of a move between collections, then the
// source; when the target cannot be written the source keeps the node on
// disk and an error says so. A target that has no folder yet is not written,
// and then the source is not either, so the node is never on disk nowhere.
func (a *App) flushMoved(from, to *request.Collection) {
	target, okTarget := a.cols.Set().ByCollection(to)
	source, okSource := a.cols.Set().ByCollection(from)
	if !okTarget || !okSource || target.Dir == "" {
		return
	}
	if err := a.flushCollection(target); err != nil {
		a.toast("Saving "+to.Name+" failed, "+from.Name+" still has the node on disk: "+err.Error(), severity.Error)
		return
	}
	if err := a.flushCollection(source); err != nil {
		a.toast("Saving "+from.Name+" failed: "+err.Error(), severity.Error)
	}
}

// dragHover lets a drag resting over a closed folder open it, and marks the
// tree stale when it did, since the view was built before this step.
func (a *App) dragHover() {
	if a.side.Tick() {
		a.treeStale = true
	}
}
