// Package port holds what the workspace needs from the desktop: the file
// chooser.
package port

import (
	"context"
	"errors"
)

// ErrCancelled is returned by a FileDialog when the user closed it.
var ErrCancelled = errors.New("cancelled")

// FileDialog asks the desktop for a file name.
type FileDialog interface {
	// Available reports whether there is a dialog to show; without one the
	// caller asks for the path as text.
	Available() bool
	// Choose shows the dialog and returns the chosen path, or ErrCancelled.
	// Cancelling ctx closes it. save asks for a new file named name.
	Choose(ctx context.Context, title string, save bool, name string) (string, error)
}
