package ui

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	. "github.com/gascoigne-ai/gascoigne-workshop/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// colorInput is the hex field of a custom color, created on first use
// from the color in effect.
func (a *App) colorInput(tok w.Token) *w.TextField {
	if in, ok := a.colorInputs[tok]; ok {
		return in
	}
	if a.colorInputs == nil {
		a.colorInputs = map[w.Token]*w.TextField{}
	}
	in := new(w.TextField)
	*in = w.NewTextField(w.FormatHex(a.ctx.Th.Color(tok)))
	a.colorInputs[tok] = in
	return in
}

func (a *App) setColorNote(tok w.Token, note string) {
	if a.colorNotes == nil {
		a.colorNotes = map[w.Token]string{}
	}
	if note == "" {
		delete(a.colorNotes, tok)
		return
	}
	a.colorNotes[tok] = note
}

// colorEdited applies the hex a user typed for c, or says why not: a
// color that does not parse, or whose text would be unreadable on it
// (with the ratio), is refused and nothing changes.
func (a *App) colorEdited(c customColor) {
	text := a.colorInput(c.token).Text
	col, err := w.ParseHex(text)
	if err != nil {
		a.setColorNote(c.token, "Use #rrggbb")
		return
	}
	need := requiredRatio(a.settings.Contrast)
	if ratio := w.ContrastRatio(col, a.ctx.Th.Color(c.text)); ratio < need {
		a.setColorNote(c.token, fmt.Sprintf("Contrast %.1f:1 is below %.1f:1", ratio, need))
		return
	}
	a.setColorNote(c.token, "")
	a.saveSettings(func(s *settings.Settings) {
		id := themeIDOf(s.Theme)
		if s.Colors == nil {
			s.Colors = map[string]map[string]string{}
		}
		if s.Colors[id] == nil {
			s.Colors[id] = map[string]string{}
		}
		s.Colors[id][string(c.token)] = w.FormatHex(col)
	})
}

// resetColor drops the override of one color in the current theme.
func (a *App) resetColor(tok w.Token) {
	a.saveSettings(func(s *settings.Settings) {
		delete(s.Colors[themeIDOf(s.Theme)], string(tok))
	})
	delete(a.colorInputs, tok)
	a.setColorNote(tok, "")
}

// resetColors drops every color override of the current theme.
func (a *App) resetColors() {
	a.saveSettings(func(s *settings.Settings) { delete(s.Colors, themeIDOf(s.Theme)) })
	a.colorInputs, a.colorNotes = nil, nil
}

// colorsView is the Colors section of the settings menu: one row per
// customizable color with a swatch, its hex and a reset button.
func (a *App) colorsView() Node {
	return Col(
		Caption("Colors", H(22)),
		For(customColors, func(_ int, c customColor) Node { return a.colorRow(c) }),
		Row(H(w.RowH), Button("Reset colors", W(110), OnClick(a.resetColors))),
	)
}

func (a *App) colorRow(c customColor) Node {
	in := a.colorInput(c.token)
	border := w.Border
	note := a.colorNotes[c.token]
	if note != "" {
		border = w.Error
	}
	return Col(
		Row(H(w.RowH), Gap(6),
			Text(c.label, Small(), W(76)),
			Row(W(20), H(18), Bg(c.token), Border(w.Border)), // shows the color in effect
			Field(in, W(70), H(24), Bg(w.Sunken), Border(border),
				OnChange(func() { a.colorEdited(c) }),
				a.textMenuOpt(in, func() { a.colorEdited(c) })),
			Button("Reset", W(44), OnClick(func() { a.resetColor(c.token) })),
		),
		If(note != "", Row(H(18), Text(note, Small(), Color(w.Error), Ellipsis(), Grow(1)))),
	)
}
