package app

import (
	"context"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/systeminfo"
)

// The ports below are the ones a program gives Options that are about the
// desktop: the window declares its own narrow views of the ones it calls (a
// FileDialog and a SystemInfo), and the app keeps the ones it uses itself
// (KeyRepeat and DisplayScale).

// KeyRepeat tells how the desktop repeats a held key. The toolkit's
// desktop/keyrepeat package implements it; a test fake stands in elsewhere.
type KeyRepeat interface {
	// Timing is the wait before the first repeat and the time between
	// repeats; both are 0 when the desktop's setting is unknown.
	Timing() (delay, interval time.Duration)
}

// DisplayScale tells the desktop's scale for the display when the window
// system does not pass it on. The toolkit's desktop/displayscale package
// implements it.
type DisplayScale interface {
	// Factor is the scale (2 for a 2x display), 0 when unknown.
	Factor() float64
}

// FileDialog asks the desktop for a file name. The toolkit's
// desktop/filedialog package implements it; a test fake stands in elsewhere.
type FileDialog interface {
	// Available reports whether there is a dialog to show; without one the
	// caller opens its in-app file browser.
	Available() bool
	// Suggested is the program to install for native dialogs on this
	// desktop ("zenity", "kdialog", "zenity or kdialog"), "" when there is
	// nothing to suggest.
	Suggested() string
	// Choose shows the dialog and returns the chosen path, or
	// filedialog.ErrCancelled. Cancelling ctx closes it. SaveFile asks for a
	// new file named name; OpenFile ignores name.
	Choose(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error)
}

// Clock tells the current time, so toasts, debounces and the caret blink
// can be driven by a fake one.
type Clock interface {
	Now() time.Time
}

// SystemInfo reports what the program runs on: the system, the display and
// the GPU rendering. The toolkit's desktop/systeminfo package implements it.
type SystemInfo interface {
	// Read is the report as it is now; the numbers that change (frame rate,
	// memory) are read at each call.
	Read() systeminfo.Report
}
