// Package prefs holds what the settings pages ask of the application, as
// typed messages. A page sends one through its host's Send and the
// application applies it; the host's other methods are only reads.
package prefs

import "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"

// Msg is one thing a settings page asks for.
type Msg interface{ prefsMsg() }

// MenuItem is one row of a menu a page asks the application to open.
type MenuItem struct {
	Label string
	Hint  string // what the item is and what it expects, shown under the label
	Run   func()
}

// Change edits the settings from the form, applies them, saves the file as
// the user has chosen and rebuilds the Colors page.
type Change struct{ Edit func(*settings.Settings) }

// Edit is Change from the Colors and Shortcuts pages, which keep their own
// fields and notes.
type Edit struct{ Edit func(*settings.Settings) }

// ScaleLive applies an interface scale while its slider is dragged,
// without saving; Save writes the file when the drag ends.
type ScaleLive struct{ Percent int }

// Save writes the settings in effect as the user has chosen.
type Save struct{}

// Replace makes Settings the ones in effect, with everything that follows
// from them, without writing the file.
type Replace struct{ Settings settings.Settings }

// Persist writes the settings in effect to the file now.
type Persist struct{}

// MarkUnsaved and MarkSaved record what the dialog did with the file itself.
type (
	MarkUnsaved struct{}
	MarkSaved   struct{}
)

// Toast shows a message over the window.
type Toast struct {
	Text  string
	IsErr bool
}

// SetStatus replaces the status line.
type SetStatus struct{ Text string }

// OpenMenu opens a menu at a point of the window.
type OpenMenu struct {
	X, Y  int
	Items []MenuItem
}

// PickFile asks the user for a file and calls Done with its path.
type PickFile struct {
	Title string
	Done  func(path string)
}

// Close closes the settings dialog.
type Close struct{}

// MarkDirty asks for a redraw.
type MarkDirty struct{}

// Later runs Func on the UI goroutine.
type Later struct{ Func func() }

func (Change) prefsMsg()      {}
func (Edit) prefsMsg()        {}
func (ScaleLive) prefsMsg()   {}
func (Save) prefsMsg()        {}
func (Replace) prefsMsg()     {}
func (Persist) prefsMsg()     {}
func (MarkUnsaved) prefsMsg() {}
func (MarkSaved) prefsMsg()   {}
func (Toast) prefsMsg()       {}
func (SetStatus) prefsMsg()   {}
func (OpenMenu) prefsMsg()    {}
func (PickFile) prefsMsg()    {}
func (Close) prefsMsg()       {}
func (MarkDirty) prefsMsg()   {}
func (Later) prefsMsg()       {}
