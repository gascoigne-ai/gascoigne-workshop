package formpage

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// SetFont picks the family called name for role, applies it and saves the
// choice. An empty name goes back to the built-in font; a name that no
// installed family has is explained next to the field and changes nothing.
func (p *Page) SetFont(host Host, role form.FontRole, name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		family, ok := fonts.Find(p.Families, name)
		if !ok {
			host.Send(prefs.Toast{Text: fmt.Sprintf("No installed font is called %q", name), Level: severity.Error})
			return
		}
		name = family.Name
	}
	host.Send(prefs.Change{Edit: func(current *settings.Settings) {
		if role == form.FontCode {
			current.CodeFont = name
		} else {
			current.UIFont = name
		}
	}})
	p.ApplyFonts(host)
}

// FontsScanned receives the installed families and applies the saved
// choices, which could not be resolved before.
func (p *Page) FontsScanned(host Host, families []fonts.Family) {
	p.Families = families
	p.ApplyFonts(host)
}

// ApplyFonts loads the saved families for both roles. A name that is not
// installed (any more) uses the built-in font and stays saved, so the font
// returns when it is installed again.
func (p *Page) ApplyFonts(host Host) {
	current := host.Settings()
	var chosen [2]w.FontData
	for _, role := range []form.FontRole{form.FontUI, form.FontCode} {
		family, ok := fonts.Find(p.Families, form.SavedFont(current, role))
		if !ok {
			continue
		}
		regular, bold, err := host.FontStore().Load(family)
		if err != nil {
			host.Send(prefs.SetStatus{Text: err.Error()})
			continue
		}
		chosen[role] = w.FontData{Regular: regular, Bold: bold}
	}
	if err := host.SetFonts(chosen[form.FontUI], chosen[form.FontCode]); err != nil {
		host.Send(prefs.SetStatus{Text: "Font not usable: " + err.Error()})
		_ = host.SetFonts(w.FontData{}, w.FontData{}) // the built-in fonts always load
	}
	host.Send(prefs.MarkDirty{})
}

// ScanFonts reads the installed fonts in the background and hands them to
// the UI goroutine.
func (p *Page) ScanFonts(host Host) {
	store := host.FontStore()
	go func() {
		families := store.Scan()
		host.Send(prefs.Later{Func: func() { p.FontsScanned(host, families) }})
	}()
}
