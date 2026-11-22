// Package workspace holds the workspace feature's types and the ports at its
// boundary: what it needs from the desktop (file chooser, user's files, clock)
// and the remembered-state store.
package workspace

import (
	"context"
	"errors"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
)

// ErrCancelled is returned by a FileDialog when the user closed it.
var ErrCancelled = errors.New("cancelled")

// FileMode is what a FileDialog asks for.
type FileMode uint8

// The modes of a FileDialog.
const (
	OpenFile FileMode = iota // an existing file
	SaveFile                 // a new file, named by the caller's suggestion
)

// FileDialog asks the desktop for a file name.
type FileDialog interface {
	// Available reports whether there is a dialog to show; without one the
	// caller opens its in-app file browser.
	Available() bool
	// Suggested is the program to install for native dialogs on this
	// desktop ("zenity", "kdialog", "zenity or kdialog"), "" when there is
	// nothing to suggest.
	Suggested() string
	// Choose shows the dialog and returns the chosen path, or ErrCancelled.
	// Cancelling ctx closes it. SaveFile asks for a new file named name;
	// OpenFile ignores name.
	Choose(ctx context.Context, title string, mode FileMode, name string) (string, error)
}

// Clock tells the current time, so toasts, debounces and the caret blink
// can be driven by a fake one.
type Clock interface {
	Now() time.Time
}

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
