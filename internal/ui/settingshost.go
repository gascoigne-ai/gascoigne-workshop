package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/settingsui"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The methods below make the App the host of the settings dialog
// (settingsui.Host). Size, Toast and SetStatus are with the other hosts.

// Settings are the settings in effect.
func (a *App) Settings() settings.Settings { return a.settings }

// Change edits the settings, applies them and saves the file.
func (a *App) Change(edit func(*settings.Settings)) { a.changeSettings(edit) }

// SetScaleLive applies an interface scale while its slider is dragged.
func (a *App) SetScaleLive(percent int) { a.settings.UIScalePct, a.dirty = percent, true }

// Save writes the settings file.
func (a *App) Save() { a.saveSettings(func(*settings.Settings) {}) }

// Replace makes s the settings in effect, with everything that follows
// from them, without writing the file.
func (a *App) Replace(s settings.Settings) { a.applySettings(s) }

// SettingsPath is the settings file, "" when settings are not kept.
func (a *App) SettingsPath() string { return a.settingsPath }

// SetFonts switches the interface and code fonts.
func (a *App) SetFonts(ui, code w.FontData) error { return a.ctx.SetFonts(ui, code) }

// ActionIDs are the ids of the shortcut actions.
func (a *App) ActionIDs() []string { return a.actionIDs }

// Close closes the settings dialog.
func (a *App) Close() { a.menu = false }

// MarkDirty asks for a redraw.
func (a *App) MarkDirty() { a.dirty = true }

// Pointer is the pointer position in the window.
func (a *App) Pointer() (int, int) { return a.ctx.In.X, a.ctx.In.Y }

// Later runs f on the UI goroutine.
func (a *App) Later(f func()) { a.later <- f }

// OpenMenu opens a menu of items at a point of the window.
func (a *App) OpenMenu(x, y int, items []settingsui.MenuItem) {
	rows := make([]menuItem, len(items))
	for i, item := range items {
		rows[i] = menuItem{label: item.Label, run: item.Run}
	}
	a.openMenu(x, y, rows)
}

// ExtraPage draws the settings pages that are tables of the application.
func (a *App) ExtraPage(id string) Node {
	switch id {
	case "colors":
		return a.colorsView()
	case "shortcuts":
		return a.shortcutsView()
	}
	return nil
}

// applySettings makes s the settings in effect, with everything that
// follows from them.
func (a *App) applySettings(s settings.Settings) {
	a.settings = s
	if err := a.ctx.SetTheme(appearance.ThemeWith(s)); err != nil {
		a.status = err.Error()
	}
	for _, t := range a.tabs {
		t.raw, t.compact = viewFlags(s.ResponseView)
		a.refreshView(t)
	}
	a.colorInputs, a.colorNotes = nil, nil
	a.prefs.Reload(a)
	a.dirty = true
}

// menuView is the settings dialog while it is open.
func (a *App) menuView() Node {
	if !a.menu {
		return nil
	}
	return a.prefs.View(a)
}
