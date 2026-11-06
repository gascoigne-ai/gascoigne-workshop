package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	"runtime"
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
	{"appearance", "Appearance", func(a *App) Node { return a.formPage("Appearance") }},
	{"colors", "Colors", (*App).colorsView},
	{"shortcuts", "Shortcuts", (*App).shortcutsView},
	{"requests", "Requests", func(a *App) Node { return a.formPage("Requests") }},
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
		Col(W(settingsW), H(h), Bg(w.Bg), Control("menu"), Shadow(), FloatCenter(), FocusScope(), Modal(), OnDismiss(func() { a.menu = false }),
			Row(H(settingsTitleH), PadX(16), AlignItems(AlignCenter), BorderB(w.BorderSoft),
				Text("Settings", Bold(), W(70)),
				Field(&a.settingsSearch, H(28), Grow(1), Bg(w.Sunken), Placeholder("Search settings"), OnChange(func() { a.menuScroll.Y = 0 }), a.textMenuOpt(&a.settingsSearch, nil)),
				Spacer(W(8)),
				IconButton(CrossIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(func() { a.menu = false }), Tooltip("Close (Esc)")),
			),
			Row(Grow(1),
				Col(W(settingsNavW), Pad(8), Gap(2), Bg(w.Frame),
					For(settingsCategories, func(_ int, c settingsCategory) Node { return a.navItem(c, c.id == cur) }),
				),
				Scroll(&a.menuScroll, Grow(1), Pad(16), BorderL(w.BorderSoft), page),
			),
		),
	)
}

// navItem is one category in the dialog's list: the open one is tinted and bold.
func (a *App) navItem(c settingsCategory, open bool) Node {
	items := []Item{
		H(w.RowH), PadX(12), AlignItems(AlignCenter), Control("button"),
		OnClick(func() { a.settingsCat, a.menuScroll.Y = c.id, 0 }),
	}
	if open {
		return Row(append(items, Bg(w.AccentSoft), Text(c.label, Bold(), Color(w.AccentLine), Grow(1)))...)
	}
	return Row(append(items, HoverBg(w.Hover), Text(c.label, Muted(), Grow(1)))...)
}

// viewIDOf is the response view setting in effect: unknown ids are compact.
func viewIDOf(id string) string {
	for _, v := range responseViews {
		if v.id == id {
			return id
		}
	}
	return "compact"
}

// shortcutsView is the Shortcuts section: every action with its key.
// Clicking a key records a new one; Reset appears when it differs from
// the default.
func (a *App) shortcutsView() Node {
	return Col(
		Caption("Shortcuts", H(w.LabelH)),
		For(shortcuts, func(_ int, s shortcut) Node { return a.shortcutRow(s) }),
		Caption("Text editing", H(w.LabelH)),
		For(textKeys, func(_ int, row struct{ action, keys string }) Node {
			return Row(H(w.CompactH), Gap(6),
				Text(row.action, Small(), Grow(1), Ellipsis()),
				Text(w.PlatformKeyText(runtime.GOOS, row.keys), Small(), Muted()),
			)
		}),
	)
}

func (a *App) shortcutRow(s shortcut) Node {
	_, overridden := a.settings.Shortcuts[s.id]
	label, button := a.shortcutText(s.id), Button
	if a.recording == s.id {
		label, button = "Press keys...", PrimaryButton
	}
	note, noteColor := a.recordNotes[s.id], w.Error
	if warning := bindingWarning(a.bindingOf(s)); note == "" && warning != "" {
		note, noteColor = warning, w.Muted
	}
	return Col(
		Row(H(w.CompactH), Gap(6),
			Text(s.name, Small(), Grow(1), Ellipsis()),
			button(label, W(96), H(w.CompactH), OnClick(func() { a.startRecording(s.id) })),
			If(overridden, Button("Reset", W(40), H(w.CompactH), OnClick(func() { a.resetBinding(s.id) }))),
		),
		If(note != "", Row(H(w.NoteH), Text(note, Small(), Color(noteColor), Ellipsis(), Grow(1)))),
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
	was := a.settings.ResponseView
	a.saveSettings(edit)
	if a.settings.ResponseView != was { // the choice applies to the tabs already open
		for _, t := range a.tabs {
			t.raw, t.compact = viewFlags(a.settings.ResponseView)
			a.refreshView(t)
		}
	}
	a.colorInputs, a.colorNotes = nil, nil
}

// saveSettings edits, applies and saves the settings.
func (a *App) saveSettings(edit func(*settings.Settings)) {
	edit(&a.settings)
	if err := a.ctx.SetTheme(appearance.ThemeWith(a.settings)); err != nil {
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
