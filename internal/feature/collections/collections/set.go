package collections

import (
	"path/filepath"
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Entry is one open collection: its tree, the folder it is saved in and
// its active environment. The Collection pointer is its identity, since
// open tabs and the tree point into it.
type Entry struct {
	Collection *request.Collection
	// Dir is the folder the collection is saved in, "" until its first
	// save. Two entries never share a folder.
	Dir string
	// Env indexes the active environment in Collection.Envs, -1 for none.
	// It is an index rather than an envs.Selection because a controller
	// reaches another domain only through its model.
	Env int
}

// Environment is the entry's active environment, nil for none.
func (entry Entry) Environment() *request.Environment {
	if entry.Env < 0 || entry.Env >= len(entry.Collection.Envs) {
		return nil
	}
	return &entry.Collection.Envs[entry.Env]
}

// VarsFor are the variables req is sent with: the entry's own collection,
// environment and folders, nothing of another collection.
func (entry Entry) VarsFor(req *request.Request) []request.KV {
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
	Collection *request.Collection
	Dir        string
}

// Close takes Collection out of the set. Closing the last one leaves none;
// what opens then is the caller's choice.
type Close struct{ Collection *request.Collection }

// Move puts Collection before Before, at the end when Before is nil.
type Move struct{ Collection, Before *request.Collection }

// Shift moves Collection By places up the set (negative) or down
// (positive); at an end it stays.
type Shift struct {
	Collection *request.Collection
	By         int
}

// Located records that Collection is now saved in Dir, after its first
// save or a rename. A folder another open collection holds is refused.
type Located struct {
	Collection *request.Collection
	Dir        string
}

// ChooseEnv makes environment Index of Collection the active one, none
// when Index is out of range.
type ChooseEnv struct {
	Collection *request.Collection
	Index      int
}

func (Open) setMsg()      {}
func (Close) setMsg()     {}
func (Move) setMsg()      {}
func (Shift) setMsg()     {}
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
	case Shift:
		return shift(set, msg.Collection, msg.By)
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

func open(set Set, col *request.Collection, dir string) Set {
	if set.index(col) >= 0 || set.Taken(dir, nil) {
		return set
	}
	set.Entries = append(set.Entries, Entry{Collection: col, Dir: dir, Env: -1})
	return set
}

func move(set Set, col, before *request.Collection) Set {
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

func shift(set Set, col *request.Collection, by int) Set {
	from := set.index(col)
	to := from + by
	if from < 0 || to < 0 || to >= len(set.Entries) {
		return set
	}
	var before *request.Collection // nil puts it last
	switch {
	case by < 0:
		before = set.Entries[to].Collection
	case to+1 < len(set.Entries):
		before = set.Entries[to+1].Collection
	}
	return move(set, col, before)
}

func envIndex(col *request.Collection, index int) int {
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

func (set Set) index(col *request.Collection) int {
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
func (set Set) Taken(dir string, except *request.Collection) bool {
	entry, ok := set.ByDir(dir)
	return ok && entry.Collection != except
}

// Owner is the open collection that holds node: its root folder, or a
// folder or request anywhere below it.
func (set Set) Owner(node request.Node) (Entry, bool) {
	for _, entry := range set.Entries {
		if entry.Collection.Folder.Contains(node) {
			return entry, true
		}
	}
	return Entry{}, false
}

// ByCollection is the entry of the open collection col.
func (set Set) ByCollection(col *request.Collection) (Entry, bool) {
	index := set.index(col)
	if index < 0 {
		return Entry{}, false
	}
	return set.Entries[index], true
}

// VarsFor are the variables req is sent with, from its own collection
// only; a request of no open collection has none, never another's.
func (set Set) VarsFor(req *request.Request) ([]request.KV, bool) {
	entry, ok := set.Owner(request.Node{Req: req})
	if !ok {
		return nil, false
	}
	return entry.VarsFor(req), true
}
