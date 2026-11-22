//go:build release

package collection

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// Sample is what a release build starts with when no workspace and no
// client config exists: an empty collection. It ships no example
// endpoints.
func Sample() model.Collection {
	return model.Collection{Folder: model.Folder{Name: "Workshop"}}
}
