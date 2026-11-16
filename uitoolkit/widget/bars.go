package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// bars is the scrollbar layout of a view in r: the area left for content
// after the bars, and each bar's track (ok false when not needed).
type bars struct {
	viewW, viewH int
	vTrack       core.Rect
	hTrack       core.Rect
	vOK, hOK     bool
}

func btoi(bool bool) int {
	if bool {
		return 1
	}
	return 0
}

// layoutBars decides which scrollbars a view in r needs for content of
// contentW by contentH. A bar takes room from the other direction, so
// the two decisions depend on each other.
func layoutBars(rect core.Rect, contentW, contentH int) bars {
	var b bars
	b.vOK = contentH > rect.H
	b.hOK = contentW > rect.W-btoi(b.vOK)*scrollbarW
	b.vOK = contentH > rect.H-btoi(b.hOK)*scrollbarW
	b.viewW, b.viewH = rect.W-btoi(b.vOK)*scrollbarW, rect.H-btoi(b.hOK)*scrollbarW
	b.vTrack = core.Rect{X: rect.X + rect.W - scrollbarW, Y: rect.Y, W: scrollbarW, H: b.viewH}
	b.hTrack = core.Rect{X: rect.X, Y: rect.Y + rect.H - scrollbarW, W: b.viewW, H: scrollbarW}
	return b
}

const (
	barNone = iota
	barV
	barH
)

// barDrag tracks a scrollbar thumb being dragged: which bar, and the grab.
type barDrag struct {
	bar  int // barNone, barV or barH
	grab grab
}

func (d *barDrag) active() bool { return d.bar != barNone }

// update runs a thumb drag, or starts one (or pages) on a press in a
// track, moving *sx and *sy.
func (d *barDrag) update(ctx *core.Ctx, sizes bars, contentW, contentH int, sx, sy *int) {
	in := ctx.In
	if d.bar != barNone {
		if !d.grab.held(in) {
			d.bar = barNone
			return
		}
		if d.bar == barV {
			*sy = posFromOffset(sizes.vTrack.H, contentH, sizes.viewH, in.Y-sizes.vTrack.Y-d.grab.off)
		} else {
			*sx = posFromOffset(sizes.hTrack.W, contentW, sizes.viewW, in.X-sizes.hTrack.X-d.grab.off)
		}
		return
	}
	if !in.Click {
		return
	}
	if sizes.vOK && sizes.vTrack.Contains(in.X, in.Y) {
		d.press(barV, sizes.vTrack.H, contentH, sizes.viewH, in.Y-sizes.vTrack.Y, sy)
	}
	if sizes.hOK && sizes.hTrack.Contains(in.X, in.Y) {
		d.press(barH, sizes.hTrack.W, contentW, sizes.viewW, in.X-sizes.hTrack.X, sx)
	}
}

// press grabs the thumb of bar when the press is on it, else pages toward
// the press.
func (d *barDrag) press(bar, track, content, view, at int, scroll *int) {
	grabbed, off, pos := pressThumb(track, content, view, *scroll, at)
	*scroll = pos
	if grabbed {
		d.bar = bar
		d.grab.begin(off)
	}
}

// drawBars draws the scrollbars of b for content scrolled to sx, sy.
func drawBars(ctx *core.Ctx, dst *ebiten.Image, sizes bars, barDrag barDrag, contentW, contentH, sx, sy int) {
	if sizes.vOK {
		drawScrollbar(ctx, dst, sizes.vTrack, contentH, sizes.viewH, sy, barDrag.bar == barV || ctx.Hovered(sizes.vTrack))
	}
	if sizes.hOK {
		drawScrollbar(ctx, dst, sizes.hTrack, contentW, sizes.viewW, sx, barDrag.bar == barH || ctx.Hovered(sizes.hTrack))
	}
}
