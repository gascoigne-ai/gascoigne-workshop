package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/overlay"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/toasts"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Overlays (context menu, dialog, toasts) are state on the App and are
// built into the window's Stack after everything else, newest on top.

const (
	menuMinW    = 120 // a menu is as wide as its content, within these
	menuMaxW    = 420
	menuHintH   = 16 // the extra height of a row with a hint
	menuRowH    = 24
	menuSepH    = 9
	menuPad     = 4
	dialogW     = 380
	toastW      = 320
	toastH      = 36
	toastGap    = 6
	toastMargin = 8
)

// menuItem is one row of a context menu. A separator has no label.
type menuItem struct {
	label, shortcut string
	hint            string // a muted line under the label saying what the item is
	icon            string // an icon name; "" takes the one the label's start maps to (menuIcons)
	disabled        bool
	separator       bool
	run             func()
}

func (m menuItem) selectable() bool { return !m.separator && !m.disabled }

// contextMenu is the open context menu: where it is, its rows, and the
// highlighted row for the keyboard.
type contextMenu struct {
	x, y  int
	items []menuItem
	hot   int
}

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
func (a *App) openMenu(x, y int, items []menuItem) {
	m := &contextMenu{x: x, y: y, items: items}
	m.hot = nextSelectable(items, -1, 1)
	a.ctxMenu = m
}

func (a *App) closeMenu() { a.ctxMenu = nil }

// nextSelectable is the index of the next enabled, non-separator row
// from index from in direction dir (1 or -1), wrapping; -1 if none.
func nextSelectable(items []menuItem, from, dir int) int {
	n := len(items)
	for step := 1; step <= n; step++ {
		i := ((from+dir*step)%n + n) % n
		if items[i].selectable() {
			return i
		}
	}
	return -1
}

// runMenuItem closes the menu, then runs row i, so the action may open
// another menu or dialog.
func (a *App) runMenuItem(i int) {
	if a.ctxMenu == nil || !a.ctxMenu.items[i].selectable() {
		return
	}
	run := a.ctxMenu.items[i].run
	a.closeMenu()
	if run != nil {
		run()
	}
}

func (a *App) contextMenuView() Node {
	m := a.ctxMenu
	if m == nil {
		return nil
	}
	return Col(W(a.menuWidth(m)), Pad(menuPad), Bg(w.Bg), Control("menu"), Shadow(),
		FloatAt(m.x, m.y), Modal(), OnDismiss(a.closeMenu),
		For(m.items, func(i int, it menuItem) Node { return a.menuRow(m, i, it) }),
	)
}

func (a *App) menuRow(m *contextMenu, i int, it menuItem) Node {
	if it.separator {
		return Row(H(menuSepH), PadX(6), HRule())
	}
	color := w.Text
	if it.disabled {
		color = w.Muted
	}
	look := HoverBg(w.Hover)
	switch {
	case it.disabled:
		look = Gap(0)
	case i == m.hot:
		look = Bg(w.Selected)
	}
	var icon Node
	if menuHasIcons(m.items) { // the column exists only in a menu where some row has an icon
		icon = Spacer(W(menuIconW))
		if name := menuIcon(it); name != "" {
			icon = Icon(name, W(menuIconW), Color(color))
		}
	}
	label := Node(Text(it.label, Color(color), Grow(1)))
	height := menuRowH
	if it.hint != "" {
		lines := strings.Split(it.hint, "\n") // each line of the hint is a muted line under the label
		parts := []Item{Grow(1), Text(it.label, Color(color))}
		for _, line := range lines {
			parts = append(parts, Text(line, Small(), Muted(), Ellipsis()))
		}
		label = Col(parts...)
		height += menuHintH * len(lines)
	}
	return Row(H(height), PadX(10), Gap(8), AlignItems(AlignCenter), Control("row"), look, OnClick(func() { a.runMenuItem(i) }),
		icon,
		label,
		If(it.shortcut != "", Text(w.PlatformKeyText(runtime.GOOS, it.shortcut), Small(), Muted())),
	)
}

// menuHasIcons reports whether any row of the menu has an icon.
func menuHasIcons(items []menuItem) bool {
	for _, item := range items {
		if !item.separator && menuIcon(item) != "" {
			return true
		}
	}
	return false
}

// menuWidth is the width of menu m: its widest row, within menuMinW and
// menuMaxW. A row is its icon column (if the menu has one), the label or the
// hint, whichever is wider, and the shortcut.
func (a *App) menuWidth(m *contextMenu) int {
	widest := 0
	icons := menuHasIcons(m.items)
	for _, item := range m.items {
		if item.separator {
			continue
		}
		row := a.ctx.Width(a.ctx.F.UI, item.label)
		for _, line := range strings.Split(item.hint, "\n") {
			row = max(row, a.ctx.Width(a.ctx.F.Small, line))
		}
		if item.shortcut != "" {
			row += 16 + a.ctx.Width(a.ctx.F.Small, w.PlatformKeyText(runtime.GOOS, item.shortcut))
		}
		if icons {
			row += menuIconW + 8
		}
		widest = max(widest, row)
	}
	return min(max(widest+2*10+2*menuPad, menuMinW), menuMaxW)
}

// ---- dialogs ---------------------------------------------------------------

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
		m := a.ctxMenu
		if in.Pressed(ebiten.KeyArrowDown) {
			m.hot = nextSelectable(m.items, m.hot, 1)
		}
		if in.Pressed(ebiten.KeyArrowUp) {
			m.hot = nextSelectable(m.items, m.hot, -1)
		}
		if enter && m.hot >= 0 {
			a.runMenuItem(m.hot)
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
