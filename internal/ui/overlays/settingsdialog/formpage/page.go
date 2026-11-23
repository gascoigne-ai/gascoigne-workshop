// Package formpage is the form of the settings dialog: one row per setting
// of a category (or of every category that matches a search), the font
// drop-down and the typed fields. What a user's action means is decided by
// the pure Update of package form; this package draws the rows and carries
// out the effects. The font drop-down is its child, package fontlist. The
// dialog shell (package settingsdialog) imports this package and is its Host.
package formpage

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Host is what the form needs from the application: what the font list
// needs, and OpenPage.
type Host interface {
	fontlist.Host
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

	Fonts fontlist.List // the font drop-down of the face rows
}

// Fresh forgets everything the user did in the form, as closing the dialog
// does; the fonts already read stay.
func (p *Page) Fresh() {
	p.Forget()
	p.Fonts.Fresh()
}

// Reload resets what depends on the settings in effect: typed text and the
// fonts.
func (p *Page) Reload(host Host) {
	p.Forget()
	p.Fonts.Apply(host)
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
