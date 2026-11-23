package window

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// baseSnapshot is the window under a modal overlay (the base column and the
// dim scrim, never the overlay itself) drawn once into an image of its own.
// While the overlay is open a frame that changed nothing in the base blits
// the image and draws only what lies above it, so dragging the dialog or
// the blink of its caret does not paint the whole window again. Each frame
// still builds the tree, lays it out and runs Update: only drawing is
// skipped.
//
// The base is not drawn again while nothing can have changed it; see
// changesBase for what can. A change is noticed when it happens (Step) and
// the image redrawn at the next Draw, so several Updates per Draw lose
// nothing.
//
// Why this and not the alternatives: a damage rectangle on the persistent
// screen (SetScreenClearedEveryFrame(false) already keeps it) still walks
// the whole base, needs damage tracking per element, and the scrim's alpha
// needs the base under the old and the new dialog rectangle; a cheaper
// full redraw keeps the cost growing with the size of the base; a
// fingerprint of the base tree cannot see the state of a custom Behavior.
// Revisit when build and layout dominate the drag frame, when the memory
// at 5K matters (4 bytes per device pixel, 59 MB) or when a non-modal
// overlay wants the same treatment.
//
// Invariant: image is nil exactly when phase is snapNone.
type baseSnapshot struct {
	phase    snapshotPhase
	image    *ebiten.Image
	geometry snapshotGeometry // what the image was drawn for
	paints   int              // how often the base was drawn into the image, for tests
}

// snapshotPhase is whether the image holds the base as the next Draw would
// draw it. Draw is the only way back to fresh, so a change noticed by any
// of several Steps since the last Draw keeps the image stale.
type snapshotPhase int

const (
	snapNone  snapshotPhase = iota // no image: no modal overlay was drawn
	snapFresh                      // the base cannot have changed since it was drawn
	snapStale                      // something in the base may have changed
)

// caretUse is what the base showed of the blinking caret when it was drawn.
type caretUse int

const (
	caretUnused caretUse = iota // no field of the base read the blink phase
	caretShown
	caretHidden
)

// snapshotGeometry is what the image was drawn for besides the tree: a
// difference redraws or reallocates it. size comes from the screen's bounds
// (not width times scale), so a fractional scale cannot mismatch.
type snapshotGeometry struct {
	size  image.Point
	scale float64
	layer int
	caret caretUse
}

// snapshotAction is what Draw does with the image.
type snapshotAction int

const (
	snapReuse      snapshotAction = iota // blit it as it is
	snapRedraw                           // clear it and draw the base again
	snapReallocate                       // make an image of the new size, then draw
	snapRelease                          // no modal overlay: free the image
)

// snapshotStep is the transition table of the snapshot. current is the
// geometry of this Draw, layer the view.ModalLayer of its tree.
func snapshotStep(phase snapshotPhase, kept, current snapshotGeometry, layer int) snapshotAction {
	switch {
	case layer < 0:
		return snapRelease
	case phase == snapNone, kept.size != current.size:
		return snapReallocate
	case phase == snapStale, kept.scale != current.scale, kept.layer != current.layer:
		return snapRedraw
	case kept.caret != caretUnused && kept.caret != current.caret:
		return snapRedraw // the base shows the blink phase and it flipped
	}
	return snapReuse
}

// frameSignals is what one Step noticed that can change the base under a
// modal overlay. The rest of the pointer (its place, the held buttons) cannot:
// hover is off under a modal.
type frameSignals struct {
	drained    bool // a message from a goroutine was applied
	pruned     bool // a toast expired
	keys       bool // input beyond the pointer: a click, text, the wheel, a key (see inputEvents)
	fired      bool // a view handler ran in Update
	stale      bool // a step after the build changed what the view shows
	everyFrame bool // a capture draws every frame

	// active is not a signal for the base: the pointer moved or a button
	// changed, which only wakes the pacer.
	active bool
}

// changesBase reports whether the base must be drawn again.
func changesBase(signals frameSignals) bool {
	return signals.drained || signals.pruned || signals.keys || signals.fired || signals.stale || signals.everyFrame
}

// moved reports whether the frame had activity the pacer must not sleep through.
func (signals frameSignals) moved() bool { return signals.active || signals.drained || signals.pruned }

// inputEvents reports whether in carries more than the pointer's place and
// held state. core.Input holds slices, so it cannot be compared with its zero
// value: the events are listed, and a test pins the field count so a new
// field cannot be forgotten.
func inputEvents(in core.Input) bool {
	return in.Click || in.MiddleClick || in.RightClick || in.Wheel != 0 || in.WheelX != 0 ||
		len(in.Typed) > 0 || len(in.Edits) > 0 || in.Enter || in.Escape || in.Tab ||
		in.SelectAll || in.Copy || in.Cut || in.Paste || in.Undo || in.Redo || len(in.Keys) > 0
}

// mark notes that the base may have changed; a fresh image turns stale.
func (snap *baseSnapshot) mark() {
	if snap.phase == snapFresh {
		snap.phase = snapStale
	}
}

// release frees the image.
func (snap *baseSnapshot) release() {
	if snap.image != nil {
		snap.image.Deallocate()
	}
	*snap = baseSnapshot{paints: snap.paints}
}

// caretNow is the blink phase a base that shows it would be drawn with.
func caretNow(ctx *core.Ctx) caretUse {
	if ctx.CaretVisible() {
		return caretShown
	}
	return caretHidden
}

// draw paints root into screen: the base from the image (drawn first when
// it is missing or stale) and the layers above it into screen.
func (snap *baseSnapshot) draw(ctx *core.Ctx, root view.Node, screen *ebiten.Image) {
	layer := view.ModalLayer(ctx, root)
	current := snapshotGeometry{size: screen.Bounds().Size(), scale: ctx.Scale, layer: layer}
	if snap.geometry.caret != caretUnused {
		current.caret = caretNow(ctx)
	}
	switch snapshotStep(snap.phase, snap.geometry, current, layer) {
	case snapRelease:
		snap.release()
		view.Draw(ctx, root, screen)
		return
	case snapReallocate:
		if snap.image != nil {
			snap.image.Deallocate()
		}
		snap.image = ebiten.NewImage(current.size.X, current.size.Y)
		snap.paint(ctx, root, current)
	case snapRedraw:
		snap.paint(ctx, root, current)
	}
	screen.DrawImage(snap.image, nil)
	view.DrawFrom(ctx, root, layer, screen)
}

// paint draws the base into a cleared image and records what it showed.
func (snap *baseSnapshot) paint(ctx *core.Ctx, root view.Node, geometry snapshotGeometry) {
	snap.image.Clear()
	reads := ctx.CaretReads
	view.DrawBelow(ctx, root, geometry.layer, snap.image)
	geometry.caret = caretUnused
	if ctx.CaretReads != reads {
		geometry.caret = caretNow(ctx)
	}
	snap.geometry, snap.phase = geometry, snapFresh
	snap.paints++
}
