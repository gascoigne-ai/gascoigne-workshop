package ui

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// fontRole is which text a font is for.
type fontRole int

// The two roles a font can be picked for.
const (
	fontUI   fontRole = iota // labels, buttons, fields
	fontCode                 // bodies, headers, the code views
)

// fontSuggestions bounds the family names offered under a font field.
const fontSuggestions = 6

// savedFont is the family name stored for role.
func (a *App) savedFont(role fontRole) string {
	if role == fontCode {
		return a.settings.CodeFont
	}
	return a.settings.UIFont
}

// setFont picks the family called name for role, applies it and saves
// the choice. An empty name goes back to the built-in font; a name that no
// installed family has is explained next to the field and changes nothing.
func (a *App) setFont(role fontRole, name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		family, ok := fonts.Find(a.families, name)
		if !ok {
			a.noteFont(role, fmt.Sprintf("No installed font is called %q", name))
			return
		}
		name = family.Name
	}
	a.noteFont(role, "")
	a.saveSettings(func(s *settings.Settings) {
		if role == fontCode {
			s.CodeFont = name
		} else {
			s.UIFont = name
		}
	})
	a.applyFonts()
}

func (a *App) noteFont(role fontRole, note string) {
	if a.fontNotes == nil {
		a.fontNotes = map[fontRole]string{}
	}
	a.fontNotes[role] = note
}

// fontsScanned receives the installed families and applies the saved
// choices, which could not be resolved before.
func (a *App) fontsScanned(families []fonts.Family) {
	a.families = families
	a.applyFonts()
}

// applyFonts loads the saved families for both roles. A name that is not
// installed (any more) uses the built-in font and stays saved, so the font
// returns when it is installed again.
func (a *App) applyFonts() {
	var chosen [2]w.FontData
	for _, role := range []fontRole{fontUI, fontCode} {
		family, ok := fonts.Find(a.families, a.savedFont(role))
		if !ok {
			continue
		}
		regular, bold, err := fonts.Load(family)
		if err != nil {
			a.status = err.Error()
			continue
		}
		chosen[role] = w.FontData{Regular: regular, Bold: bold}
	}
	if err := a.ctx.SetFonts(chosen[fontUI], chosen[fontCode]); err != nil {
		a.status = "Font not usable: " + err.Error()
		_ = a.ctx.SetFonts(w.FontData{}, w.FontData{}) // the built-in fonts always load
	}
	a.dirty = true
}

// scanFonts reads the installed fonts in the background and hands them to
// the UI goroutine.
func (a *App) scanFonts() {
	home, _ := os.UserHomeDir()
	go func() {
		families := fonts.Scan(fonts.SystemDirs(runtime.GOOS, home)...)
		a.later <- func() { a.fontsScanned(families) }
	}()
}

// fontPicker is the field of one font role with the families that match
// what was typed, as buttons.
func (a *App) fontPicker(role fontRole, field *TextField) Node {
	apply := func() { a.setFont(role, field.Text) }
	note, noteColor := a.fontNotes[role], w.Error
	if note == "" && len(a.families) == 0 {
		note, noteColor = "Looking for installed fonts...", w.Muted
	}
	return Col(
		Row(H(w.RowH), Gap(6),
			Field(field, Grow(1), Bg(w.Sunken), Placeholder("Built-in"), OnEnter(apply), a.textMenuOpt(field, nil)),
			Button("Apply", W(64), OnClick(apply)),
		),
		If(note != "", Row(H(w.NoteH), Text(note, Small(), Color(noteColor), Ellipsis(), Grow(1)))),
		Row(H(w.RowH), Gap(4),
			For(fonts.Matching(a.families, field.Text, fontSuggestions), func(_ int, family fonts.Family) Node {
				return Button(family.Name, H(w.CompactH), OnClick(func() { *field = NewTextField(family.Name); a.setFont(role, family.Name) }))
			}),
		),
	)
}
