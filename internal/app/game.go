package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"
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
	a.Step(a.sampler.Read().Scaled(a.ctx.Scale))
	if title := a.Title(); title != a.title {
		a.title = title
		ebiten.SetWindowTitle(title)
	}
	ebiten.SetCursorShape(cursorShapes[a.ctx.Cursor])
	if tps := a.pacer.TPS(); tps != a.tps {
		ebiten.SetTPS(tps)
		a.tps = tps
	}
	return nil
}

// Step runs one frame with the given input: it applies finished sends,
// builds the view, lays it out and lets it react. Handlers in the view
// change App state. Tests drive the app through Step without a window; it
// is exported only for apptest, the test-support package that lives outside
// package app so whole-app tests can move to e2e.
func (a *App) Step(in core.Input) {
	moved := a.beginFrame(in)
	a.routeKeys(in)
	a.endFrame(moved)
}

// shortcutsPageShown reports whether the Shortcuts page is what the settings
// dialog draws: not another page, the raw text or a search. Only then do its
// rows take the arrow keys and Enter.
func (a *App) shortcutsPageShown() bool {
	return a.menu && a.prefs.Cat == "shortcuts" && !a.prefs.Raw && !a.prefs.Searching()
}

// beginFrame takes in the input and what goroutines reported, and reports
// whether anything moved: input, a finished send or an expired toast.
func (a *App) beginFrame(in core.Input) bool {
	a.ctx.Tick++
	a.ctx.In = in
	a.ctx.Cursor = core.CursorDefault
	a.strip.BeginFrame(in)
	a.kv.BeginFrame(in)
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
	a.side.BeginFrame(in)
	return active || drained || pruned
}

// routeKeys hands the keys of the frame to the overlay, the tree, the
// shortcuts or the table undo, in that order of claim.
func (a *App) routeKeys(in core.Input) {
	hadOverlay := a.overlayOpen()
	wasRecording := a.shortcutKeys.Recording() != ""
	menuWasOpen := widget.MenuOpen(a.ctx)
	a.overlayKeys(in)
	if menuWasOpen {
		a.swallowMenuKeys()
	}
	if !hadOverlay && !a.overlayOpen() && !wasRecording {
		a.side.Keys(in, a.textFocused(), a.sidebarWidth())
	}
	if wasRecording {
		a.shortcutKeys.RecordKey(a, in)
	} else if a.shortcutsPageShown() {
		a.shortcutKeys.Navigate(in, a)
	} else if !a.overlayOpen() {
		a.runShortcuts(in)
	}
	if in.Escape && !wasRecording { // one step per press: overlays, the settings menu, a cell edit, find, a running request
		a.escape()
	}
	if !a.overlayOpen() {
		a.kv.Keys(in, a.textFocused())
	}
}

// endFrame builds the view once, lets it react and runs the per-frame
// upkeep; moved is what beginFrame reported. Draw paints the tree built here
// unless treeStale says that a handler or a step below changed what it shows.
func (a *App) endFrame(moved bool) {
	root := a.buildTree()
	view.Layout(a.ctx, root, a.w, a.h)
	fired := a.ctx.Fired
	view.Update(a.ctx, root)
	a.drawTree, a.treeStale = root, a.ctx.Fired != fired
	// The steps below run after the tree was built. Each one that changes
	// what the view shows sets a.treeStale itself (commitEdit, the kv Tick,
	// dragHover, saveCollection and the settings Poll do); the others only
	// read state or touch the cursor, the clock or a write queue.
	a.treeStale = a.kv.Tick() || a.treeStale // a click elsewhere ends a cell edit, leaving the table applies its text
	if a.ctx.Tick%settingsdialog.PollTicks == 0 {
		a.treeStale = true // an outside change of settings.json replaces settings and the raw text
		a.prefs.Poll(a)
	}
	a.dragHover()
	a.showGrab()
	a.stateTick()
	a.workspaceTick()
	busy := moved || a.tips.Waiting()
	a.markFrame(busy)
	a.pacer.NoteIdle(busy)
}

// Draw implements ebiten.Game. It paints the tree of the last step, built
// again only when something changed after that tree was built.
func (a *App) Draw(screen *ebiten.Image) {
	if !a.dirty && !a.alwaysDraw() {
		return // nothing changed: the screen keeps what it has and the GPU stays idle
	}
	view.Draw(a.ctx, a.frameTree(), screen)
	a.dirty = false
}

// ---- keyboard ---------------------------------------------------------------

// buildTree builds the window's view; the count is for tests.
func (a *App) buildTree() *view.El {
	a.builds++
	return a.view()
}

// frameTree is the laid out tree Draw paints: the one endFrame built when
// nothing it shows changed since, else a fresh build. A capture
// (Options.DrawEveryFrame) changes the scene after the step, so it always
// builds fresh. A revision
// counter bumped by every change was rejected: one missed bump loses an
// edit on screen; a test that compares the reused tree with a fresh build
// catches a missed treeStale instead
// (TestTheDrawnTreeMatchesAFreshBuildAcrossASession).
func (a *App) frameTree() *view.El {
	root := a.drawTree
	if root == nil || a.treeStale || a.drawEveryFrame {
		root = a.buildTree()
	}
	view.Layout(a.ctx, root, a.w, a.h)
	return root
}

// Tree is the laid-out view tree the next Draw paints. It is exported
// because apptest cannot build a dump or a widget query without it, and
// dump formatting belongs to the tests, not to production code. It is
// frameTree, so a stale tree is rebuilt and the builds counter moves.
func (a *App) Tree() view.Node { return a.frameTree() }

// SetSize sets the window size in logical pixels and nothing else. Unlike
// the Layout path (setSize) it leaves the device scale alone, which is what
// a test window of fixed size wants; apptest calls it before the first Step.
func (a *App) SetSize(width, height int) { a.w, a.h = width, height }

// textFocused reports whether a text widget has the keyboard.
func (a *App) textFocused() bool {
	t := a.cur()
	focused := a.side.Filter().Focused || t.url.Focused || t.find.Focused || t.timeout.Focused || t.body.Focused ||
		t.tests.Focused || t.gqlVars.Focused || t.file.Focused || t.msg.Focused || a.kv.Typing()
	for _, in := range t.auth {
		focused = focused || in.Focused
	}
	return focused
}
