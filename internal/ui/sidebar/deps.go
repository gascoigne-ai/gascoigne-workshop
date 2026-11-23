package sidebar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/sentlog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the sidebar asks of the application, each a narrow
// interface the sidebar declares itself. The sidebar only decides and
// draws; the store and window work behind these stays in the application.
// A picker for where a node goes is not here: it is a sibling overlay, so
// Nodes.Pick asks the application to open it.
type Deps struct {
	Collections Collections
	Requests    Requests
	Nodes       Nodes
	Drops       Drops
	Folders     Folders
	Extras      Extras
	Collection  CollectionOps
	Workspace   Workspace
	Menus       Menus
	Sent        sentlog.Host
}

// Collections is the open collections the tree lists.
type Collections interface {
	// Set is the open collections, in tree order.
	Set() collections.Set
	// Focused is the collection the chip menu acts on, nil when none is open.
	Focused() *request.Collection
}

// Requests shows a request the user chose in the tree.
type Requests interface{ Show(req *request.Request) }

// Nodes acts on one folder or request of the tree.
type Nodes interface {
	Rename(node request.Node)
	Delete(node request.Node)
	Duplicate(node request.Node)
	// Pick asks where node goes and then moves or copies it there.
	Pick(node request.Node, mode transfer.Mode)
}

// Drops moves or copies a node dropped on another row.
type Drops interface{ Drop(move transfer.Transfer) }

// ImportKind is the format of the clipboard text an import reads.
type ImportKind int

// The formats a folder can import.
const (
	ImportCurl ImportKind = iota
	ImportOpenAPI
	ImportPostman
)

// Folders is what a folder's menu does inside it.
type Folders interface {
	NewRequest(dir *request.Folder)
	NewFolder(dir *request.Folder)
	Import(dir *request.Folder, kind ImportKind)
	Export(dir *request.Folder)
}

// Extras is the rest of a row's menu.
type Extras interface {
	CopyURL(req *request.Request)
	EditVars(dir *request.Folder)
}

// CollectionOps is what a collection's root menu does to a whole collection.
type CollectionOps interface {
	Rename(col *request.Collection)
	// Move moves col by one place: -1 up, 1 down.
	Move(col *request.Collection, by int)
	Close(col *request.Collection)
	Delete(col *request.Collection)
}

// Workspace is what the Collections chip's menu does to the workspace.
type Workspace interface {
	NewCollection()
	OpenFolder()
	Save(col *request.Collection)
}

// Menus shows a context menu at the pointer position.
type Menus interface {
	Open(posX, posY int, items []widget.MenuItem)
}
