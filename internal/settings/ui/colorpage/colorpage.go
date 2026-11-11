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

// colorEdited applies the hex a user typed for c, or says why not: a
// color that does not parse, or whose text would be unreadable on it
// (with the ratio), is refused and nothing changes.
func (p *Page) edited(host Host, c appearance.CustomColor) {
	text := p.Input(host, c.Token).Text
	col, err := w.ParseHex(text)
	if err != nil {
		p.setNote(c.Token, "Use #rrggbb")
		return
	}
	need := appearance.RequiredRatio(host.Settings().Contrast)
	if ratio := w.ContrastRatio(col, host.Color(c.Text)); ratio < need {
		p.setNote(c.Token, fmt.Sprintf("Contrast %.1f:1 is below %.1f:1", ratio, need))
		return
	}
	p.setNote(c.Token, "")
	host.Send(prefs.Edit{Edit: func(s *settings.Settings) {
		id := appearance.ThemeIDOf(s.Theme)
		if s.Colors == nil {
			s.Colors = map[string]map[string]string{}
		}
		if s.Colors[id] == nil {
			s.Colors[id] = map[string]string{}
		}
		s.Colors[id][string(c.Token)] = w.FormatHex(col)
	}})
}

// resetColor drops the override of one color in the current theme.
func (p *Page) reset(host Host, tok w.Token) {
	host.Send(prefs.Edit{Edit: func(s *settings.Settings) {
		delete(s.Colors[appearance.ThemeIDOf(s.Theme)], string(tok))
	}})
	delete(p.inputs, tok)
	p.setNote(tok, "")
}

// resetColors drops every color override of the current theme.
func (p *Page) resetAll(host Host) {
	host.Send(prefs.Edit{Edit: func(s *settings.Settings) { delete(s.Colors, appearance.ThemeIDOf(s.Theme)) }})
	p.Forget()
}

// colorsView is the Colors section of the settings menu: one row per
// customizable color with a swatch, its hex and a reset button.
// View is the page.
func (p *Page) View(host Host) Node {
	return Col(
		Caption("Colors", H(w.LabelH)),
		For(appearance.CustomColors, func(_ int, c appearance.CustomColor) Node { return p.row(host, c) }),
		Row(H(w.RowH), Button("Reset colors", W(110), OnClick(func() { p.resetAll(host) }))),
	)
}

func (p *Page) row(host Host, c appearance.CustomColor) Node {
	in := p.Input(host, c.Token)
	border := w.Border
	note := p.notes[c.Token]
	if note != "" {
		border = w.Error
	}
	return Col(
		Row(H(w.RowH), Gap(6),
			Text(c.Label, Small(), W(76)),
			Row(W(20), H(w.NoteH), Bg(c.Token), Control("swatch")), // shows the color in effect
			Field(in, W(70), H(w.CompactH), Bg(w.Sunken), Border(border),
				OnChange(func() { p.edited(host, c) }),
				TextMenu(in)),
			Button("Reset", W(44), H(w.CompactH), OnClick(func() { p.reset(host, c.Token) })),
		),
		If(note != "", Note(note, w.Error)),
	)
}
