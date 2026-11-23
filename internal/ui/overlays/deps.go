package overlays

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Deps is what the composite asks of the application. The children's own
// Deps (dialogs, toasts, window) are answered by the composite itself; only
// what no overlay can answer comes in here.
type Deps struct {
	// Ctx is the toolkit context: the context menu lives in it.
	Ctx     *w.Ctx
	Window  Window
	Clock   Clock
	Desktop filebrowser.Desktop
	Folders filebrowser.Folders
	Bus     filebrowser.Bus

	// Settings are the deps of the settings dialog, which the composite
	// builds and holds; the application answers them.
	Settings settingsdialog.Deps
}

// Window is the window the overlays are drawn over.
type Window interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Stale says the view built this frame no longer shows what changed.
	Stale()
	// Redraw says the screen must be drawn again.
	Redraw()
}

// Clock tells the time of the frame.
type Clock interface {
	Now() time.Time
}
