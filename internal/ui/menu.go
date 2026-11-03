package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	. "github.com/gascoigne-ai/gascoigne-workshop/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/widget"
)

const menuW = 260

// menuView is the settings menu, floating under the header's three-dot
// button. It swallows clicks while open; a click outside closes it.
func (a *App) menuView() Node {
	if !a.menu {
		return nil
	}
	closeMenu := Modal(func() { a.menu = false })
	return Col(W(menuW), Pad(w.Pad), Bg(w.Bg), Border(w.Border), Shadow(), Float(headerH, w.Pad), closeMenu,
		Row(H(w.RowH), BorderBottom(w.Rule),
			Caption("Settings"),
		),
		Spacer(H(6)),

		Caption("Theme", H(22)),
		Row(H(w.RowH), Gap(6),
			For(themeChoices, func(_ int, c themeChoice) Node {
				return a.choice(c.label, c.id == a.settings.Theme, func(s *settings.Settings) { s.Theme = c.id })
			}),
		),
		Spacer(H(w.Pad)),

		Caption("Text size", H(22)),
		Row(H(w.RowH), Gap(6),
			For(textSizeChoices, func(_ int, c sizeChoice) Node {
				return a.choice(c.label, c.id == a.settings.TextSize, func(s *settings.Settings) { s.TextSize = c.id })
			}),
		),
		Spacer(H(w.Pad)),

		Caption("Request / response", H(22)),
		Row(H(w.RowH), Gap(6),
			For(layoutChoices, func(_ int, c layoutChoice) Node {
				return a.choice(c.label, c.id == a.settings.Layout, func(s *settings.Settings) { s.Layout = c.id })
			}),
		),
		Spacer(H(w.Pad)),

		Caption("Contrast", H(22)),
		Row(H(w.RowH), Gap(6),
			For(contrastChoices, func(_ int, c contrastChoice) Node {
				return a.choice(c.label, c.id == a.settings.Contrast, func(s *settings.Settings) { s.Contrast = c.id })
			}),
		),
		Spacer(H(w.Pad)),

		a.colorsView(),
		Spacer(H(w.Pad)),

		Caption("Shortcuts", H(22)),
		For(shortcuts, func(_ int, s shortcut) Node {
			return Row(H(20),
				Text(s.name, Small()),
				Spacer(),
				Text(s.label(), Small(), Muted()),
			)
		}),
	)
}

// choice is one chip of a segmented choice: gold when selected.
func (a *App) choice(label string, selected bool, edit func(*settings.Settings)) Node {
	apply := OnClick(func() { a.changeSettings(edit) })
	if selected {
		return PrimaryButton(label, Grow(1), apply)
	}
	return Button(label, Grow(1), apply)
}

// changeSettings edits the settings, applies them and saves them. The
// color fields are rebuilt, since the colors in effect may have changed.
func (a *App) changeSettings(edit func(*settings.Settings)) {
	a.saveSettings(edit)
	a.colorInputs, a.colorNotes = nil, nil
}

// saveSettings edits, applies and saves the settings.
func (a *App) saveSettings(edit func(*settings.Settings)) {
	edit(&a.settings)
	if err := a.ctx.SetTheme(ThemeWith(a.settings)); err != nil {
		a.status = err.Error()
		return
	}
	if a.settingsPath == "" {
		return
	}
	if err := a.settings.Save(a.settingsPath); err != nil {
		a.status = "Could not save settings: " + err.Error()
		a.toast(a.status, true)
	}
}
