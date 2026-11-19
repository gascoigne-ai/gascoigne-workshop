package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/settingsui"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// cursorShapes maps the cursor widgets ask for to the window system's.
var cursorShapes = map[core.Cursor]ebiten.CursorShapeType{
	core.CursorDefault:    ebiten.CursorShapeDefault,
	core.CursorResizeH:    ebiten.CursorShapeEWResize,
	core.CursorResizeV:    ebiten.CursorShapeNSResize,
	core.CursorMove:       ebiten.CursorShapeMove,
	core.CursorResizeNWSE: ebiten.CursorShapeNWSEResize,
	core.CursorResizeNESW: ebiten.CursorShapeNESWResize,
}

// Layout implements ebiten.Game. The UI is laid out in logical pixels
// (width by height); the screen is that size times the monitor's scale factor so
// text and lines stay sharp on HiDPI displays.
func (a *App) Layout(width, height int) (int, int) {
	reported := ebiten.Monitor().DeviceScaleFactor()
	f := frameSize(width, height, reported, a.displayScale(reported), a.uiScale())
	a.windowScale = f.WindowScale
	a.setSize(f.LW, f.LH, f.Scale)
	return f.SW, f.SH
}

// Update implements ebiten.Game: it samples input and runs one frame.
func (a *App) Update() error {
	a.winX, a.winY = ebiten.WindowPosition()
	a.step(a.sampler.Read().Scaled(a.ctx.Scale))
	if title := a.Title(); title != a.title {
		a.title = title
		ebiten.SetWindowTitle(title)
	}
	ebiten.SetCursorShape(cursorShapes[a.ctx.Cursor])
	if tps := a.pacer.TPS(); tps != a.tps {
		ebiten.SetTPS(tps)
		a.tps = tps
	}
	return a.shotUpdate()
}

// step runs one frame with the given input: it applies finished sends,
// builds the view, lays it out and lets it react. Handlers in the view
// change App state. Tests drive the app through step without a window.
func (a *App) step(in core.Input) {
	a.ctx.Tick++
	a.ctx.In = in
	a.ctx.Cursor = core.CursorDefault
	if !in.Held && !in.MiddleHeld {
		a.updateTabs(tabs.DragEnd{})
		a.dragRow = nil
	}
	drained := a.drain()
	pruned := a.pruneToasts()
	active := engine.InputActive(a.lastIn, in)
	a.noteGeometry()
	if active || drained || pruned {
		a.markStale()
	}
	if in.RestartsCaret() {
		a.ctx.RestartCaret()
	}
	a.lastIn = in
	if in.Held {
		a.dropTarget = model.Node{} // set again by the row under the pointer
	} else {
		a.finishDrag()
	}
	hadOverlay := a.overlayOpen()
	wasRecording := a.shortcutKeys.Recording() != ""
	menuWasOpen := widget.MenuOpen(a.ctx)
	a.overlayKeys(in)
	if menuWasOpen {
		a.swallowMenuKeys()
	}
	if !hadOverlay && !a.overlayOpen() && !wasRecording {
		a.treeKeys(in)
	}
	if wasRecording {
		a.shortcutKeys.RecordKey(a, in)
	} else if !a.overlayOpen() {
		a.runShortcuts(in)
	}
	if in.Escape && !wasRecording { // one step per press: overlays, the settings menu, a cell edit, find, a running request
		a.escape()
	}
	if (in.Undo || in.Redo) && !a.overlayOpen() && !a.textFocused() {
		if p, t, ok := a.visibleTable(); ok {
			if in.Undo {
				a.undoRows(p, t)
			} else {
				a.redoRows(p, t)
			}
		}
	}
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	view.Update(a.ctx, root)
	if a.edit != nil && !a.edit.field.Focused { // a click elsewhere ends the edit
		a.commitEdit()
	}
	if a.ctx.Tick%settingsui.PollTicks == 0 {
		a.prefs.Poll(a)
	}
	a.dragHover()
	a.showGrab()
	a.settleBulk()
	a.stateTick()
	a.workspaceTick()
	a.markFrame(active || drained || pruned || a.tips.Waiting())
	a.pacer.NoteIdle(active || drained || pruned || a.tips.Waiting())
}

// Draw implements ebiten.Game. The view is rebuilt so it shows the
// state the handlers just changed.
func (a *App) Draw(screen *ebiten.Image) {
	if !a.dirty && !a.alwaysDraw() {
		return // nothing changed: the screen keeps what it has and the GPU stays idle
	}
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	view.Draw(a.ctx, root, screen)
	a.dirty = false
	a.shotDraw(screen)
}

// ---- keyboard ---------------------------------------------------------------

// textFocused reports whether a text widget has the keyboard.
func (a *App) textFocused() bool {
	t := a.cur()
	focused := a.filter.Focused || t.url.Focused || t.find.Focused || t.timeout.Focused || t.body.Focused ||
		t.tests.Focused || t.gqlVars.Focused || t.file.Focused || t.msg.Focused || a.edit != nil ||
		(a.bulk != nil && a.bulk.area.Focused)
	for _, in := range t.auth {
		focused = focused || in.Focused
	}
	return focused
}
