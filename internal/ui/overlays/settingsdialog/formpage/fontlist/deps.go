package fontlist

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
)

// FontStore finds installed fonts and reads their files. The toolkit's
// desktop/fonts package implements it; the tests of this package use it too.
type FontStore interface {
	// Scan lists the installed families, sorted by name.
	Scan() []fonts.Family
	// Load reads the regular and bold files of family.
	Load(family fonts.Family) (regular, bold []byte, err error)
}

// Deps is what the list asks of the application: reads, and Send for
// everything it asks to be done.
type Deps interface {
	// Settings are the settings in effect.
	Settings() settings.Settings
	// Send asks the application to do something (see package prefs).
	Send(msg prefs.Msg)
	// FontStore finds installed fonts and reads their files.
	FontStore() FontStore
	// SetFonts switches the interface and code fonts; the error says why
	// they could not be used.
	SetFonts(ui, code w.FontData) error
}

// Window is the window the list floats in.
type Window interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Pointer is the pointer position in the window, where the list opens.
	Pointer() (x, y int)
}

// Host is what the list needs from the application.
type Host interface {
	Deps
	Window
}
