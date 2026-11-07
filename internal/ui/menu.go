package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	"runtime"
	"strings"
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

// shortcutsView is the Shortcuts page: a search field and a table of
// actions with their keys. A key button records the next combination;
// Clear unbinds and Reset gives the default back. A key another action has
// is offered with a Replace button, and a key that types text is a warning.
func (a *App) shortcutsView() Node {
	var rows []Item
	for _, s := range shortcuts {
		if shortcutMatches(s, a.shortcutText(s.id), a.shortcutSearch.Text) {
			rows = append(rows, a.shortcutRow(s))
		}
	}
	if len(rows) == 0 {
		rows = append(rows, Text("No shortcut matches", Muted()))
	}
	return Col(
		Row(H(w.RowH), Gap(6),
			Field(&a.shortcutSearch, Grow(1), Bg(w.Sunken), Placeholder("Search by action or key"), a.textMenuOpt(&a.shortcutSearch, nil)),
		),
		Spacer(H(w.Gutter)),
		Col(rows...),
		Caption("Text editing", H(w.LabelH)),
		For(textKeys, func(_ int, row struct{ action, keys string }) Node {
			return Row(H(w.CompactH), Gap(6),
				Text(row.action, Small(), Grow(1), Ellipsis()),
				Text(w.PlatformKeyText(runtime.GOOS, row.keys), Small(), Muted()),
			)
		}),
	)
}

// shortcutMatches reports whether an action's name or its key text contains
// the search text, ignoring case.
func shortcutMatches(s shortcut, keyText, query string) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	return needle == "" || strings.Contains(strings.ToLower(s.name), needle) || strings.Contains(strings.ToLower(keyText), needle)
}

func (a *App) shortcutRow(s shortcut) Node {
	_, overridden := a.settings.Shortcuts[s.id]
	bound := a.bindingOf(s).Bound
	label, button := a.shortcutText(s.id), Button
	if a.recording == s.id {
		label, button = "Press keys...", PrimaryButton
	}
	var note Node
	if pending, ok := a.pendingKeys[s.id]; ok {
		note = Row(H(w.CompactH), Gap(6),
			Text(fmt.Sprintf("Used by %s (%s)", pending.otherName, w.PlatformKeyText(runtime.GOOS, pending.binding.String())), Small(), Color(w.Error), Ellipsis(), Grow(1)),
			Button("Replace", W(64), H(w.CompactH), OnClick(func() { a.replaceBinding(s.id) }), Tooltip("Unbind "+pending.otherName+" and use this key here")),
		)
	} else if text := a.recordNotes[s.id]; text != "" {
		note = Row(H(w.NoteH), Text(text, Small(), Color(w.Error), Ellipsis(), Grow(1)))
	} else if warning := bindingWarning(a.bindingOf(s)); warning != "" {
		note = Row(H(w.NoteH), Text(warning, Small(), Color(w.Muted), Ellipsis(), Grow(1)))
	}
	return Col(
		Row(H(w.CompactH), Gap(6),
			Text(s.name, Small(), Grow(1), Ellipsis()),
			button(label, W(96), H(w.CompactH), OnClick(func() { a.startRecording(s.id) }), Tooltip("Click, then press the keys")),
			If(bound, Button("Clear", W(44), H(w.CompactH), OnClick(func() { a.clearBinding(s.id) }), Tooltip("Leave the action without a key"))),
			If(overridden, Button("Reset", W(44), H(w.CompactH), OnClick(func() { a.resetBinding(s.id) }), Tooltip("Back to the default key"))),
		),
		note,
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
