package request

import (
	"errors"
	"fmt"
	"strings"
)

// Errors of the tree operations.
var (
	ErrEmptyName  = errors.New("a name cannot be empty")
	ErrNameTaken  = errors.New("a sibling already has that name")
	ErrIntoItself = errors.New("a folder cannot move into itself")
	ErrRootMove   = errors.New("the root cannot be moved")
	ErrNotInTree  = errors.New("not in this tree")
)

func (f *Folder) siblingNames() map[string]bool {
	taken := map[string]bool{}
	for _, sub := range f.Folders {
		taken[sub.Name] = true
	}
	for _, r := range f.Requests {
		taken[r.Name] = true
	}
	return taken
}

// UniqueName returns base, or "base 2", "base 3", ... when a sibling
// (folder or request) of f already has the name.
func (f *Folder) UniqueName(base string) string {
	taken := f.siblingNames()
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		if name := fmt.Sprintf("%s %d", base, n); !taken[name] {
			return name
		}
	}
}

// AddRequest appends r to f under a name unique among its siblings and
// returns the stored request.
func (f *Folder) AddRequest(request Request) *Request {
	request.Name = f.UniqueName(request.Name)
	stored := &request
	f.Requests = append(f.Requests, stored)
	return stored
}

// AddFolder appends an empty folder named uniquely and returns it.
func (f *Folder) AddFolder(name string) *Folder {
	sub := &Folder{Name: f.UniqueName(name)}
	f.Folders = append(f.Folders, sub)
	return sub
}

// Parent returns the folder that directly holds node somewhere below f. The
// root and the zero Node have no parent.
func (f *Folder) Parent(node Node) (*Folder, bool) {
	if node.None() {
		return nil, false
	}
	for _, r := range f.Requests {
		if node.Req == r {
			return f, true
		}
	}
	for _, sub := range f.Folders {
		if node.Dir == sub {
			return f, true
		}
		if p, ok := sub.Parent(node); ok {
			return p, true
		}
	}
	return nil, false
}

// Contains reports whether node is f itself or anywhere below it.
func (f *Folder) Contains(node Node) bool {
	if node.None() {
		return false
	}
	if node.Dir == f {
		return true
	}
	_, ok := f.Parent(node)
	return ok
}

func nameOf(node Node) *string {
	switch {
	case node.Req != nil:
		return &node.Req.Name
	case node.Dir != nil:
		return &node.Dir.Name
	}
	return nil
}

// Rename gives node a new name, unique among its siblings.
func (f *Folder) Rename(node Node, name string) error {
	field := nameOf(node)
	if field == nil || !f.Contains(node) {
		return ErrNotInTree
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptyName
	}
	if parent, ok := f.Parent(node); ok && name != *field && parent.siblingNames()[name] {
		return ErrNameTaken
	}
	*field = name
	return nil
}

// Duplicate copies node (a folder with everything in it) right after
// the original, named "<name> copy", and returns the copy. The root
// cannot be duplicated; the zero Node is returned for it, for the zero Node
// and for a node of no tree.
func (f *Folder) Duplicate(node Node) Node {
	parent, ok := f.Parent(node)
	if !ok {
		return Node{}
	}
	switch {
	case node.Req != nil:
		cp := node.Req.Clone()
		cp.Name = parent.UniqueName(node.Req.Name + " copy")
		stored := &cp
		parent.Requests = insertAt(parent.Requests, indexOf(parent.Requests, node.Req)+1, stored)
		return Node{Req: stored}
	default:
		cp := node.Dir.deepCopy()
		cp.Name = parent.UniqueName(node.Dir.Name + " copy")
		parent.Folders = insertAt(parent.Folders, indexOf(parent.Folders, node.Dir)+1, cp)
		return Node{Dir: cp}
	}
}

func (f *Folder) deepCopy() *Folder {
	cp := &Folder{Name: f.Name, Vars: cloneKV(f.Vars)}
	for _, sub := range f.Folders {
		cp.Folders = append(cp.Folders, sub.deepCopy())
	}
	for _, r := range f.Requests {
		rc := r.Clone()
		cp.Requests = append(cp.Requests, &rc)
	}
	return cp
}

// Delete removes node and everything below it; the root cannot be deleted.
func (f *Folder) Delete(node Node) bool {
	parent, ok := f.Parent(node)
	if !ok {
		return false
	}
	switch {
	case node.Req != nil:
		parent.Requests = removeAt(parent.Requests, indexOf(parent.Requests, node.Req))
	case node.Dir != nil:
		parent.Folders = removeAt(parent.Folders, indexOf(parent.Folders, node.Dir))
	}
	return true
}

func indexOf[T comparable](items []T, item T) int {
	for i, it := range items {
		if it == item {
			return i
		}
	}
	return -1
}

func insertAt[T any](items []T, index int, item T) []T {
	items = append(items, item)
	copy(items[index+1:], items[index:])
	items[index] = item
	return items
}

func removeAt[T any](items []T, index int) []T {
	if index < 0 {
		return items
	}
	return append(items[:index], items[index+1:]...)
}

// Clone returns a deep copy of c.
func (c Collection) Clone() Collection {
	cp := c.Folder.deepCopy()
	out := Collection{Folder: *cp, Vars: cloneKV(c.Vars)}
	for _, e := range c.Envs {
		out.Envs = append(out.Envs, Environment{Name: e.Name, Vars: cloneKV(e.Vars)})
	}
	return out
}
