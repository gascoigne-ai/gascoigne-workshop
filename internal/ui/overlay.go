package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	. "github.com/gascoigne-ai/gascoigne-workshop/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Overlays (context menu, dialog, toasts) are state on the App and are
// built into the window's Stack after everything else, newest on top.

const (
	menuListW    = 220
	menuRowH     = 24
	menuSepH     = 9
	menuPad      = 4
	dialogW      = 380
	toastW       = 320
	toastH       = 36
	toastGap     = 6
	toastMargin  = 8
	maxToasts    = 5 // older toasts are dropped beyond this
	infoToastFor = 3 * time.Second
)

// menuItem is one row of a context menu. A separator has no label.
type menuItem struct {
	label, shortcut string
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
	input       *w.TextField
	buttons     []dialogButton
}

// toast is a short message over the window.
type toast struct {
	text    string
	isErr   bool
	expires time.Time // zero: stays until clicked
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
	return Col(W(menuListW), Pad(menuPad), Bg(w.Bg), Border(w.Border), Shadow(),
		FloatAt(m.x, m.y), Modal(a.closeMenu),
		For(m.items, func(i int, it menuItem) Node { return a.menuRow(m, i, it) }),
	)
}

func (a *App) menuRow(m *contextMenu, i int, it menuItem) Node {
	if it.separator {
		return Row(H(menuSepH), PadXY(6, 0), HRule())
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
	return Row(H(menuRowH), PadXY(10, 0), Gap(8), AlignItems(AlignBaseline), look, OnClick(func() { a.runMenuItem(i) }),
		Text(it.label, Color(color)),
		Spacer(),
		Text(it.shortcut, Small(), Muted()),
	)
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
	input := w.NewTextField(initial)
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
		Col(W(dialogW), Pad(16), Gap(12), Bg(w.Bg), Border(w.Border), Shadow(),
			FloatCenter(), FocusScope(), Modal(a.closeDialog),
			Text(d.title, H(22)),
			If(d.body != "", Lines(d.body, w.Muted)),
			If(d.input != nil, Field(d.input, H(w.RowH), Bg(w.Sunken), Border(w.Border), OnEnter(a.dialogDefault), a.textMenuOpt(d.input, nil))),
			Row(H(w.RowH), Gap(8),
				Spacer(),
				For(d.buttons, func(_ int, b dialogButton) Node {
					if b.primary {
						return PrimaryButton(b.label, W(80), OnClick(b.run))
					}
					return Button(b.label, W(80), OnClick(b.run))
				}),
			),
		),
	)
}

// ---- toasts ----------------------------------------------------------------

// toast shows a message: an info one fades after a few seconds, an
// error stays until it is clicked.
func (a *App) toast(text string, isErr bool) {
	t := toast{text: text, isErr: isErr}
	if !isErr {
		t.expires = a.now().Add(infoToastFor)
	}
	a.toasts = append(a.toasts, t)
	if len(a.toasts) > maxToasts {
		a.toasts = a.toasts[len(a.toasts)-maxToasts:]
	}
}

// pruneToasts drops the info toasts whose time is up.
func (a *App) pruneToasts() {
	now := a.now()
	kept := a.toasts[:0]
	for _, t := range a.toasts {
		if t.expires.IsZero() || now.Before(t.expires) {
			kept = append(kept, t)
		}
	}
	a.toasts = kept
}

func (a *App) dismissToast(i int) { a.toasts = append(a.toasts[:i], a.toasts[i+1:]...) }

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
		FloatAt(a.w-w.Pad-toastW, a.h-statusH-toastMargin-total),
		For(a.toasts, func(i int, t toast) Node {
			border, color := w.Border, w.Text
			if t.isErr {
				border, color = w.Error, w.Error
			}
			return Row(H(toastH), PadXY(w.Pad, 0), Bg(w.Panel), Border(border), Shadow(), OnClick(func() { a.dismissToast(i) }),
				Text(t.text, Ellipsis(), Color(color), Grow(1)),
			)
		}),
	)
}

// ---- keyboard --------------------------------------------------------------

// overlayKeys handles the keys of an open menu (Up, Down, Enter) or
// dialog (Enter, for one without a text field, which has its own).
func (a *App) overlayKeys(in w.Input) {
	enter := in.Enter || hasKey(in.Keys, ebiten.KeyEnter)
	switch {
	case a.ctxMenu != nil:
		m := a.ctxMenu
		if hasKey(in.Keys, ebiten.KeyArrowDown) {
			m.hot = nextSelectable(m.items, m.hot, 1)
		}
		if hasKey(in.Keys, ebiten.KeyArrowUp) {
			m.hot = nextSelectable(m.items, m.hot, -1)
		}
		if enter && m.hot >= 0 {
			a.runMenuItem(m.hot)
		}
	case a.dialog != nil && a.dialog.input == nil && enter:
		a.dialogDefault()
	}
}

func hasKey(keys []ebiten.Key, k ebiten.Key) bool {
	for _, p := range keys {
		if p == k {
			return true
		}
	}
	return false
}

// overlayOpen reports whether a menu or dialog has the keyboard.
func (a *App) overlayOpen() bool { return a.ctxMenu != nil || a.dialog != nil }
