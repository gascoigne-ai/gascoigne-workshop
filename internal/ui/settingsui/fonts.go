package settingsui

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// FontRole is which text a font is for.
type FontRole int

// The two roles a font can be picked for.
const (
	FontUI   FontRole = iota // labels, buttons, fields
	FontCode                 // bodies, headers, the code views
)

// savedFont is the family name stored for role.
func savedFont(s settings.Settings, role FontRole) string {
	if role == FontCode {
		return s.CodeFont
	}
	return s.UIFont
}

// SetFont picks the family called name for role, applies it and saves the
// choice. An empty name goes back to the built-in font; a name that no
// installed family has is explained next to the field and changes nothing.
func (d *Dialog) SetFont(host Host, role FontRole, name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		family, ok := fonts.Find(d.Families, name)
		if !ok {
			host.Toast(fmt.Sprintf("No installed font is called %q", name), true)
			return
		}
		name = family.Name
	}
	host.Change(func(s *settings.Settings) {
		if role == FontCode {
			s.CodeFont = name
		} else {
			s.UIFont = name
		}
	})
	d.ApplyFonts(host)
}

// FontsScanned receives the installed families and applies the saved
// choices, which could not be resolved before.
func (d *Dialog) FontsScanned(host Host, families []fonts.Family) {
	d.Families = families
	d.ApplyFonts(host)
}

// ApplyFonts loads the saved families for both roles. A name that is not
// installed (any more) uses the built-in font and stays saved, so the font
// returns when it is installed again.
func (d *Dialog) ApplyFonts(host Host) {
	current := host.Settings()
	var chosen [2]w.FontData
	for _, role := range []FontRole{FontUI, FontCode} {
		family, ok := fonts.Find(d.Families, savedFont(current, role))
		if !ok {
			continue
		}
		regular, bold, err := fonts.Load(family)
		if err != nil {
			host.SetStatus(err.Error())
			continue
		}
		chosen[role] = w.FontData{Regular: regular, Bold: bold}
	}
	if err := host.SetFonts(chosen[FontUI], chosen[FontCode]); err != nil {
		host.SetStatus("Font not usable: " + err.Error())
		_ = host.SetFonts(w.FontData{}, w.FontData{}) // the built-in fonts always load
	}
	host.MarkDirty()
}

// ScanFonts reads the installed fonts in the background and hands them to
// the UI goroutine.
func (d *Dialog) ScanFonts(host Host) {
	home, _ := os.UserHomeDir()
	go func() {
		families := fonts.Scan(fonts.SystemDirs(runtime.GOOS, home)...)
		host.Later(func() { d.FontsScanned(host, families) })
	}()
}
