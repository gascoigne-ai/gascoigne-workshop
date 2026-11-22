package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// varsKind is which variables the Vars pane edits.
type varsKind int

const (
	varsActive     varsKind = iota // the active environment's, else the focused collection's
	varsFolder                     // one folder's
	varsCollection                 // a collection's own, although an environment is active
)

// varsTarget is what the Vars pane edits: the kind says which of folder and
// collection is set, so the pane never holds both. The zero value is
// varsActive.
type varsTarget struct {
	kind       varsKind
	folder     *request.Folder
	collection *request.Collection
}

func folderVars(folder *request.Folder) varsTarget {
	return varsTarget{kind: varsFolder, folder: folder}
}

func collectionVars(col *request.Collection) varsTarget {
	return varsTarget{kind: varsCollection, collection: col}
}

// rows are the variables to edit; entry is the focused collection, which only
// an active target looks at. With nothing open they are a scratch list that
// is kept nowhere.
func (target varsTarget) rows(entry collections.Entry) *[]request.KV {
	switch target.kind {
	case varsFolder:
		return &target.folder.Vars
	case varsCollection:
		return &target.collection.Vars
	}
	if entry.Collection == nil {
		return &[]request.KV{}
	}
	if env := entry.Environment(); env != nil {
		return &env.Vars
	}
	return &entry.Collection.Vars
}

// caption names what is edited; env is the active environment, nil for none.
func (target varsTarget) caption(env *request.Environment) string {
	switch {
	case target.kind == varsFolder:
		return "Folder: " + target.folder.Name
	case target.kind == varsActive && env != nil:
		return "Environment: " + env.Name
	}
	return "Collection variables"
}

// refersTo reports whether the variables are col's own or those of a folder
// inside it, so closing col has to reset the target.
func (target varsTarget) refersTo(col *request.Collection) bool {
	switch target.kind {
	case varsFolder:
		return col.Folder.Contains(request.Node{Dir: target.folder})
	case varsCollection:
		return target.collection == col
	}
	return false
}
