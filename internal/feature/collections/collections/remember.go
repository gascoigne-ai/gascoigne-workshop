package collections

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/openset"
	"path/filepath"
)

// AbsDir is the folder a collection is identified by: absolute, so one
// folder typed two ways is still one open collection.
func AbsDir(dir string) string {
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return dir
}

// At is the open collection a remembered folder names: the one saved
// there, or for "" the one not saved.
func (set Set) At(folder string) (Entry, bool) {
	if folder != "" {
		return set.ByDir(AbsDir(folder))
	}
	for _, entry := range set.Entries {
		if entry.Dir == "" {
			return entry, true
		}
	}
	return Entry{}, false
}

// Remembered is the list the settings keep of the open set, in tree order.
// Unreadable are remembered folders that failed to load: they are not open
// now, but they are not forgotten either.
func Remembered(set Set, unreadable []openset.OpenCollection) []openset.OpenCollection {
	remembered := make([]openset.OpenCollection, 0, len(set.Entries))
	for _, entry := range set.Entries {
		item := openset.OpenCollection{Folder: entry.Dir}
		if env := entry.Environment(); env != nil {
			item.Environment = env.Name
		}
		remembered = append(remembered, item)
	}
	for _, item := range unreadable {
		if _, open := set.At(item.Folder); !open {
			remembered = append(remembered, item)
		}
	}
	return remembered
}

// WithEnvironments is remembered with the active environment of each entry
// that names an open collection brought up to date; it reports whether any
// changed. Entries and their order are kept, so a run that opened other
// folders does not replace the list the next ordinary start opens.
func WithEnvironments(set Set, remembered []openset.OpenCollection) ([]openset.OpenCollection, bool) {
	updated := append([]openset.OpenCollection(nil), remembered...)
	changed := false
	for i, item := range updated {
		entry, ok := set.At(item.Folder)
		if !ok {
			continue
		}
		name := ""
		if env := entry.Environment(); env != nil {
			name = env.Name
		}
		if item.Environment != name {
			updated[i].Environment, changed = name, true
		}
	}
	return updated, changed
}
