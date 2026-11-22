// Package collection is the root of the collections feature: the sample
// collections a build starts from, the collection tree rows and the Store
// port, where a collection is saved. The package is named collection, not
// collections, because the sub-package collections (the open set) already
// owns that name and every caller reads collection.Sample().
package collection

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Store reads and writes a collection in a folder.
type Store interface {
	// Load reads the collection in dir.
	Load(dir string) (request.Collection, error)
	// Save writes the collection to dir.
	Save(dir string, collection request.Collection) error
	// Create makes an empty collection called name in a new folder below
	// parent and returns that folder.
	Create(parent, name string) (dir string, err error)
	// Rename gives the collection in dir a new name and moves it to a folder
	// of that name; it returns the folder it is in now.
	Rename(dir, name string) (newDir string, err error)
	// Delete removes the collection in dir from the disk: its own files, not
	// the files of the user.
	Delete(dir string) error
	// List is the folders below parent that hold a collection.
	List(parent string) ([]string, error)
}
