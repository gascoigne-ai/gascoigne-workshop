package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/overlay"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Keys hands the keys of the frame to the overlays, the tree, the editor and
// response, the shortcuts and the table undo, in that order of claim.
//
// Each step reads the gates of the ones before it. The tree gets keys even
// while a field has them (it needs the shortcuts to keep running there); the
// shortcuts stand back for a field that took the frame as typing. A strict
// "first claim wins" chain was rejected: the tree, the text fields and the
// shortcuts share keys today (Ctrl+Enter, Alt+Down), and this chain keeps
// those gates exactly as they were. Revisit when the shared keys get their
// own owner.
func (a *Component) Keys(in core.Input) {
	held, recording := a.over.HoldsKeyboard(), a.over.Settings.Recording()
	a.over.Keys(a.ctx, in)
	if !held && !a.over.HoldsKeyboard() && !recording {
		a.side.Keys(in, a.typing(), a.sidebarWidth())
	}
	text := !a.over.HoldsKeyboard() && (a.editor.Keys(in) || a.resp.Keys(in))
	if !a.over.Settings.Keys(in) && !a.over.HoldsKeyboard() && !typedInText(text, in) {
		a.runKeymap(in)
	}
	if in.Escape && !recording { // one step per press: overlays, the settings menu, a cell edit, find, a running request
		a.escape()
	}
	if !a.over.HoldsKeyboard() && !text {
		a.editor.TableKeys(in, a.typing())
	}
}

// typing reports whether a text widget has the keyboard: the tree's filter,
// a field of the request pane or the response's find bar.
func (a *Component) typing() bool {
	return a.side.Typing() || a.editor.Typing() || a.resp.Typing()
}

// openLayers are the things Escape can close right now: the overlays' own
// and the window's.
func (a *Component) openLayers() overlay.Open {
	open := a.over.Layers()
	open[overlay.CellEdit] = a.editor.Tables.Editing()
	open[overlay.FindBar] = a.resp.Look(a.cur().Tab).FindOpen
	open[overlay.Request] = true // the last resort: cancel what is being sent
	return open
}

// escape does one step of Escape: it closes the topmost open layer. The
// overlays close their own; the table, the find bar and the request are the
// window's.
func (a *Component) escape() {
	layer, ok := a.openLayers().Top()
	if !ok || a.over.Close(layer) {
		return
	}
	closers := map[overlay.Layer]func(){
		overlay.CellEdit: a.editor.Tables.CancelEdit,
		overlay.FindBar:  func() { a.resp.CloseFind(a.cur().Tab) },
		overlay.Request:  func() { a.cancelSend(a.cur()) },
	}
	closers[layer]()
}

// typedInText reports whether the URL field or the body took the frame's keys
// as typing, which no shortcut may take. A claim for Undo or Redo is not that:
// it keeps the table's Undo from running, but a user binding on the same keys
// still fires.
func typedInText(claimed bool, in core.Input) bool { return claimed && !in.Undo && !in.Redo }
