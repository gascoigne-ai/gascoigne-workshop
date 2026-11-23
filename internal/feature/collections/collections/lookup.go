package collections

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// historyKeySep joins a collection's folder and a request path in a
// history key. A key is never split again, and the folder is absolute, so
// it cannot be mistaken for a bare path.
const historyKeySep = "#"

// PathOf is the open collection that holds req and req's remembered path in
// it; ok is false when req is in no open collection.
func (set Set) PathOf(req *request.Request) (entry Entry, path string, ok bool) {
	entry, ok = set.Owner(request.Node{Req: req})
	if !ok {
		return Entry{}, "", false
	}
	return entry, tree.PathOf(&entry.Collection.Folder, req), true
}

// Dirs are the folders of the open collections in tree order, "" for one
// not saved, as state.CollectionAt takes them.
func (set Set) Dirs() []string {
	dirs := make([]string, len(set.Entries))
	for i, entry := range set.Entries {
		dirs[i] = entry.Dir
	}
	return dirs
}

// ExpandedPaths are the paths of the folders of entry's collection that
// isOpen says are open, in tree order.
func (set Set) ExpandedPaths(entry Entry, isOpen func(*request.Folder) bool) []string {
	var paths []string
	tree.WalkFolders(&entry.Collection.Folder, "", func(path string, folder *request.Folder) {
		if isOpen(folder) {
			paths = append(paths, path)
		}
	})
	return paths
}

// FoldersAt are the folders of col at the remembered paths, in the order of
// paths; a path with no folder gives none, and a nil col has none.
func (set Set) FoldersAt(col *request.Collection, paths []string) []*request.Folder {
	if col == nil {
		return nil
	}
	var folders []*request.Folder
	for _, wanted := range paths {
		tree.WalkFolders(&col.Folder, "", func(path string, folder *request.Folder) {
			if path == wanted {
				folders = append(folders, folder)
			}
		})
	}
	return folders
}

// HistoryKey is what req's saved responses are kept under: its
// collection's folder and its path, since two collections can hold the
// same path. A collection not saved yet has no folder, so its key is the
// bare path. "" when req is in no open collection.
func (set Set) HistoryKey(req *request.Request) string {
	owner, path, ok := set.PathOf(req)
	if path == "" || !ok || owner.Dir == "" {
		return path
	}
	return owner.Dir + historyKeySep + path
}

// LegacyHistoryKey is the bare path req's responses were kept under while
// one collection was open at a time, for a request of the first open
// collection, which is where that one collection is now; "" otherwise.
func (set Set) LegacyHistoryKey(req *request.Request) string {
	owner, path, ok := set.PathOf(req)
	if !ok || owner.Dir == "" || set.Entries[0].Collection != owner.Collection {
		return ""
	}
	return path
}
