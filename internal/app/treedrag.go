package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/ui/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/hajimehoshi/ebiten/v2"
)

// ---- drag and drop ---------------------------------------------------------

// finishDrag drops the dragged node on the row it was released over: it
// moves there, or is copied when Alt is held at the release.
func (a *App) finishDrag() {
	node, target, pressed := a.dragNode, a.dropTarget, a.pressedFolder
	a.dragNode, a.dropTarget, a.pressedFolder = request.Node{}, request.Node{}, nil
	if pressed != nil && target == (request.Node{Dir: pressed}) { // a click on a folder row
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
	if node.None() || a.isRoot(node.Dir) || a.ownerOf(node) == nil {
		return
	}
	stripe := func(col *request.Collection) core.Token { return a.collectionToken(request.Node{Dir: &col.Folder}) }
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
		a.toast("Cannot "+verb+": "+out.Err.Error(), severity.Error)
		return
	}
	a.selectNode(out.Node)
	a.expanded[&t.To.Folder] = true
	a.expanded[t.Into] = true
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
	target, okTarget := a.open.ByCollection(to)
	source, okSource := a.open.ByCollection(from)
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
		a.treeStale = true
		a.hoverFolder, a.hoverTicks = nil, 0
	}
}
