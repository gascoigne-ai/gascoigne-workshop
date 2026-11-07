// Package port holds what the collection domain needs from the outside:
// where a collection is saved.
package port

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// Store reads and writes a collection in a folder.
type Store interface {
	// Load reads the collection in dir.
	Load(dir string) (model.Collection, error)
	// Save writes the collection to dir.
	Save(dir string, collection model.Collection) error
}
