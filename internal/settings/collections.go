package settings

// OpenCollection is one collection the window had open: the folder it is
// saved in and the name of its active environment ("" for none). An entry
// without a folder stands for the collection the program starts with when
// none is saved: the built-in sample or the client config.
type OpenCollection struct {
	Folder      string `json:"folder,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// RememberedCollections are the collections to open at the start, in tree
// order. A file written when one collection was open at a time has no
// Collections; its Workspace and Environment are then that one.
func (s Settings) RememberedCollections() []OpenCollection {
	if len(s.Collections) > 0 {
		return s.Collections
	}
	if s.Workspace == "" && s.Environment == "" {
		return nil
	}
	return []OpenCollection{{Folder: s.Workspace, Environment: s.Environment}}
}

// RememberCollections records the open collections and drops the older
// single-collection keys, so the file is written in the new form only.
func (s *Settings) RememberCollections(open []OpenCollection) {
	s.Collections = open
	s.Workspace, s.Environment = "", ""
}
