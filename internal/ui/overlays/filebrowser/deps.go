package filebrowser

import (
	"context"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
)

// Deps is what the component asks of the application, each a narrow
// interface the application answers.
type Deps struct {
	Desktop Desktop
	Folders Folders
	Bus     Bus
	Window  Window
	Dialogs Dialogs
}

// Desktop is the file dialog of the desktop, when it has a program for one.
type Desktop interface {
	// Available reports whether there is a dialog to show; without one the
	// in-app browser is used.
	Available() bool
	// Suggested is the program to install for native dialogs, "" when there
	// is nothing to suggest.
	Suggested() string
	// Choose shows the dialog and returns the chosen path, or
	// filedialog.ErrCancelled. Cancelling ctx closes it.
	Choose(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error)
}

// Folders is the file system the browser lists.
type Folders interface {
	// Home is the user's home folder: where the browser opens.
	Home() (string, error)
	// List is what dir holds, in no particular order.
	List(dir string) ([]workspace.FolderEntry, error)
}

// Bus runs work off the UI goroutine and brings its answer back to it.
type Bus interface {
	// Run starts cmd on its own goroutine; the message it returns is
	// dispatched on the UI goroutine.
	Run(ctx context.Context, cmd bus.Cmd)
}

// Window is the window the browser is shown in.
type Window interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Toast shows a message over the window.
	Toast(text string, level severity.Level)
}

// Dialogs asks questions over the browser.
type Dialogs interface {
	// Confirm asks a yes/no question; run happens only on its primary button.
	Confirm(title, body, okLabel string, run func())
	// Asking reports whether a dialog is open over the browser.
	Asking() bool
}
