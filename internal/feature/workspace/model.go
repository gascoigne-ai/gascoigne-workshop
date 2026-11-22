// Package workspace holds the workspace feature's types and the ports at its
// boundary: what it needs from the desktop (the user's files)
// and the remembered-state store.
package workspace

import (
	"context"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
)

// FolderEntry is one name in a folder.
type FolderEntry struct {
	Name string
	Dir  bool // a folder, or a link to one, so the browser can open it
}

// Folders is the user's file system: the listings of the in-app file
// browser (the file chooser of a desktop without a dialog program) and the
// files the user names, such as a saved response body, an exported
// collection, the file a request sends and a screenshot.
//
// The App reaches the user's files only through it, off the UI goroutine.
// It was widened rather than a second port added: it is already the user's
// file system at the same boundary and already wired, and a second port
// would be a second adapter over the same disk. Revisit when it outgrows
// the browser and named files (then split it by caller).
type Folders interface {
	// List is what dir holds, in no particular order.
	List(dir string) ([]FolderEntry, error)
	// Home is the user's home folder: where the browser opens, and what a
	// typed ~ means.
	Home() (string, error)
	// Write replaces the file at path with data, creating it when missing.
	Write(ctx context.Context, path string, data []byte) error
	// IsFile reports whether path names a file that is not a folder; a
	// path that cannot be looked at is not one.
	IsFile(ctx context.Context, path string) bool
}

// StateStore keeps what the app remembers between runs.
type StateStore interface {
	// Load reads the state at path; a missing file or an empty path is the
	// empty state.
	Load(path string) (state.State, error)
	// Save writes the state to path; an empty path keeps it in memory.
	Save(path string, current state.State) error
}
