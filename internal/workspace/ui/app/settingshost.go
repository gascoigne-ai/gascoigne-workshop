package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"image/color"
)

// The methods below make the App the host of the settings dialog
// (settingsui.Host). Size, Toast and SetStatus are with the other hosts.

// Settings are the settings in effect.
func (a *App) Settings() settings.Settings { return a.settings }

// SettingsPath is the settings file, "" when settings are not kept.
func (a *App) SettingsPath() string { return a.settingsPath }

// SetFonts switches the interface and code fonts.
func (a *App) SetFonts(ui, code w.FontData) error { return a.ctx.SetFonts(ui, code) }

// ActionIDs are the ids of the shortcut actions.
func (a *App) ActionIDs() []string { return a.actionIDs }

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

// pickSettingsFile asks for a file and calls done with its path.
func (a *App) pickSettingsFile(title string, done func(path string)) {
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

// openMenuItems opens a menu of a page's items at a point of the window.
func (a *App) openMenuItems(x, y int, items []prefs.MenuItem) {
	rows := make([]MenuItem, len(items))
	for i, item := range items {
		rows[i] = MenuItem{Label: item.Label, Hint: item.Hint, Run: item.Run}
	}
	a.openMenu(x, y, rows)
}

// ExtraPage draws the settings pages that are tables of the application.
func (a *App) ExtraPage(id string) Node {
	switch id {
	case "colors":
		return a.colors.View(a)
	case "shortcuts":
		return a.shortcutKeys.View(a)
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

// persistSettings writes the settings in effect to the file.
func (a *App) persistSettings() {
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
	a.dialog = &dialog{DialogState: DialogState{
		Title: "Unsaved settings",
		Body:  "The changes are applied but not written to settings.json.\nSave them, or discard them and go back to what is stored?",
		Buttons: []DialogButton{
			{Label: "Cancel", Run: a.closeDialog},
			{Label: "Discard", Run: func() { a.closeDialog(); a.revertSettings(); a.closeSettingsNow() }},
			{Label: "Save", Primary: true, Run: func() { a.closeDialog(); a.persistSettings(); a.closeSettingsNow() }},
		},
	}}
}

// SettingsStore is the settings file's store.
func (a *App) SettingsStore() port.SettingsStore { return a.settingFiles }

// FontStore is the installed fonts of this machine.
func (a *App) FontStore() port.FontStore { return a.fonts }

// Send applies a message of a settings page. The page's host is the App:
// its other methods only read.
func (a *App) Send(msg prefs.Msg) {
	switch msg := msg.(type) {
	case prefs.Change:
		a.changeSettings(msg.Edit)
	case prefs.Edit:
		a.saveSettings(msg.Edit)
	case prefs.ScaleLive:
		a.settings.UIScalePct, a.dirty = msg.Percent, true
	case prefs.Save:
		a.saveSettings(func(*settings.Settings) {})
	case prefs.Replace:
		a.applySettings(msg.Settings)
	case prefs.Persist:
		a.persistSettings()
	case prefs.MarkUnsaved:
		a.settingsUnsaved = true
	case prefs.MarkSaved:
		a.settingsUnsaved = false
	case prefs.Toast:
		a.toast(msg.Text, msg.IsErr)
	case prefs.SetStatus:
		a.status = msg.Text
	case prefs.OpenMenu:
		a.openMenuItems(msg.X, msg.Y, msg.Items)
	case prefs.PickFile:
		a.pickSettingsFile(msg.Title, msg.Done)
	case prefs.Close:
		a.closeSettings()
	case prefs.MarkDirty:
		a.dirty = true
	case prefs.Later:
		a.bus.Dispatch(msg.Func)
	}
}

// Color is the color in effect for a token.
func (a *App) Color(tok w.Token) color.RGBA { return a.ctx.Th.Color(tok) }

// Actions are the shortcut actions that have keys, in display order.
func (a *App) Actions() []shortcutpage.Action {
	actions := make([]shortcutpage.Action, len(shortcuts))
	for i, s := range shortcuts {
		actions[i] = s.action()
	}
	return actions
}
