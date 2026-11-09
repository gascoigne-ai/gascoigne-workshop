// Package tree decides what the collection tree's operations mean: where a
// dragged node lands, which open requests a delete closes and what the
// delete asks. The window draws the tree and asks the questions.
package tree

import (
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// DropPlace is the folder a node dropped on target goes into and the node it
// goes before (nil for the end). A request dropped on another request lands
// before it; a folder is always added at the end.
func DropPlace(root *model.Folder, node, target any) (dir *model.Folder, before any) {
	switch dropped := target.(type) {
	case *model.Folder:
		dir = dropped
	case *model.Request:
		dir, _ = root.Parent(dropped)
		before = dropped
	}
	if _, movingFolder := node.(*model.Folder); movingFolder {
		before = nil
	}
	return dir, before
}

// Inside reports whether req is node or lies inside the folder node.
func Inside(node any, req *model.Request) bool {
	if folder, ok := node.(*model.Folder); ok {
		return folder.Contains(req)
	}
	return any(req) == node
}

// NameOf is the name of a request or folder, "" for anything else.
func NameOf(node any) string {
	switch named := node.(type) {
	case *model.Request:
		return named.Name
	case *model.Folder:
		return named.Name
	}
	return ""
}

// DeletePrompt is the title and the warning of the question asked before
// node is deleted.
func DeletePrompt(node any) (title, body string) {
	body = "This cannot be undone."
	if _, isFolder := node.(*model.Folder); isFolder {
		body = "The folder and everything in it will be removed."
	}
	return "Delete " + NameOf(node) + "?", body
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
