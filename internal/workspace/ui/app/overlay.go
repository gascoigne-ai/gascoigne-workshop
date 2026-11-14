package ui

import (
	"embed"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/overlay"

	"github.com/hajimehoshi/ebiten/v2"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Overlays (context menu, dialog, toasts) are state on the App and are
// built into the window's Stack after everything else, newest on top.

const ()

// dialog is a message dialog (the toolkit's DialogState) that may also
// carry the typed form of an MCP tool, drawn between its body and input.
type dialog struct {
	DialogState
	form *mcpform.Form
}

// ---- context menu --------------------------------------------------------

// openMenu shows a context menu at the pointer position x, y.
func (a *App) openMenu(x, y int, items []MenuItem) {
	OpenMenu(a.ctx, x, y, withIcons(items))
}

func (a *App) closeMenu() { CloseMenu(a.ctx) }

// withIcons gives every item without an icon the one its label maps to.
func withIcons(items []MenuItem) []MenuItem {
	out := make([]MenuItem, len(items))
	for i, item := range items {
		item.Icon = menuIcon(item)
		out[i] = item
	}
	return out
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

//go:embed dialog.vml toasts.vml
var overlayTemplates embed.FS

var (
	dialogTemplate = markup.MustLoad(overlayTemplates, "dialog.vml")
	toastsTemplate = markup.MustLoad(overlayTemplates, "toasts.vml")
)

// dialogView is the open dialog, nothing when none is open: a template
// always builds its element, so Go decides whether there is one.
func (a *App) dialogView() Node {
	d := a.dialog
	if d == nil {
		return nil
	}
	return dialogTemplate.MustBuild(markup.Scope{
		"state":   &d.DialogState,
		"dismiss": a.closeDialog,
		"form":    a.dialogForm(d),
	})
}

// ---- toasts ----------------------------------------------------------------

// toast shows a message: an info one fades after a few seconds, an
// error stays until it is clicked.
func (a *App) toast(text string, isErr bool) {
	if a.toasts.Show(text, isErr, a.now()) {
		a.dirty = true
	}
}

// pruneToasts drops the info toasts whose time is up.
func (a *App) pruneToasts() bool { return a.toasts.Prune(a.now()) }

// clipboardNote reports what the clipboard did: failures are error
// toasts, a successful copy is a status line.
func (a *App) clipboardNote(msg string, isErr bool) {
	if isErr {
		a.toast(msg, true)
		return
	}
	a.status = msg
}

// toastsView is the stack of toasts in the window's bottom-right corner,
// above the status bar.
func (a *App) toastsView() Node {
	return toastsTemplate.MustBuild(markup.Scope{
		"stack":   &a.toasts,
		"width":   a.w,
		"height":  a.h,
		"statusH": statusH,
	})
}

// ---- keyboard --------------------------------------------------------------

// overlayKeys handles the keys of an open menu (Up, Down, Enter) or
// dialog (Enter, for one without a text field, which has its own).
func (a *App) overlayKeys(in w.Input) {
	enter := in.Enter || in.Pressed(ebiten.KeyEnter)
	switch {
	case MenuOpen(a.ctx):
		PopupMenuKeys(a.ctx, in)
	case a.dialog != nil && a.dialog.Input == nil && enter:
		a.dialogDefault()
	}
}

// swallowMenuKeys removes the keys a menu used from this frame's input, so
// the text widget that still has the focus under it does not act on them too.
func (a *App) swallowMenuKeys() {
	a.ctx.In.Consume(ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyEscape, ebiten.KeyTab)
}

// overlayOpen reports whether a menu or dialog has the keyboard.
func (a *App) overlayOpen() bool { return MenuOpen(a.ctx) || a.dialog != nil || a.calls.Shown() }

// dialogForm is the typed form of a dialog, nothing for the others.
func (a *App) dialogForm(d *dialog) Node {
	if d.form == nil {
		return nil
	}
	return d.form.View(a)
}

// openLayers are the things Escape can close right now.
func (a *App) openLayers() overlay.Open {
	var open overlay.Open
	open[overlay.ContextMenu] = MenuOpen(a.ctx)
	open[overlay.Dialog] = a.dialog != nil
	open[overlay.CallLog] = a.calls.Shown()
	open[overlay.FontList] = a.prefs.FontListOpen()
	open[overlay.Settings] = a.menu
	open[overlay.CellEdit] = a.edit != nil
	open[overlay.FindBar] = a.cur().look.FindOpen
	open[overlay.Request] = true // the last resort: cancel what is being sent
	return open
}

// escape does one step of Escape: it closes the topmost open layer.
func (a *App) escape() {
	closers := [...]func(){
		overlay.ContextMenu: a.closeMenu,
		overlay.Dialog:      a.closeDialog,
		overlay.CallLog:     a.calls.Close,
		overlay.FontList:    a.prefs.CloseFontList,
		overlay.Settings:    a.closeSettings,
		overlay.CellEdit:    a.cancelEdit,
		overlay.FindBar:     func() { a.closeFind(a.cur()) },
		overlay.Request:     func() { a.cancelSend(a.cur()) },
	}
	if layer, ok := a.openLayers().Top(); ok {
		closers[layer]()
	}
}
