package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	. "github.com/gascoigne-ai/gascoigne-workshop/ui/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// The settings dialog: a title bar, a list of categories on the left and
// the chosen category's settings on the right.
const (
	settingsW      = 760
	settingsMaxH   = 560
	settingsMargin = 40 // kept free around the dialog in a small window
	settingsNavW   = 170
	settingsTitleH = 44
)

// settingsCategory is one entry of the dialog's list.
type settingsCategory struct {
	id, label string
	view      func(a *App) Node
}

// settingsCategories are the dialog's pages, in order.
var settingsCategories = []settingsCategory{
	{"appearance", "Appearance", (*App).appearanceView},
	{"colors", "Colors", (*App).colorsView},
	{"shortcuts", "Shortcuts", (*App).shortcutsView},
}

// settingsCategoryID is the page shown: the chosen one, else the first.
func (a *App) settingsCategoryID() string {
	for _, c := range settingsCategories {
		if c.id == a.settingsCat {
			return c.id
		}
	}
	return settingsCategories[0].id
}

// menuView is the settings dialog, centered over a dimmed window while
// it is open.
func (a *App) menuView() Node {
	if !a.menu {
		return nil
	}
	h := min(settingsMaxH, a.h-2*settingsMargin)
	cur := a.settingsCategoryID()
	var page Node
	for _, c := range settingsCategories {
		if c.id == cur {
			page = c.view(a)
		}
	}
	return Frag(
		Row(Bg(w.Shadow)), // dims the window
		Col(W(settingsW), H(h), Bg(w.Bg), Control("menu"), Shadow(), FloatCenter(), FocusScope(), Modal(func() { a.menu = false }),
			Row(H(settingsTitleH), PadXY(16, 0), AlignItems(AlignCenter), BorderBottom(w.BorderSoft),
				Text("Settings", Bold(), Grow(1)),
				IconButton(Cross(Grow(1)), W(28), H(24), OnClick(func() { a.menu = false }), Tooltip("Close (Esc)")),
			),
			Row(Grow(1),
				Col(W(settingsNavW), Pad(8), Gap(2), Bg(w.Frame),
					For(settingsCategories, func(_ int, c settingsCategory) Node { return a.navItem(c, c.id == cur) }),
				),
				Scroll(&a.menuScroll, Grow(1), Pad(16), BorderLeft(w.BorderSoft), page),
			),
		),
	)
}

// navItem is one category in the dialog's list: the open one is tinted and bold.
func (a *App) navItem(c settingsCategory, open bool) Node {
	items := []Item{
		H(32), PadXY(12, 0), AlignItems(AlignCenter), Control("button"),
		OnClick(func() { a.settingsCat, a.menuScroll.Y = c.id, 0 }),
	}
	if open {
		return Row(append(items, Bg(w.AccentSoft), Text(c.label, Bold(), Color(w.AccentLine), Grow(1)))...)
	}
	return Row(append(items, HoverBg(w.Hover), Text(c.label, Muted(), Grow(1)))...)
}

// appearanceView is the first page: theme, sizes, corners and layout.
func (a *App) appearanceView() Node {
	return Col(
		Caption("Theme", H(22)),
		Row(H(w.RowH), Gap(6),
			For(allThemeChoices(), func(_ int, c themeChoice) Node {
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

		Caption("Interface scale", H(22)),
		Row(H(w.RowH), Gap(8),
			Slider(&a.scaleSlider, scaleMin, scaleMax, scaleStep, a.scalePct(),
				func(v int) { a.settings.UIScalePct, a.dirty = v, true }, // live: the UI resizes while dragging
				SliderMid(settings.DefaultUIScalePct),
				OnSlideEnd(func(int) { a.saveSettings(func(*settings.Settings) {}) }),
			),
			Text(fmt.Sprintf("%d%%", a.scaleSlider.Shown(a.scalePct())), W(40), Color(w.Muted)),
		),
		Spacer(H(w.Pad)),

		Caption("Corners", H(22)),
		Row(H(w.RowH), Gap(6),
			For(cornerChoices, func(_ int, c struct {
				id, label string
				scale     float64
			}) Node {
				return a.choice(c.label, c.id == cornerIDOf(a.settings.Corners), func(s *settings.Settings) { s.Corners = c.id })
			}),
		),
		Spacer(H(w.Pad)),

		Caption("Gradients", H(22)),
		Row(H(w.RowH), Gap(6),
			a.choice("On", !a.settings.Flat, func(s *settings.Settings) { s.Flat = false }),
			a.choice("Off", a.settings.Flat, func(s *settings.Settings) { s.Flat = true }),
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
	)
}

// shortcutsView is the Shortcuts section: every action with its key.
// Clicking a key records a new one; Reset appears when it differs from
// the default.
func (a *App) shortcutsView() Node {
	return Col(
		Caption("Shortcuts", H(22)),
		For(shortcuts, func(_ int, s shortcut) Node { return a.shortcutRow(s) }),
	)
}

func (a *App) shortcutRow(s shortcut) Node {
	_, overridden := a.settings.Shortcuts[s.id]
	label, button := a.shortcutText(s.id), Button
	if a.recording == s.id {
		label, button = "Press keys...", PrimaryButton
	}
	note := a.recordNotes[s.id]
	return Col(
		Row(H(24), Gap(6),
			Text(s.name, Small(), Grow(1), Ellipsis()),
			button(label, W(96), OnClick(func() { a.startRecording(s.id) })),
			If(overridden, Button("Reset", W(40), OnClick(func() { a.resetBinding(s.id) }))),
		),
		If(note != "", Row(H(18), Text(note, Small(), Color(w.Error), Ellipsis(), Grow(1)))),
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

// scalePct is the interface scale in percent, as the slider shows it.
func (a *App) scalePct() int { return int(a.settings.UIScale()*100 + 0.5) }
