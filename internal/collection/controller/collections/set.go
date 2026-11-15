package collections

import (
	"path/filepath"
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Entry is one open collection: its tree, the folder it is saved in and
// its active environment. The Collection pointer is its identity, since
// open tabs and the tree point into it.
type Entry struct {
	Collection *model.Collection
	// Dir is the folder the collection is saved in, "" until its first
	// save. Two entries never share a folder.
	Dir string
	// Env indexes the active environment in Collection.Envs, -1 for none.
	// It is an index rather than an envs.Selection because a controller
	// reaches another domain only through its model.
	Env int
}

// Environment is the entry's active environment, nil for none.
func (entry Entry) Environment() *model.Environment {
	if entry.Env < 0 || entry.Env >= len(entry.Collection.Envs) {
		return nil
	}
	return &entry.Collection.Envs[entry.Env]
}

// VarsFor are the variables req is sent with: the entry's own collection,
// environment and folders, nothing of another collection.
func (entry Entry) VarsFor(req *model.Request) []model.KV {
	return entry.Collection.VarsFor(req, entry.Environment())
}

// Set is the open collections, in the order the tree shows them. Folders
// are compared after filepath.Clean only, so callers pass absolute paths
// with symbolic links resolved.
type Set struct {
	Entries []Entry
}

// Msg is a change to a Set.
type Msg interface{ setMsg() }

// Open adds Collection, saved in Dir ("" when not yet saved), after the
// open ones with no active environment. A collection already open, or a
// folder another open collection is saved in, changes nothing: two copies
// of one folder would overwrite each other's files.
type Open struct {
	Collection *model.Collection
	Dir        string
}

// Close takes Collection out of the set. Closing the last one leaves none;
// what opens then is the caller's choice.
type Close struct{ Collection *model.Collection }

// Move puts Collection before Before, at the end when Before is nil.
type Move struct{ Collection, Before *model.Collection }

// Located records that Collection is now saved in Dir, after its first
// save or a rename. A folder another open collection holds is refused.
type Located struct {
	Collection *model.Collection
	Dir        string
}

// ChooseEnv makes environment Index of Collection the active one, none
// when Index is out of range.
type ChooseEnv struct {
	Collection *model.Collection
	Index      int
}

func (Open) setMsg()      {}
func (Close) setMsg()     {}
func (Move) setMsg()      {}
func (Located) setMsg()   {}
func (ChooseEnv) setMsg() {}

// Update returns the set after msg. The set it is given is not changed. A
// message that names a collection that is not open, such as a menu that
// outlived it, leaves the set as it is.
func Update(set Set, msg Msg) Set {
	set.Entries = slices.Clone(set.Entries)
	switch msg := msg.(type) {
	case Open:
		return open(set, msg.Collection, cleanDir(msg.Dir))
	case Close:
		if at := set.index(msg.Collection); at >= 0 {
			set.Entries = slices.Delete(set.Entries, at, at+1)
		}
	case Move:
		return move(set, msg.Collection, msg.Before)
	case Located:
		dir := cleanDir(msg.Dir)
		if at := set.index(msg.Collection); at >= 0 && !set.Taken(dir, msg.Collection) {
			set.Entries[at].Dir = dir
		}
	case ChooseEnv:
		if at := set.index(msg.Collection); at >= 0 {
			set.Entries[at].Env = envIndex(msg.Collection, msg.Index)
		}
	}
	return set
}

func open(set Set, col *model.Collection, dir string) Set {
	if set.index(col) >= 0 || set.Taken(dir, nil) {
		return set
	}
	set.Entries = append(set.Entries, Entry{Collection: col, Dir: dir, Env: -1})
	return set
}

func move(set Set, col, before *model.Collection) Set {
	from := set.index(col)
	if from < 0 || col == before || (before != nil && set.index(before) < 0) {
		return set
	}
	entry := set.Entries[from]
	set.Entries = slices.Delete(set.Entries, from, from+1)
	to := len(set.Entries)
	if before != nil {
		to = set.index(before)
	}
	set.Entries = slices.Insert(set.Entries, to, entry)
	return set
}

func envIndex(col *model.Collection, index int) int {
	if index < 0 || index >= len(col.Envs) {
		return -1
	}
	return index
}

// cleanDir keeps "" as "not saved" rather than letting filepath.Clean turn
// it into ".", the working directory.
func cleanDir(dir string) string {
	if dir == "" {
		return ""
	}
	return filepath.Clean(dir)
}

func (set Set) index(col *model.Collection) int {
	return slices.IndexFunc(set.Entries, func(entry Entry) bool { return entry.Collection == col })
}

// ByDir is the open collection saved in dir.
func (set Set) ByDir(dir string) (Entry, bool) {
	dir = cleanDir(dir)
	for _, entry := range set.Entries {
		if dir != "" && entry.Dir == dir {
			return entry, true
		}
	}
	return Entry{}, false
}

// Taken reports whether an open collection other than except is saved in
// dir, so that a save or rename into it has to be refused.
func (set Set) Taken(dir string, except *model.Collection) bool {
	entry, ok := set.ByDir(dir)
	return ok && entry.Collection != except
}

// Owner is the open collection that holds node: a *model.Collection, its
// root folder, or a *model.Folder or *model.Request anywhere below it.
func (set Set) Owner(node any) (Entry, bool) {
	for _, entry := range set.Entries {
		if node == any(entry.Collection) || entry.Collection.Folder.Contains(node) {
			return entry, true
		}
	}
	return Entry{}, false
}

// VarsFor are the variables req is sent with, from its own collection
// only; a request of no open collection has none, never another's.
func (set Set) VarsFor(req *model.Request) ([]model.KV, bool) {
	entry, ok := set.Owner(req)
	if !ok {
		return nil, false
	}
	return entry.VarsFor(req), true
}
