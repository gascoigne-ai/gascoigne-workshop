// Package colorpage is the Colors page of the settings dialog: one row per
// customizable color with a hex field, a swatch and a Reset, checked for
// readable contrast. The application is reached through Host.
package colorpage

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Host is what the page needs of the application.
type Host interface {
	// Settings are the settings in effect.
	Settings() settings.Settings
	// Send asks the application to change the settings (prefs.Edit).
	Send(msg prefs.Msg)
	// Color is the color in effect for a token.
	Color(tok w.Token) color.RGBA
}

// Page holds the hex fields of the colors and why a typed one was refused.
type Page struct {
	inputs map[w.Token]*TextField
	notes  map[w.Token]string
}

// Forget drops the fields and notes, so they are rebuilt from the colors in
// effect.
func (p *Page) Forget() { p.inputs, p.notes = nil, nil }

// Note is why the color typed for tok was refused, "" when it was not.
func (p *Page) Note(tok w.Token) string { return p.notes[tok] }

// Input is the hex field of a custom color, created on first use from the
// color in effect.
func (p *Page) Input(host Host, tok w.Token) *TextField {
	if in, ok := p.inputs[tok]; ok {
		return in
	}
	if p.inputs == nil {
		p.inputs = map[w.Token]*TextField{}
	}
	in := new(TextField)
	*in = NewTextField(w.FormatHex(host.Color(tok)))
	p.inputs[tok] = in
	return in
}

func (p *Page) setNote(tok w.Token, note string) {
	if p.notes == nil {
		p.notes = map[w.Token]string{}
	}
	if note == "" {
		delete(p.notes, tok)
		return
	}
	p.notes[tok] = note
}

// edited applies the hex a user typed for customColor, or says why not: a
// color that does not parse, or whose text would be unreadable on it
// (with the ratio), is refused and nothing changes.
func (p *Page) edited(host Host, customColor appearance.CustomColor) {
	text := p.Input(host, customColor.Token).Text
	col, err := w.ParseHex(text)
	if err != nil {
		p.setNote(customColor.Token, "Use #rrggbb")
		return
	}
	need := appearance.RequiredRatio(host.Settings().Contrast)
	if ratio := w.ContrastRatio(col, host.Color(customColor.Text)); ratio < need {
		p.setNote(customColor.Token, fmt.Sprintf("Contrast %.1f:1 is below %.1f:1", ratio, need))
		return
	}
	p.setNote(customColor.Token, "")
	host.Send(prefs.Edit{Edit: func(prefs *settings.Settings) {
		id := appearance.ThemeIDOf(prefs.Theme)
		if prefs.Colors == nil {
			prefs.Colors = map[string]map[string]string{}
		}
		if prefs.Colors[id] == nil {
			prefs.Colors[id] = map[string]string{}
		}
		prefs.Colors[id][string(customColor.Token)] = w.FormatHex(col)
	}})
}

// resetColor drops the override of one color in the current theme.
func (p *Page) reset(host Host, tok w.Token) {
	host.Send(prefs.Edit{Edit: func(prefs *settings.Settings) {
		delete(prefs.Colors[appearance.ThemeIDOf(prefs.Theme)], string(tok))
	}})
	delete(p.inputs, tok)
	p.setNote(tok, "")
}

// resetColors drops every color override of the current theme.
func (p *Page) resetAll(host Host) {
	host.Send(prefs.Edit{Edit: func(prefs *settings.Settings) { delete(prefs.Colors, appearance.ThemeIDOf(prefs.Theme)) }})
	p.Forget()
}

var pageTemplate = markup.Must("colorpage.vml", colorpageSource)

// colorRow is what the template shows of one customizable color.
type colorRow struct {
	Label  string
	Token  w.Token // also the swatch's background: the color in effect
	Input  *TextField
	Border w.Token // the field's border, red while the typed hex is refused
	Note   string  // why the typed hex was refused, "" when it was not
	Edited func()
	Reset  func()
}

// View is the Colors section of the settings menu: one row per
// customizable color with a swatch, its hex and a reset button.
func (p *Page) View(host Host) Node {
	colors := make([]colorRow, 0, len(appearance.CustomColors))
	for _, custom := range appearance.CustomColors {
		colors = append(colors, p.row(host, custom))
	}
	return pageTemplate.MustBuild(markup.Scope{
		"tightGap": w.TightGap,
		"colors":   colors,
		"resetAll": func() { p.resetAll(host) },
		"labelH":   w.LabelH,
		"rowH":     w.RowH,
		"noteH":    w.NoteH,
		"compactH": w.CompactH,
	})
}

func (p *Page) row(host Host, custom appearance.CustomColor) colorRow {
	note := p.notes[custom.Token]
	border := w.Border
	if note != "" {
		border = w.Error
	}
	return colorRow{
		Label:  custom.Label,
		Token:  custom.Token,
		Input:  p.Input(host, custom.Token),
		Border: border,
		Note:   note,
		Edited: func() { p.edited(host, custom) },
		Reset:  func() { p.reset(host, custom.Token) },
	}
}

// Templates, parsed once above.
const (
	colorpageSource = `<col>
  <caption h="{{labelH}}">Colors</caption>
  <col for="color in colors">
    <row h="{{rowH}}" gap="{{tightGap}}">
      <text small w="76">{{color.Label}}</text>
      <row w="20" h="{{noteH}}" bg="{{color.Token}}" control="swatch"/>
      <field state="color.Input" w="70" h="{{compactH}}" bg="sunken" border="{{color.Border}}"
             on-change="color.Edited" text-menu="color.Input"/>
      <button w="44" h="{{compactH}}" on-click="color.Reset">Reset</button>
    </row>
    <note if="color.Note" tone="error">{{color.Note}}</note>
  </col>
  <row h="{{rowH}}"><button w="110" on-click="resetAll">Reset colors</button></row>
</col>
`
)
