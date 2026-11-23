package window

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Step runs one frame with the given input: it applies finished sends,
// builds the view, lays it out and lets it react. Handlers in the view
// change the window's state. Tests drive the window through Step without a
// GPU; the app calls it once per Update.
func (a *Component) Step(in core.Input) {
	moved := a.beginFrame(in)
	a.Keys(in)
	a.endFrame(moved)
}

// beginFrame takes in the input and what goroutines reported, and reports
// whether anything moved: input, a finished send or an expired toast.
func (a *Component) beginFrame(in core.Input) bool {
	a.ctx.Tick++
	a.ctx.In = in
	a.ctx.Cursor = core.CursorDefault
	a.strip.BeginFrame(in)
	a.editor.Tables.BeginFrame(in)
	drained := a.drain()
	pruned := a.over.Toasts.Prune()
	active := engine.InputActive(a.lastIn, in)
	a.keys.NoteGeometry([4]int{a.w, a.h, a.winX, a.winY})
	if active || drained || pruned {
		a.keys.MarkStale()
	}
	if in.RestartsCaret() {
		a.ctx.RestartCaret()
	}
	a.lastIn = in
	a.side.BeginFrame(in)
	return active || drained || pruned
}

// endFrame builds the view once, lets it react and runs the per-frame
// upkeep; moved is what beginFrame reported. Draw paints the tree built here
// unless treeStale says that a handler or a step below changed what it shows.
func (a *Component) endFrame(moved bool) {
	root := a.buildTree()
	view.Layout(a.ctx, root, a.w, a.h)
	fired := a.ctx.Fired
	view.Update(a.ctx, root)
	a.drawTree, a.treeStale = root, a.ctx.Fired != fired
	// The steps below run after the tree was built. Each one that changes
	// what the view shows reports it (the Tick of a child, dragHover) or sets
	// a.treeStale itself (commitEdit, saveCollection, a toast); the others
	// only read state or touch the cursor, the clock or a write queue.
	a.treeStale = a.editor.Tick() || a.treeStale // a click elsewhere ends a cell edit, leaving the table applies its text
	a.treeStale = a.resp.Tick() || a.treeStale
	a.treeStale = a.over.Tick() || a.treeStale // an outside change of settings.json replaces settings and the raw text
	a.dragHover()
	a.showGrab()
	a.stateFile.Tick(a.keys.State(), a.now())
	a.workspaceTick()
	busy := moved || a.tips.Waiting()
	a.markFrame(busy)
	a.pacer.NoteIdle(busy)
}

// Draw paints the tree of the last step, built again only when something
// changed after that tree was built.
func (a *Component) Draw(screen *ebiten.Image) {
	if !a.dirty && !a.alwaysDraw() {
		return // nothing changed: the screen keeps what it has and the GPU stays idle
	}
	view.Draw(a.ctx, a.frameTree(), screen)
	a.dirty = false
}

// buildTree builds the window's view; the count is for tests.
func (a *Component) buildTree() *view.El {
	a.builds++
	return a.view()
}

// frameTree is the laid out tree Draw paints: the one endFrame built when
// nothing it shows changed since, else a fresh build. A capture
// (Start.DrawEveryFrame) changes the scene after the step, so it always
// builds fresh. A revision
// counter bumped by every change was rejected: one missed bump loses an
// edit on screen; a test that compares the reused tree with a fresh build
// catches a missed treeStale instead
// (TestTheDrawnTreeMatchesAFreshBuildAcrossASession).
func (a *Component) frameTree() *view.El {
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
func (a *Component) Tree() view.Node { return a.frameTree() }

// SetSize sets the window size in logical pixels and nothing else. Unlike
// Resize it leaves the device scale alone, which is what a test window of
// fixed size wants; apptest calls it before the first Step.
func (a *Component) SetSize(width, height int) { a.w, a.h = width, height }

// Resize is what the frame loop's Layout reports: the logical size, the
// device scale and the window units per logical pixel, for remembered sizes.
func (a *Component) Resize(width, height int, scale, windowScale float64) {
	a.windowScale = windowScale
	a.setSize(width, height, scale)
}

// SetPosition records where the window is, read once a frame.
func (a *Component) SetPosition(posX, posY int) { a.winX, a.winY = posX, posY }

// UIScale is the user's interface scale; the app multiplies it by the
// platform's base factor.
func (a *Component) UIScale() float64 { return a.settings.UIScale() }

// Scale is the device pixels per logical pixel of the last Resize.
func (a *Component) Scale() float64 { return a.ctx.Scale }

// Cursor is the cursor the widgets of the last frame asked for.
func (a *Component) Cursor() core.Cursor { return a.ctx.Cursor }

// HostOS is the OS the shortcut text follows.
func (a *Component) HostOS() string { return a.ctx.HostOS }

// TPS is the update rate the pacer wants now.
func (a *Component) TPS() int { return a.pacer.TPS() }

// Now is the time of the window's clock.
func (a *Component) Now() time.Time { return a.now() }

// Err is the error that ended the run, if any.
func (a *Component) Err() error { return a.err }

// WaitWrites blocks until the background writes are done, for the paths that
// must not lose data. It is exported for apptest, whose tests wait for saves
// before they read files.
func (a *Component) WaitWrites() { a.writes.Wait() }

// RestoredWindow is the window geometry of the last run, clamped to a
// monitor of mw by mh; the zero Window when nothing was remembered.
func (a *Component) RestoredWindow(mw, mh int) state.Window { return a.stateFile.Restored(mw, mh) }

// SaveState writes the state now; main calls it on exit.
func (a *Component) SaveState() error { return a.stateFile.Save() }
