package model

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
func (f *Folder) AddRequest(r Request) *Request {
	r.Name = f.UniqueName(r.Name)
	stored := &r
	f.Requests = append(f.Requests, stored)
	return stored
}

// AddFolder appends an empty folder named uniquely and returns it.
func (f *Folder) AddFolder(name string) *Folder {
	sub := &Folder{Name: f.UniqueName(name)}
	f.Folders = append(f.Folders, sub)
	return sub
}

// Parent returns the folder that directly holds node (a *Request or
// *Folder) somewhere below f. The root has no parent.
func (f *Folder) Parent(node any) (*Folder, bool) {
	for _, r := range f.Requests {
		if any(r) == node {
			return f, true
		}
	}
	for _, sub := range f.Folders {
		if any(sub) == node {
			return f, true
		}
		if p, ok := sub.Parent(node); ok {
			return p, true
		}
	}
	return nil, false
}

// Contains reports whether node is f itself or anywhere below it.
func (f *Folder) Contains(node any) bool {
	if any(f) == node {
		return true
	}
	_, ok := f.Parent(node)
	return ok
}

func nameOf(node any) *string {
	switch n := node.(type) {
	case *Request:
		return &n.Name
	case *Folder:
		return &n.Name
	}
	return nil
}

// Rename gives node a new name, unique among its siblings.
func (f *Folder) Rename(node any, name string) error {
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
// cannot be duplicated.
func (f *Folder) Duplicate(node any) any {
	parent, ok := f.Parent(node)
	if !ok {
		return nil
	}
	switch n := node.(type) {
	case *Request:
		cp := n.Clone()
		cp.Name = parent.UniqueName(n.Name + " copy")
		stored := &cp
		parent.Requests = insertAt(parent.Requests, indexOf(parent.Requests, n)+1, stored)
		return stored
	case *Folder:
		cp := n.deepCopy()
		cp.Name = parent.UniqueName(n.Name + " copy")
		parent.Folders = insertAt(parent.Folders, indexOf(parent.Folders, n)+1, cp)
		return cp
	}
	return nil
}

func (f *Folder) deepCopy() *Folder {
	cp := &Folder{Name: f.Name}
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
func (f *Folder) Delete(node any) bool {
	parent, ok := f.Parent(node)
	if !ok {
		return false
	}
	switch n := node.(type) {
	case *Request:
		parent.Requests = removeAt(parent.Requests, indexOf(parent.Requests, n))
	case *Folder:
		parent.Folders = removeAt(parent.Folders, indexOf(parent.Folders, n))
	}
	return true
}

// Move takes node out of its folder and puts it into target, before the
// sibling `before` (a request for a request, a folder for a folder), or
// at the end when before is nil. Nothing changes when it fails.
func (f *Folder) Move(node any, target *Folder, before any) error {
	if any(f) == node {
		return ErrRootMove
	}
	src, ok := f.Parent(node)
	if !ok || !f.Contains(target) {
		return ErrNotInTree
	}
	if sub, isFolder := node.(*Folder); isFolder && sub.Contains(target) {
		return ErrIntoItself
	}
	if src != target && target.siblingNames()[*nameOf(node)] {
		return ErrNameTaken
	}
	switch n := node.(type) {
	case *Request:
		src.Requests = removeAt(src.Requests, indexOf(src.Requests, n))
		at := len(target.Requests)
		if b, ok := before.(*Request); ok && indexOf(target.Requests, b) >= 0 {
			at = indexOf(target.Requests, b)
		}
		target.Requests = insertAt(target.Requests, at, n)
	case *Folder:
		src.Folders = removeAt(src.Folders, indexOf(src.Folders, n))
		at := len(target.Folders)
		if b, ok := before.(*Folder); ok && indexOf(target.Folders, b) >= 0 {
			at = indexOf(target.Folders, b)
		}
		target.Folders = insertAt(target.Folders, at, n)
	}
	return nil
}

func indexOf[T comparable](items []T, item T) int {
	for i, it := range items {
		if it == item {
			return i
		}
	}
	return -1
}

func insertAt[T any](items []T, i int, item T) []T {
	items = append(items, item)
	copy(items[i+1:], items[i:])
	items[i] = item
	return items
}

func removeAt[T any](items []T, i int) []T {
	if i < 0 {
		return items
	}
	return append(items[:i], items[i+1:]...)
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
