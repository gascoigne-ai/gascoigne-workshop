package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/settingsui"
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
func (a *App) Close() { a.closeSettings() }

// closeSettings closes the dialog and what is open inside it, so the next
// opening starts clean.
func (a *App) closeSettings() {
	if a.menu && a.settingsUnsaved {
		a.askSaveSettings()
		return
	}
	a.closeSettingsNow()
}

// closeSettingsNow closes the dialog without asking.
func (a *App) closeSettingsNow() {
	a.cancelSettingsFilePick()
	a.menu = false
	a.prefs.Fresh()
}

// MarkDirty asks for a redraw.
func (a *App) MarkDirty() { a.dirty = true }

// PickFile asks for a file and calls done with its path.
func (a *App) PickFile(title string, done func(path string)) {
	a.cancelSettingsFilePick()
	a.cancelSettingsPick = a.pickFile(title, false, "", done)
}

// cancelSettingsFilePick takes back the file question the settings dialog
// asked, if one is open; a question asked by anything else is left alone.
func (a *App) cancelSettingsFilePick() {
	if a.cancelSettingsPick != nil {
		a.cancelSettingsPick()
		a.cancelSettingsPick = nil
	}
}

// Pointer is the pointer position in the window.
func (a *App) Pointer() (int, int) { return a.ctx.In.X, a.ctx.In.Y }

// Later runs f on the UI goroutine.
func (a *App) Later(f func()) { a.bus.Dispatch(f) }

// OpenMenu opens a menu of items at a point of the window.
func (a *App) OpenMenu(x, y int, items []settingsui.MenuItem) {
	rows := make([]menuItem, len(items))
	for i, item := range items {
		rows[i] = menuItem{label: item.Label, hint: item.Hint, run: item.Run}
	}
	a.openMenu(x, y, rows)
}

// ExtraPage draws the settings pages that are tables of the application.
func (a *App) ExtraPage(id string) Node {
	switch id {
	case "colors":
		return a.colors.View(pageHost{a})
	case "shortcuts":
		return a.shortcutKeys.View(pageHost{a})
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
	for _, t := range a.openTabs() {
		t.look.Mode = viewer.ModeOf(s.ResponseView)
		a.refreshView(t)
	}
	a.colors.Forget()
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

// toggleSettings is the header button: it opens the dialog or closes it.
func (a *App) toggleSettings() {
	if a.menu {
		a.closeSettings()
		return
	}
	a.menu = true
}

// Unsaved reports whether settings were changed but not written (auto-save
// is off).
func (a *App) Unsaved() bool { return a.settingsUnsaved }

// MarkUnsaved and MarkSaved record what the dialog did with the file itself.
func (a *App) MarkUnsaved() { a.settingsUnsaved = true }

// MarkSaved says the file now holds the settings in effect.
func (a *App) MarkSaved() { a.settingsUnsaved = false }

// Persist writes the settings in effect to the file.
func (a *App) Persist() {
	if a.settingsPath == "" {
		a.settingsUnsaved = false
		return
	}
	if err := a.settingFiles.Save(a.settingsPath, a.settings); err != nil {
		a.status = "Could not save settings: " + err.Error()
		a.toast(a.status, true)
		return
	}
	a.prefs.Stamp(a, a.settingsPath)
	a.settingsUnsaved = false
}

// revertSettings puts back what the file holds, dropping unsaved changes.
func (a *App) revertSettings() {
	a.settingsUnsaved = false
	if a.settingsPath == "" {
		return
	}
	loaded, err := a.settingFiles.Load(a.settingsPath)
	if err != nil {
		a.toast("Could not read settings.json: "+err.Error(), true)
		return
	}
	a.applySettings(loaded)
}

// askSaveSettings asks what to do with unsaved changes when the dialog
// is closed.
func (a *App) askSaveSettings() {
	a.dialog = &dialog{
		title: "Unsaved settings",
		body:  "The changes are applied but not written to settings.json.\nSave them, or discard them and go back to what is stored?",
		buttons: []dialogButton{
			{label: "Cancel", run: a.closeDialog},
			{label: "Discard", run: func() { a.closeDialog(); a.revertSettings(); a.closeSettingsNow() }},
			{label: "Save", primary: true, run: func() { a.closeDialog(); a.Persist(); a.closeSettingsNow() }},
		},
	}
}

// SettingsStore is the settings file's store.
func (a *App) SettingsStore() port.SettingsStore { return a.settingFiles }

// FontStore is the installed fonts of this machine.
func (a *App) FontStore() port.FontStore { return a.fonts }
