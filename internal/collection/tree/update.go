package tree

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request"

// Msg is something done to the tree. The window sends it from a menu, a
// key or a drop; Update applies it.
type Msg interface{ treeMsg() }

// Rename gives Node a new name.
type Rename struct {
	Node request.Node
	Name string
}

// Duplicate copies Node next to itself and selects the copy.
type Duplicate struct{ Node request.Node }

// Delete removes Node and what is inside it.
type Delete struct{ Node request.Node }

// NewRequest adds a blank request to Dir and opens it.
type NewRequest struct{ Dir *request.Folder }

// NewFolder adds an empty folder to Dir and selects it.
type NewFolder struct{ Dir *request.Folder }

func (Rename) treeMsg()     {}
func (Duplicate) treeMsg()  {}
func (Delete) treeMsg()     {}
func (NewRequest) treeMsg() {}
func (NewFolder) treeMsg()  {}

// Result is what a message did, for the window to show: the selection
// afterwards, a folder to expand, a request to open, a node whose open
// tabs to close, or the reason it was refused.
type Result struct {
	Selected request.Node
	Expand   *request.Folder
	Open     *request.Request
	Removed  request.Node
	Err      error
}

// Update applies msg to the collection in place (open tabs hold pointers
// into it) and says what happened. selected is the current selection.
func Update(collection *request.Collection, selected request.Node, msg Msg) Result {
	root := &collection.Folder
	switch msg := msg.(type) {
	case Rename:
		return Result{Selected: selected, Err: root.Rename(msg.Node, msg.Name)}
	case Duplicate:
		if cp := root.Duplicate(msg.Node); !cp.None() {
			return Result{Selected: cp}
		}
	case Delete:
		return deleteNode(root, selected, msg.Node)
	case NewRequest:
		req := msg.Dir.AddRequest(request.Request{Name: "New Request", Protocol: request.ProtocolREST, Method: "GET"})
		return Result{Selected: selected, Expand: msg.Dir, Open: req}
	case NewFolder:
		sub := msg.Dir.AddFolder("New Folder")
		return Result{Selected: request.Node{Dir: sub}, Expand: msg.Dir}
	}
	return Result{Selected: selected}
}

func deleteNode(root *request.Folder, selected, node request.Node) Result {
	if node.None() || node.Dir == root || !root.Delete(node) {
		return Result{Selected: selected}
	}
	if selected == node {
		selected = request.Node{}
	}
	return Result{Selected: selected, Removed: node}
}
