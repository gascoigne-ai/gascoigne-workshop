package app

import (
	"cmp"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/overlay"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/menus"

	"github.com/hajimehoshi/ebiten/v2"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The dialog is state on the App; the menu, the toasts, the file browser and
// the welcome notes are components in ui/overlays. All are built into the
// window's Stack after everything else, newest on top.

// dialog is a message dialog (the toolkit's DialogState) that may also
// carry a content of its own, drawn between its body and input: the typed
// form of an MCP tool, which the mcp component owns and hands in through
// mcpHost.Open.
type dialog struct {
	DialogState
	move MoveState // where the user dragged it
	// content builds the dialog's own body, between the lead text and the
	// buttons; width is the panel's width, zero for a message dialog's.
	content func() Node
	width   int
}

// confirm asks a yes/no question; onOK runs only on the primary button or Enter.
func (a *App) confirm(title, body, okLabel string, onOK func()) {
	a.dialog = &dialog{DialogState: DialogState{Title: title, Body: body, Buttons: []DialogButton{
		{Label: "Cancel", Run: a.closeDialog},
		{Label: okLabel, Primary: true, Run: func() { a.closeDialog(); onOK() }},
	}}}
}

// prompt asks for a line of text, starting with initial selected.
func (a *App) prompt(title, initial, okLabel string, onOK func(text string)) {
	input := NewTextField(initial)
	input.Focused = true
	if initial != "" {
		input.SelectAll()
	}
	a.dialog = &dialog{DialogState: DialogState{Title: title, Input: &input, Buttons: []DialogButton{
		{Label: "Cancel", Run: a.closeDialog},
		{Label: okLabel, Primary: true, Run: func() { a.closeDialog(); onOK(input.Text) }},
	}}}
}

func (a *App) closeDialog() { a.dialog = nil }

// dialogDefault runs the primary button: what Enter does.
func (a *App) dialogDefault() {
	if a.dialog != nil {
		a.dialog.Default()
	}
}

var dialogTemplate = markup.Must("dialog.vml", dialogSource)

// dialogView is the open dialog, nothing when none is open: a template
// always builds its element, so Go decides whether there is one.
func (a *App) dialogView() Node {
	d := a.dialog
	if d == nil {
		return nil
	}
	return dialogTemplate.MustBuild(markup.Scope{
		"width":   cmp.Or(d.width, w.DialogW),
		"state":   &d.DialogState,
		"move":    &d.move,
		"dismiss": a.closeDialog,
		"form":    a.dialogForm(d),
	})
}

// ---- keyboard --------------------------------------------------------------

// overlayKeys handles the keys of an open menu (Up, Down, Enter) or
// dialog (Enter, for one without a text field, which has its own), or of
// the file browser when no question is asked over it.
func (a *App) overlayKeys(in w.Input) {
	enter := in.Enter || in.Pressed(ebiten.KeyEnter)
	switch {
	case menus.Shown(a.ctx):
		menus.Keys(a.ctx, in)
	case a.dialog != nil && a.dialog.Input == nil && enter:
		a.dialogDefault()
	case a.dialog == nil && a.picker != nil:
		a.pickerKeys(in)
	case a.browser.Shown():
		a.browser.Keys(a.ctx, in)
	}
}

// overlayOpen reports whether a menu or dialog has the keyboard.
func (a *App) overlayOpen() bool {
	return a.openLayers().HoldsKeyboard()
}

// pickerView is the destination picker while it is open, nothing otherwise.
func (a *App) pickerView() Node {
	if a.picker == nil {
		return nil
	}
	return a.picker.View(a.ctx, a.pickerHost)
}

// dialogForm is the content of a dialog, nothing for the others.
func (a *App) dialogForm(open *dialog) Node {
	if open.content == nil {
		return nil
	}
	return open.content()
}

// openLayers are the things Escape can close right now.
func (a *App) openLayers() overlay.Open {
	var open overlay.Open
	open[overlay.ContextMenu] = menus.Shown(a.ctx)
	open[overlay.Dialog] = a.dialog != nil
	open[overlay.DestPicker] = a.picker != nil
	open[overlay.FileBrowser] = a.browser.Shown()
	open[overlay.CallLog] = a.calls.Shown()
	open[overlay.FontList] = a.prefs.FontListOpen()
	open[overlay.Settings] = a.prefs.Open()
	open[overlay.CellEdit] = a.kv.Editing()
	open[overlay.FindBar] = a.resp.Look(a.cur().Tab).FindOpen
	open[overlay.Request] = true // the last resort: cancel what is being sent
	return open
}

// escape does one step of Escape: it closes the topmost open layer.
func (a *App) escape() {
	closers := [...]func(){
		overlay.ContextMenu: func() { menus.Close(a.ctx) },
		overlay.Dialog:      a.closeDialog,
		overlay.DestPicker:  a.closePicker,
		overlay.FileBrowser: a.browser.Close,
		overlay.CallLog:     a.calls.Close,
		overlay.FontList:    a.prefs.CloseFontList,
		overlay.Settings:    a.prefs.Close,
		overlay.CellEdit:    a.kv.CancelEdit,
		overlay.FindBar:     func() { a.resp.CloseFind(a.cur().Tab) },
		overlay.Request:     func() { a.cancelSend(a.cur()) },
	}
	if layer, ok := a.openLayers().Top(); ok {
		closers[layer]()
	}
}

// Templates, parsed once above.
const (
	dialogSource = `<!-- A dialog's own content, an MCP tool's typed form for one, sits between its body and its input. -->
<message-dialog w="{{width}}" state="state" moved="move" on-dismiss="dismiss">
  <slot if="form" name="form"/>
</message-dialog>
`
)
