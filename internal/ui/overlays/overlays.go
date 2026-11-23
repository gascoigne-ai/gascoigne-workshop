// Package overlays is the composite of everything that floats over the
// window: the one dialog, the context menu, the destination picker, the file
// browser, the call trace, the settings dialog, the welcome notes and the
// toasts. It owns their open state and answers what its children ask of each
// other (a dialog opened for the browser, a toast for the settings), so
// siblings never import each other: the window wires only this package.
//
// The dialog is one slot, not a stack: a second dialog replaces the first,
// and the closer Open returns clears only the dialog it opened. A stack
// would let Escape and Enter reach a dialog under another one, which no
// caller asks for; revisit when a flow needs a question over a question
// (task 044 changes the key handling and is the place to look).
//
// The window's template keeps one slot per overlay and Views returns one
// node per slot, not a single fragment: the z-order and the names the
// window dumps pin are the template's, and a fragment would change the
// structure of every dump.
package overlays

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/overlay"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/calltrace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/menus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/toasts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/welcome"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// dialog is a message dialog (the toolkit's DialogState) that may also
// carry a content of its own, drawn between its body and input: the typed
// form of an MCP tool, which the mcp component owns.
type dialog struct {
	DialogState
	move MoveState // where the user dragged it
	// content builds the dialog's own body, between the lead text and the
	// buttons; width is the panel's width, zero for a message dialog's.
	content func(ctx *w.Ctx) Node
	width   int
}

// Component is the overlays of the window.
type Component struct {
	deps Deps

	// The children, built here and wired to each other here.
	Toasts   *toasts.Component
	Welcome  *welcome.Component
	Browser  *filebrowser.Component
	Settings *settingsdialog.Component

	calls      calltrace.Log
	dialog     *dialog
	picker     *destpicker.Picker
	pickerHost destpicker.Host
}

// New is the overlays with nothing open.
func New(deps Deps) *Component {
	c := &Component{deps: deps}
	c.Toasts = toasts.New(toasts.Deps{Clock: deps.Clock, Window: deps.Window})
	c.Welcome = welcome.New(welcome.Deps{Dialogs: c, Window: deps.Window})
	c.Browser = filebrowser.New(filebrowser.Deps{
		Desktop: deps.Desktop, Folders: deps.Folders, Bus: deps.Bus, Window: browserWindow{c}, Dialogs: asking{c},
	})
	c.Settings = settingsdialog.New(deps.Settings)
	return c
}

// browserWindow is the window as the file browser asks for it: its size and
// the toasts over it.
type browserWindow struct{ *Component }

// Size is the window's size in logical pixels.
func (host browserWindow) Size() (int, int) { return host.deps.Window.Size() }

// Toast shows a message over the window.
func (host browserWindow) Toast(text string, level severity.Level) { host.Toasts.Show(text, level) }

// asking is the dialog as the file browser asks for it: a question over it,
// and whether one is open.
type asking struct{ *Component }

// Asking reports whether a dialog is open over the browser.
func (host asking) Asking() bool { return host.Busy() }

// Confirm asks a yes/no question; onOK runs only on the primary button or Enter.
func (c *Component) Confirm(title, body, okLabel string, onOK func()) {
	c.dialog = &dialog{DialogState: DialogState{Title: title, Body: body, Buttons: []DialogButton{
		{Label: "Cancel", Run: c.CloseDialog},
		{Label: okLabel, Primary: true, Run: func() { c.CloseDialog(); onOK() }},
	}}}
}

// Prompt asks for a line of text, starting with initial selected.
func (c *Component) Prompt(title, initial, okLabel string, onOK func(text string)) {
	input := NewTextField(initial)
	input.Focused = true
	if initial != "" {
		input.SelectAll()
	}
	c.dialog = &dialog{DialogState: DialogState{Title: title, Input: &input, Buttons: []DialogButton{
		{Label: "Cancel", Run: c.CloseDialog},
		{Label: okLabel, Primary: true, Run: func() { c.CloseDialog(); onOK(input.Text) }},
	}}}
}

// Open shows state as the dialog, in place of any open one, with content as
// its own body between the lead text and the buttons when it has one, in a
// panel width wide (zero is a message dialog's). The returned func closes
// that dialog and no other.
func (c *Component) Open(state DialogState, content func(ctx *w.Ctx) Node, width int) (close func()) {
	open := &dialog{DialogState: state, content: content, width: width}
	c.dialog = open
	return func() {
		if c.dialog == open {
			c.dialog = nil
		}
	}
}

// Busy reports whether a dialog is open.
func (c *Component) Busy() bool { return c.dialog != nil }

// Default runs the primary button of the open dialog: what Enter does.
func (c *Component) Default() {
	if c.dialog != nil {
		c.dialog.Default()
	}
}

// CloseDialog closes the open dialog, whichever it is.
func (c *Component) CloseDialog() { c.dialog = nil }

// Dialog is the state of the open dialog, nil when none is open.
func (c *Component) Dialog() *DialogState {
	if c.dialog == nil {
		return nil
	}
	return &c.dialog.DialogState
}

// OpenPicker shows the destination picker; host answers what it asks.
func (c *Component) OpenPicker(picker *destpicker.Picker, host destpicker.Host) {
	c.picker, c.pickerHost = picker, host
}

// Picker is the open destination picker, nil when none is open.
func (c *Component) Picker() *destpicker.Picker { return c.picker }

// ClosePicker closes the destination picker.
func (c *Component) ClosePicker() { c.picker, c.pickerHost = nil, nil }

// Calls is the call trace behind the Log button; the pointer is stable.
func (c *Component) Calls() *calltrace.Log { return &c.calls }

// Layers are the layers of Escape that are open and belong to the overlays;
// the window fills the rest (the table cell, the find bar, the request).
func (c *Component) Layers() overlay.Open {
	var open overlay.Open
	open[overlay.ContextMenu] = menus.Shown(c.deps.Ctx)
	open[overlay.Dialog] = c.dialog != nil
	open[overlay.DestPicker] = c.picker != nil
	open[overlay.FileBrowser] = c.Browser.Shown()
	open[overlay.CallLog] = c.calls.Shown()
	open[overlay.FontList] = c.Settings.FontListOpen()
	open[overlay.Settings] = c.Settings.Open()
	return open
}

// Close closes layer when it is one of the overlays' and reports whether it
// was, so the window closes the others itself.
func (c *Component) Close(layer overlay.Layer) bool {
	closers := map[overlay.Layer]func(){
		overlay.ContextMenu: func() { menus.Close(c.deps.Ctx) },
		overlay.Dialog:      c.CloseDialog,
		overlay.DestPicker:  c.ClosePicker,
		overlay.FileBrowser: c.Browser.Close,
		overlay.CallLog:     c.calls.Close,
		overlay.FontList:    c.Settings.CloseFontList,
		overlay.Settings:    c.Settings.Close,
	}
	closer, own := closers[layer]
	if own {
		closer()
	}
	return own
}

// HoldsKeyboard reports whether a menu, dialog, picker, browser or the call
// log is open: the layers whose keys the text under them must not act on.
func (c *Component) HoldsKeyboard() bool { return c.Layers().HoldsKeyboard() }
