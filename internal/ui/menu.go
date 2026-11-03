package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/vml"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

const menuW = 260

// menuView is the settings menu, floating under the header's three-dot
// button. It swallows clicks while open; a click outside closes it.
func (a *App) menuView() vml.Node {
	if !a.menu {
		return nil
	}
	return vml.Col(
		vml.Row(vml.Caption("Settings")).H(widget.RowH).BorderBottom(widget.Rule),
		vml.Spacer().H(6),
		vml.Caption("Theme").H(22),
		vml.Row(vml.For(themeChoices, func(_ int, c themeChoice) vml.Node {
			return a.choice(c.label, c.id == a.settings.Theme, func(s *settings.Settings) { s.Theme = c.id })
		})).Gap(6).H(widget.RowH),
		vml.Spacer().H(widget.Pad),
		vml.Caption("Text size").H(22),
		vml.Row(vml.For(textSizeChoices, func(_ int, c sizeChoice) vml.Node {
			return a.choice(c.label, c.id == a.settings.TextSize, func(s *settings.Settings) { s.TextSize = c.id })
		})).Gap(6).H(widget.RowH),
	).W(menuW).Pad(widget.Pad).Gap(0).
		Bg(widget.Bg).Border(widget.Border).Shadow().
		Float(headerH, widget.Pad).
		Modal(func() { a.menu = false })
}

// choice is one chip of a segmented choice: gold when selected.
func (a *App) choice(label string, selected bool, edit func(*settings.Settings)) vml.Node {
	btn := vml.Button(label)
	if selected {
		btn = vml.PrimaryButton(label)
	}
	return btn.Grow(1).OnClick(func() { a.changeSettings(edit) })
}

// changeSettings edits the settings, applies them and saves them.
func (a *App) changeSettings(edit func(*settings.Settings)) {
	edit(&a.settings)
	if err := a.ctx.SetTheme(ThemeFor(a.settings.Theme, a.settings.TextSize)); err != nil {
		a.status = err.Error()
		return
	}
	if a.settingsPath == "" {
		return
	}
	if err := a.settings.Save(a.settingsPath); err != nil {
		a.status = "Could not save settings: " + err.Error()
	}
}
