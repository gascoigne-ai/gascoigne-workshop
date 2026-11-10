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
}

// ClientConfigs reads the client config: the collections and servers a
// build offers when no workspace is open.
type ClientConfigs interface {
	Load() (clientconfig.Config, error)
}
