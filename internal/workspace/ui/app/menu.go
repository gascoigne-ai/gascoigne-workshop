package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"runtime"
	"strings"
)

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
	a.colors.Forget()
}

// saveSettings edits and applies the settings and, unless auto-save is off,
// writes them. Turning auto-save off writes that choice once; turning it on
// writes whatever was waiting.
func (a *App) saveSettings(edit func(*settings.Settings)) {
	wasAuto := a.settings.AutoSave
	edit(&a.settings)
	if err := a.ctx.SetTheme(appearance.ThemeWith(a.settings)); err != nil {
		a.status = err.Error()
		return
	}
	if a.settingsPath == "" {
		return
	}
	if !a.settings.AutoSave && wasAuto == false {
		a.settingsUnsaved = true // applied now, written when the user saves
		return
	}
	// auto-save is on, was just turned on (which writes what was pending) or
	// was just turned off (which writes that choice, and nothing else changed)
	a.Persist()
}
