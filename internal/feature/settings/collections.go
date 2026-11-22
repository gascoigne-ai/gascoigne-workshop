package settings

import "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"

// RememberedCollections are the collections to open at the start, in tree
// order. A file written when one collection was open at a time has no
// Collections; its Workspace and Environment are then that one.
func (s Settings) RememberedCollections() []request.OpenCollection {
	if len(s.Collections) > 0 {
		return s.Collections
	}
	if s.Workspace == "" && s.Environment == "" {
		return nil
	}
	return []request.OpenCollection{{Folder: s.Workspace, Environment: s.Environment}}
}

// RememberCollections records the open collections and drops the older
// single-collection keys, so the file is written in the new form only.
func (s *Settings) RememberCollections(open []request.OpenCollection) {
	s.Collections = open
	s.Workspace, s.Environment = "", ""
}
