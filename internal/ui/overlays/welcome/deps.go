package welcome

import (
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the component asks of the application.
type Deps struct {
	Dialogs Dialogs
	Window  Window
}

// Dialogs shows the dialog of the window.
type Dialogs interface {
	// Open shows state as the dialog of the window, with content as its own
	// body, in a panel width wide. The returned func closes that dialog and
	// no other.
	Open(state widget.DialogState, content func(ctx *w.Ctx) Node, width int) (close func())
}

// Window is the window the notes are shown in.
type Window interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
}
