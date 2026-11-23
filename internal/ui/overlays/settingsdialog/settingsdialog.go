package settingsdialog

import (
	"errors"
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/colorpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/systempage"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/systeminfo"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the settings dialog with its pages: whether it is open, what
// was applied but not written to settings.json, and the file question it
// asked. It is the Host of the shell and of every page, so the application
// answers only the narrow Deps. The pages' state is exported for the tests
// of the application, which drive them as a user does.
//
// The settings themselves stay with the application (the theme, the open
// tabs and the protocols follow from them); the component owns what is
// about the file: when to write it, when to ask, and the refusal to write a
// file a newer build wrote.
type Component struct {
	deps Deps

	open       bool   // the dialog is shown
	unsaved    bool   // settings were changed but not written (auto-save is off)
	cancelPick func() // takes back a file question the dialog asked
	ctx        *w.Ctx // the context of the last View: the shell asks for pages without one

	Shell     SettingsDialog    // the dialog around the pages
	Form      formpage.Page     // the form of the settings of a category
	Colors    colorpage.Page    // the Colors page
	Shortcuts shortcutpage.Page // the Shortcuts page
}

// New is the component; the dialog starts closed.
func New(deps Deps) *Component {
	c := &Component{deps: deps}
	c.Shell.Stamp(c, deps.Path)
	return c
}

// Open reports whether the dialog is shown.
func (c *Component) Open() bool { return c.open }

// Show opens the dialog.
func (c *Component) Show() { c.open = true }

// Toggle is the header button: it opens the dialog or closes it.
func (c *Component) Toggle() {
	if c.open {
		c.Close()
		return
	}
	c.open = true
}

// Close closes the dialog and what is open inside it, so the next opening
// starts clean. It asks first when changes were applied but not written.
func (c *Component) Close() {
	if c.open && c.unsaved {
		c.askSave()
		return
	}
	c.closeNow()
}

// closeNow closes the dialog without asking.
func (c *Component) closeNow() {
	c.cancelFilePick()
	c.open = false
	c.Shell.Fresh()
	c.Form.Fresh()
}

// View is the dialog while it is open.
func (c *Component) View(ctx *w.Ctx) Node {
	if !c.open {
		return nil
	}
	c.ctx = ctx
	return Frag(c.Shell.View(ctx, c), c.Form.Fonts.View(ctx, c)) // the font list floats over the dialog
}

// FontListOpen reports whether the font list floats over the dialog.
func (c *Component) FontListOpen() bool { return c.Form.Fonts.IsOpen() }

// CloseFontList closes the font list.
func (c *Component) CloseFontList() { c.Form.Fonts.Close() }

// ScanFonts looks for installed fonts in the background.
func (c *Component) ScanFonts() { c.Form.Fonts.Scan(c) }

// Poll picks up an outside change of settings.json.
func (c *Component) Poll() { c.Shell.Poll(c) }

// OpenRaw shows the text of settings.json.
func (c *Component) OpenRaw() { c.Shell.OpenRaw(c) }

// RawFocused reports whether the text of settings.json has the keyboard.
func (c *Component) RawFocused() bool { return c.Shell.RawFocused() }

// CompleteRaw offers what fits at the caret of the text of settings.json.
func (c *Component) CompleteRaw() { c.Shell.CompleteRaw(c) }

// Send applies a message of a settings page.
func (c *Component) Send(msg prefs.Msg) {
	c.deps.Window.Stale() // a message changes settings or a page's state, which the built tree shows
	switch msg := msg.(type) {
	case prefs.Change:
		c.Change(msg.Edit)
	case prefs.Edit:
		c.Edit(msg.Edit)
	case prefs.ScaleLive:
		c.deps.Settings.SetScale(msg.Percent)
	case prefs.Save:
		c.Edit(func(*settings.Settings) {})
	case prefs.Replace:
		c.replace(msg.Settings)
	case prefs.Persist:
		c.persist()
	case prefs.FileMarked:
		c.unsaved = !msg.Saved
	case prefs.Toast:
		c.deps.Window.Toast(msg.Text, msg.Level)
	case prefs.SetStatus:
		c.deps.Window.SetStatus(msg.Text)
	case prefs.OpenMenu:
		c.openMenu(msg)
	case prefs.PickFile:
		c.pick(msg.Title, msg.Done)
	case prefs.Close:
		c.Close()
	case prefs.MarkDirty:
		c.deps.Window.Redraw()
	case prefs.Later:
		c.deps.Bus.Dispatch(msg.Func)
	}
}

// Change is Edit for the dialog's own controls: the color fields are rebuilt
// afterwards, since the colors in effect may have changed.
func (c *Component) Change(edit func(*settings.Settings)) {
	c.Edit(edit)
	c.Colors.Forget() // not in Edit: the Colors page edits through it while a field is being typed
}

// Edit changes the settings and, unless auto-save is off, writes them.
// Turning auto-save off writes that choice once; turning it on writes
// whatever was waiting.
func (c *Component) Edit(edit func(*settings.Settings)) {
	wasAuto := c.deps.Settings.Settings().AutoSave
	if err := c.deps.Settings.Edit(edit); err != nil {
		c.deps.Window.SetStatus(err.Error())
		return
	}
	if c.deps.Path == "" {
		return
	}
	if !c.deps.Settings.Settings().AutoSave && !wasAuto {
		c.unsaved = true // applied now, written when the user saves
		return
	}
	// auto-save is on, was just turned on (which writes what was pending) or
	// was just turned off (which writes that choice, and nothing else changed)
	c.persist()
}

// replace makes current the settings in effect, as when the file changed
// outside, and rebuilds what the pages show of them.
func (c *Component) replace(current settings.Settings) {
	c.deps.Settings.Apply(current)
	c.Colors.Forget()
	c.Form.Reload(c)
}

// persist writes the settings in effect to the file.
func (c *Component) persist() {
	if c.deps.Path == "" {
		c.unsaved = false
		return
	}
	if err := c.SettingsStore().Save(c.deps.Path, c.deps.Settings.Settings()); err != nil {
		message := "Could not save settings: " + err.Error()
		c.deps.Window.SetStatus(message)
		c.deps.Window.Toast(message, severity.Error)
		return
	}
	c.Shell.Stamp(c, c.deps.Path)
	c.unsaved = false
}

// revert puts back what the file holds, dropping unsaved changes.
func (c *Component) revert() {
	c.unsaved = false
	if c.deps.Path == "" {
		return
	}
	loaded, err := c.deps.Store.Load(c.deps.Path)
	var clamped *settings.ClampedError
	switch {
	case errors.As(err, &clamped): // the values are usable: applied, with a note
		c.deps.Window.Toast(clamped.Error(), severity.Error)
	case err != nil:
		c.deps.Window.Toast("Could not read settings.json: "+err.Error(), severity.Error)
		return
	}
	c.replace(loaded)
}

// askSave asks what to do with unsaved changes when the dialog is closed.
func (c *Component) askSave() {
	var closeAsk func()
	closeAsk = c.deps.Dialogs.Open(widget.DialogState{
		Title: "Unsaved settings",
		Body:  "The changes are applied but not written to settings.json.\nSave them, or discard them and go back to what is stored?",
		Buttons: []widget.DialogButton{
			{Label: "Cancel", Run: func() { closeAsk() }},
			{Label: "Discard", Run: func() { closeAsk(); c.revert(); c.closeNow() }},
			{Label: "Save", Primary: true, Run: func() { closeAsk(); c.persist(); c.closeNow() }},
		},
	})
}

// pick asks for a file and calls done with its path.
func (c *Component) pick(title string, done func(path string)) {
	c.cancelFilePick()
	c.cancelPick = c.deps.Files.Pick(title, done)
}

// cancelFilePick takes back the file question the dialog asked, so a late
// answer does not reach a dialog that was closed.
func (c *Component) cancelFilePick() {
	if c.cancelPick != nil {
		c.cancelPick()
		c.cancelPick = nil
	}
}

// openMenu opens a menu of a page's items at a point of the window.
func (c *Component) openMenu(msg prefs.OpenMenu) {
	rows := make([]widget.MenuItem, len(msg.Items))
	for i, item := range msg.Items {
		rows[i] = widget.MenuItem{Label: item.Label, Hint: item.Hint, Run: item.Run}
	}
	c.deps.Window.OpenMenu(msg.X, msg.Y, rows)
}

// Size is the window's size in logical pixels.
func (c *Component) Size() (width, height int) { return c.deps.Window.Size() }

// Pointer is the pointer position in the window.
func (c *Component) Pointer() (int, int) { return c.deps.Window.Pointer() }

// Settings are the settings in effect.
func (c *Component) Settings() settings.Settings { return c.deps.Settings.Settings() }

// SettingsPath is the settings file, "" when settings are not kept.
func (c *Component) SettingsPath() string { return c.deps.Path }

// SettingsStore is the settings file's store; it refuses to write a file a
// newer build wrote, for the whole session.
func (c *Component) SettingsStore() settings.SettingsStore {
	if c.deps.Access == storage.ReadOnly {
		return settings.ReadOnly(c.deps.Store)
	}
	return c.deps.Store
}

// Unsaved reports whether settings were changed but not written (auto-save
// is off).
func (c *Component) Unsaved() bool { return c.unsaved }

// FormPage draws the form of the settings of a category, or of every
// category that matches the search text.
func (c *Component) FormPage(category, query string) Node {
	return c.Form.View(c.ctx, c, category, query)
}

// ExtraPage draws the settings pages that are tables of the application.
func (c *Component) ExtraPage(id string) Node {
	switch id {
	case "colors":
		return c.Colors.View(c.ctx, c)
	case "shortcuts":
		return c.Shortcuts.View(c.ctx, c)
	case "system":
		return systempage.View(c.ctx, c)
	}
	return nil
}

// OpenPage shows a page of the dialog, as a click on a search result that
// leads to it does.
func (c *Component) OpenPage(id string) { c.Shell.OpenPage(id) }

// ActionIDs are the ids of the shortcut actions, in display order.
func (c *Component) ActionIDs() []string {
	var ids []string
	for _, command := range c.Keymap().Commands() {
		ids = append(ids, string(command.ID))
	}
	return ids
}

// Keymap is the binding of every shortcut action, in display order.
func (c *Component) Keymap() keymap.Keymap { return c.deps.Keys.Keymap() }

// FontStore is the installed fonts of this machine.
func (c *Component) FontStore() fontlist.FontStore { return c.deps.Fonts }

// SetFonts switches the interface and code fonts.
func (c *Component) SetFonts(ui, code w.FontData) error { return c.deps.Looks.SetFonts(ui, code) }

// Color is the color in effect for a token.
func (c *Component) Color(token w.Token) color.RGBA { return c.deps.Looks.Color(token) }

// HostOS is the operating system shortcut text follows.
func (c *Component) HostOS() string { return c.deps.Looks.HostOS() }

// CopyText puts text on the clipboard.
func (c *Component) CopyText(text string) { c.deps.Looks.CopyText(text) }

// System is the report the System page shows: empty without a source.
func (c *Component) System() systeminfo.Report {
	if c.deps.System == nil {
		return systeminfo.Report{}
	}
	return c.deps.System.Read()
}
