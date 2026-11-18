package tree

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// Msg is something done to the tree. The window sends it from a menu, a
// key or a drop; Update applies it.
type Msg interface{ treeMsg() }

// Rename gives Node a new name.
type Rename struct {
	Node model.Node
	Name string
}

// Duplicate copies Node next to itself and selects the copy.
type Duplicate struct{ Node model.Node }

// Delete removes Node and what is inside it.
type Delete struct{ Node model.Node }

// Move drops Node on Target (a folder, or a request it lands before).
type Move struct{ Node, Target model.Node }

// NewRequest adds a blank request to Dir and opens it.
type NewRequest struct{ Dir *model.Folder }

// NewFolder adds an empty folder to Dir and selects it.
type NewFolder struct{ Dir *model.Folder }

func (Rename) treeMsg()     {}
func (Duplicate) treeMsg()  {}
func (Delete) treeMsg()     {}
func (Move) treeMsg()       {}
func (NewRequest) treeMsg() {}
func (NewFolder) treeMsg()  {}

// Result is what a message did, for the window to show: the selection
// afterwards, a folder to expand, a request to open, a node whose open
// tabs to close, or the reason it was refused.
type Result struct {
	Selected model.Node
	Expand   *model.Folder
	Open     *model.Request
	Removed  model.Node
	Err      error
}

// Update applies msg to the collection in place (open tabs hold pointers
// into it) and says what happened. selected is the current selection.
func Update(collection *model.Collection, selected model.Node, msg Msg) Result {
	root := &collection.Folder
	switch msg := msg.(type) {
	case Rename:
		return Result{Selected: selected, Err: root.Rename(msg.Node.Any(), msg.Name)}
	case Duplicate:
		if cp := model.Of(root.Duplicate(msg.Node.Any())); !cp.None() {
			return Result{Selected: cp}
		}
	case Delete:
		return deleteNode(root, selected, msg.Node)
	case Move:
		return move(root, selected, msg)
	case NewRequest:
		req := msg.Dir.AddRequest(model.Request{Name: "New Request", Protocol: model.ProtocolREST, Method: "GET"})
		return Result{Selected: selected, Expand: msg.Dir, Open: req}
	case NewFolder:
		sub := msg.Dir.AddFolder("New Folder")
		return Result{Selected: model.Node{Dir: sub}, Expand: msg.Dir}
	}
	return Result{Selected: selected}
}

func deleteNode(root *model.Folder, selected, node model.Node) Result {
	if node.None() || node.Dir == root || !root.Delete(node.Any()) {
		return Result{Selected: selected}
	}
	if selected == node {
		selected = model.Node{}
	}
	return Result{Selected: selected, Removed: node}
}

func move(root *model.Folder, selected model.Node, msg Move) Result {
	if msg.Node.None() || msg.Target.None() || msg.Target == msg.Node {
		return Result{Selected: selected}
	}
	dir, before := DropPlace(root, msg.Node, msg.Target)
	if err := root.Move(msg.Node.Any(), dir, before.Any()); err != nil {
		return Result{Selected: selected, Err: err}
	}
	return Result{Selected: selected, Expand: dir}
}
