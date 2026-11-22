// Package formpage is the form of the settings dialog: one row per setting
// of a category (or of every category that matches a search), the font
// drop-down and the typed fields. What a user's action means is decided by
// the pure Update of package form; this package draws the rows and carries
// out the effects. The dialog shell (package settingsui) does not import it,
// because a ui package imports no other ui package: the application draws
// this page for the shell, through the shell's Host.
package formpage

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Host is what the form needs from the application: reads, and Send for
// everything it asks to be done.
type Host interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Settings are the settings in effect.
	Settings() settings.Settings
	// Send asks the application to do something (see package prefs).
	Send(msg prefs.Msg)
	// FontStore finds installed fonts and reads their files.
	FontStore() settings.FontStore
	// SetFonts switches the interface and code fonts; the error says why
	// they could not be used.
	SetFonts(ui, code w.FontData) error
	// Pointer is the pointer position in the window, where a dropdown opens.
	Pointer() (x, y int)
	// OpenPage shows the dialog page with the given id: the one that edits
	// a setting a form row only leads to.
	OpenPage(id string)
}

// Page is the state of the form: the text typed into the fields, why a typed
// value was refused (the form Model), the interface scale slider and the font
// drop-down. The zero value is ready.
type Page struct {
	model  form.Model
	inputs map[string]*TextField // by key; the text of the fields of text and number settings
	scale  SliderState

	Families   []settings.Family // installed font families, once scanned
	FontSearch TextField         // narrows the font dropdown
	list       fontList
}

// Fresh forgets everything the user did in the form, as closing the dialog
// does; the fonts already read stay.
func (p *Page) Fresh() {
	p.Forget()
	p.FontSearch = NewTextField("")
	p.list = fontList{faces: p.list.faces}
}

// Reload resets what depends on the settings in effect: typed text and the
// fonts.
func (p *Page) Reload(host Host) {
	p.Forget()
	p.ApplyFonts(host)
}

// scope is the sizes of the faces the context draws with plus the view's own
// names, so a template reads both.
func scope(sizes w.Sizes, own markup.Scope) markup.Scope {
	merged := sizes.Scope()
	for name, value := range own {
		merged[name] = value
	}
	return merged
}
