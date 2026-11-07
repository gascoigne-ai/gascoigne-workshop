package settingsui

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// FontRole is which text a font is for.
type FontRole int

// The two roles a font can be picked for.
const (
	FontUI   FontRole = iota // labels, buttons, fields
	FontCode                 // bodies, headers, the code views
)

// SuggestionCount bounds the family names offered under a font field.
const SuggestionCount = 3

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
			d.noteFont(role, fmt.Sprintf("No installed font is called %q", name))
			return
		}
		name = family.Name
	}
	d.noteFont(role, "")
	host.Change(func(s *settings.Settings) {
		if role == FontCode {
			s.CodeFont = name
		} else {
			s.UIFont = name
		}
	})
	d.ApplyFonts(host)
}

func (d *Dialog) noteFont(role FontRole, note string) {
	if d.fontNote == nil {
		d.fontNote = map[FontRole]string{}
	}
	d.fontNote[role] = note
}

// FontNote is why a font name was refused, "" when it was not.
func (d *Dialog) FontNote(role FontRole) string { return d.fontNote[role] }

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

// Suggestions are the families to offer for the typed text: none until
// something is typed, since the whole list would not fit the pane.
func (d *Dialog) Suggestions(typed string) []fonts.Family {
	if strings.TrimSpace(typed) == "" {
		return nil
	}
	return fonts.Matching(d.Families, typed, SuggestionCount)
}

// fontPicker is the field of one font role with the families that match
// what was typed, as buttons.
func (d *Dialog) fontPicker(host Host, role FontRole, field *TextField) Node {
	apply := func() { d.SetFont(host, role, field.Text) }
	note, noteColor := d.fontNote[role], w.Error
	if note == "" && len(d.Families) == 0 {
		note, noteColor = "Looking for installed fonts...", w.Muted
	}
	return Col(
		Row(H(w.RowH), Gap(6),
			Field(field, Grow(1), Bg(w.Sunken), Placeholder("Built-in"), OnEnter(apply), host.TextMenu(field, nil)),
			Button("Apply", W(64), OnClick(apply)),
		),
		If(note != "", Row(H(w.NoteH), Text(note, Small(), Color(noteColor), Ellipsis(), Grow(1)))),
		Row(H(w.RowH), Gap(4),
			For(d.Suggestions(field.Text), func(_ int, family fonts.Family) Node {
				return Button(family.Name, H(w.CompactH), OnClick(func() { *field = NewTextField(family.Name); d.SetFont(host, role, family.Name) }))
			}),
		),
	)
}
