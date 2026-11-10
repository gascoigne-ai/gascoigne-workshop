package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/overlay"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/toasts"

	"github.com/hajimehoshi/ebiten/v2"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Overlays (context menu, dialog, toasts) are state on the App and are
// built into the window's Stack after everything else, newest on top.

const (
	dialogW     = 380
	toastW      = 320
	toastH      = 36
	toastGap    = 6
	toastMargin = 8
)

// dialogButton is a button of a dialog.
type dialogButton struct {
	label   string
	primary bool
	run     func()
}

// dialog is a modal message with buttons and, for a prompt, a text field.
type dialog struct {
	title, body string
	input       *TextField
	form        *mcpform.Form // a typed MCP tool form, instead of a single input
	buttons     []dialogButton
}

// ---- context menu --------------------------------------------------------

// openMenu shows a context menu at the pointer position x, y.
func (a *App) openMenu(x, y int, items []MenuItem) {
	a.ctxMenu = NewMenu(x, y, withIcons(items))
}

func (a *App) closeMenu() { a.ctxMenu = nil }

// withIcons gives every item without an icon the one its label maps to.
func withIcons(items []MenuItem) []MenuItem {
	out := make([]MenuItem, len(items))
	for i, item := range items {
		item.Icon = menuIcon(item)
		out[i] = item
	}
	return out
}

// runMenuItem closes the menu, then runs row i, so the action may open
// another menu or dialog.
func (a *App) runMenuItem(i int) {
	if a.ctxMenu == nil || !a.ctxMenu.Items[i].Selectable() {
		return
	}
	run := a.ctxMenu.Items[i].Run
	a.closeMenu()
	if run != nil {
		run()
	}
}

func (a *App) contextMenuView() Node {
	if a.ctxMenu == nil {
		return nil
	}
	return Menu(a.ctx, a.ctxMenu, OnSelect(a.runMenuItem), OnDismiss(a.closeMenu))
}

// confirm asks a yes/no question; onOK runs only on the primary button or Enter.
func (a *App) confirm(title, body, okLabel string, onOK func()) {
	a.dialog = &dialog{title: title, body: body, buttons: []dialogButton{
		{label: "Cancel", run: a.closeDialog},
		{label: okLabel, primary: true, run: func() { a.closeDialog(); onOK() }},
	}}
}

// prompt asks for a line of text, starting with initial selected.
func (a *App) prompt(title, initial, okLabel string, onOK func(text string)) {
	input := NewTextField(initial)
	input.Focused = true
	if initial != "" {
		input.SelectAll()
	}
	d := &dialog{title: title, input: &input}
	d.buttons = []dialogButton{
		{label: "Cancel", run: a.closeDialog},
		{label: okLabel, primary: true, run: func() { a.closeDialog(); onOK(input.Text) }},
	}
	a.dialog = d
}

func (a *App) closeDialog() { a.dialog = nil }

// dialogDefault runs the primary button: what Enter does.
func (a *App) dialogDefault() {
	if a.dialog == nil {
		return
	}
	for _, b := range a.dialog.buttons {
		if b.primary {
			b.run()
			return
		}
	}
}

func (a *App) dialogView() Node {
	d := a.dialog
	if d == nil {
		return nil
	}
	return Frag(
		Row(Bg(w.Shadow)), // dims the window
		Col(W(dialogW), Pad(16), Gap(12), Bg(w.Bg), Control("menu"), Shadow(),
			FloatCenter(), FocusScope(), Modal(), OnDismiss(a.closeDialog),
			Text(d.title, H(w.LabelH)),
			If(d.body != "", Lines(d.body, w.Muted)),
			a.dialogForm(d),
			If(d.input != nil, Field(d.input, H(w.RowH), Bg(w.Sunken), OnEnter(a.dialogDefault), a.textMenuOpt(d.input, nil))),
			Row(H(w.RowH), Gap(8),
				Spacer(),
				For(d.buttons, func(_ int, b dialogButton) Node {
					if b.primary {
						return PrimaryButton(b.label, MinW(80), OnClick(b.run))
					}
					return Button(b.label, MinW(80), OnClick(b.run))
				}),
			),
		),
	)
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

func (a *App) dismissToast(i int) { a.toasts.Dismiss(i) }

// clipboardNote reports what the clipboard did: failures are error
// toasts, a successful copy is a status line.
func (a *App) clipboardNote(msg string, isErr bool) {
	if isErr {
		a.toast(msg, true)
		return
	}
	a.status = msg
}

func (a *App) toastsView() Node {
	if len(a.toasts) == 0 {
		return nil
	}
	total := len(a.toasts)*toastH + (len(a.toasts)-1)*toastGap
	return Col(W(toastW), Gap(toastGap),
		FloatAt(a.w-w.Gutter-toastW, a.h-statusH-toastMargin-total),
		For(a.toasts, func(i int, t toasts.Toast) Node {
			border, color := w.Border, w.Text
			if t.IsErr {
				border, color = w.Error, w.Error
			}
			return Row(H(toastH), PadX(w.Gutter), Bg(w.Panel), Border(border), Shadow(), OnClick(func() { a.dismissToast(i) }),
				Text(t.Text, Ellipsis(), Color(color), Grow(1)),
			)
		}),
	)
}

// ---- keyboard --------------------------------------------------------------

// overlayKeys handles the keys of an open menu (Up, Down, Enter) or
// dialog (Enter, for one without a text field, which has its own).
func (a *App) overlayKeys(in w.Input) {
	enter := in.Enter || in.Pressed(ebiten.KeyEnter)
	switch {
	case a.ctxMenu != nil:
		if picked := a.ctxMenu.Keys(in); picked >= 0 {
			a.runMenuItem(picked)
		}
	case a.dialog != nil && a.dialog.input == nil && enter:
		a.dialogDefault()
	}
}

// swallowMenuKeys removes the keys a menu used from this frame's input, so
// the text widget that still has the focus under it does not act on them too.
func (a *App) swallowMenuKeys() {
	a.ctx.In.Consume(ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyEscape, ebiten.KeyTab)
}

// overlayOpen reports whether a menu or dialog has the keyboard.
func (a *App) overlayOpen() bool { return a.ctxMenu != nil || a.dialog != nil || a.calls.Shown() }

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
	open[overlay.ContextMenu] = a.ctxMenu != nil
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
