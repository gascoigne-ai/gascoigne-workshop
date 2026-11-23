package settingsdialog

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/systeminfo"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the component asks of the application, each a narrow
// interface the host implements, plus the values it keeps for itself: the
// settings file (path, store and whether it may be written) and the sources
// of the font and System pages.
type Deps struct {
	Settings Settings
	Window   Window
	Files    Files
	Dialogs  Dialogs
	Bus      Bus
	Looks    Looks
	Keys     Keys

	Path   string                 // the settings file, "" keeps settings in memory only
	Store  settings.SettingsStore // reads and writes that file
	Access storage.Access         // ReadOnly when a newer build wrote the file
	Fonts  fontlist.FontStore     // installed fonts
	System SystemSource           // what the System page shows; nil has no report
}

// SystemSource reports what the program runs on.
type SystemSource interface {
	// Read is the report as it is now.
	Read() systeminfo.Report
}

// Settings are the settings in effect and what changing them touches: the
// theme, the open tabs and the protocols stay with the application.
type Settings interface {
	// Settings are the settings in effect.
	Settings() settings.Settings
	// Edit changes the settings in effect and everything that follows from
	// the change. The error says why the theme could not be built.
	Edit(edit func(*settings.Settings)) error
	// Apply makes current the settings in effect, with everything that
	// follows from them.
	Apply(current settings.Settings)
	// SetScale sets the interface scale while the slider is dragged, with no
	// other effect of a settings change.
	SetScale(percent int)
}

// Window is the window around the dialog.
type Window interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Pointer is the pointer position in the window.
	Pointer() (x, y int)
	// Toast shows a message over the window.
	Toast(text string, level severity.Level)
	// SetStatus writes the status line.
	SetStatus(text string)
	// OpenMenu shows a menu of items at a point of the window.
	OpenMenu(x, y int, items []widget.MenuItem)
	// Stale says the view built this frame no longer shows what a message
	// changed.
	Stale()
	// Redraw says the screen must be drawn again.
	Redraw()
}

// Files asks the user for a file.
type Files interface {
	// Pick asks for a file and calls done with its path. The returned func
	// takes the question back.
	Pick(title string, done func(path string)) (cancel func())
}

// Dialogs is the one dialog the window shows over everything.
type Dialogs interface {
	// Open shows state in place of any open dialog. The returned func closes
	// that dialog; it does nothing when another one replaced it.
	Open(state widget.DialogState) (close func())
}

// Bus runs work on the UI goroutine.
type Bus interface {
	// Dispatch runs run on the UI goroutine. It is safe from any goroutine.
	Dispatch(run func())
}

// Looks are what the pages read of the toolkit and the desktop.
type Looks interface {
	// Color is the color in effect for a token.
	Color(token w.Token) color.RGBA
	// SetFonts switches the interface and code fonts; the error says why
	// they could not be used.
	SetFonts(ui, code w.FontData) error
	// HostOS is the operating system shortcut text follows.
	HostOS() string
	// CopyText puts text on the clipboard.
	CopyText(text string)
}

// Keys is the binding of every command.
type Keys interface {
	// Keymap is the resolved binding of every command, in display order.
	Keymap() keymap.Keymap
}
