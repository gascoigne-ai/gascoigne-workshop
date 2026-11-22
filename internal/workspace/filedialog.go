// Package workspace holds the workspace domain's types and the ports at its
// boundary: what it needs from the desktop, such as the file chooser.
package workspace

import (
	"context"
	"errors"
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
