// Package tree decides what the collection tree's operations mean: where a
// dragged node lands, which open requests a delete closes and what the
// delete asks. The window draws the tree and asks the questions.
package tree

import (
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// DropPlace is the folder a node dropped on target goes into and the node it
// goes before (zero for the end). A request dropped on another request lands
// before it; a folder is always added at the end.
func DropPlace(root *model.Folder, node, target Node) (dir *model.Folder, before Node) {
	switch {
	case target.Dir != nil:
		dir = target.Dir
	case target.Req != nil:
		dir, _ = root.Parent(target.Req)
		before = target
	}
	if node.Dir != nil {
		before = Node{}
	}
	return dir, before
}

// Inside reports whether req is node or lies inside the folder node.
func Inside(node Node, req *model.Request) bool {
	if node.Dir != nil {
		return node.Dir.Contains(req)
	}
	return node.Req == req
}

// DeletePrompt is the title and the warning of the question asked before
// node is deleted.
func DeletePrompt(node Node) (title, body string) {
	body = "This cannot be undone."
	if node.Dir != nil {
		body = "The folder and everything in it will be removed."
	}
	return "Delete " + node.Name() + "?", body
}

// FirstRequest is the first request in tree order, or nil.
func FirstRequest(folder *model.Folder) *model.Request {
	if len(folder.Requests) > 0 {
		return folder.Requests[0]
	}
	for _, sub := range folder.Folders {
		if first := FirstRequest(sub); first != nil {
			return first
		}
	}
	return nil
}

// Key is a comparable form of the collection: it differs when the
// collection does, which is how a change is noticed.
func Key(collection model.Collection) string {
	data, err := json.Marshal(collection)
	if err != nil {
		return err.Error()
	}
	return string(data)
}
