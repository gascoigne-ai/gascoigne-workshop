// Package port holds what the collection domain needs from the outside:
// where a collection is saved and where the client config is read from.
package port

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/clientconfig"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Store reads and writes a collection in a folder.
type Store interface {
	// Load reads the collection in dir.
	Load(dir string) (model.Collection, error)
	// Save writes the collection to dir.
	Save(dir string, collection model.Collection) error
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

// ClientConfigs reads the client config: the collections and servers a
// build offers when no workspace is open.
type ClientConfigs interface {
	Load() (clientconfig.Config, error)
}
