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

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// layoutBars decides which scrollbars a view in r needs for content of
// contentW by contentH. A bar takes room from the other direction, so
// the two decisions depend on each other.
func layoutBars(r core.Rect, contentW, contentH int) bars {
	var b bars
	b.vOK = contentH > r.H
	b.hOK = contentW > r.W-btoi(b.vOK)*scrollbarW
	b.vOK = contentH > r.H-btoi(b.hOK)*scrollbarW
	b.viewW, b.viewH = r.W-btoi(b.vOK)*scrollbarW, r.H-btoi(b.hOK)*scrollbarW
	b.vTrack = core.Rect{X: r.X + r.W - scrollbarW, Y: r.Y, W: scrollbarW, H: b.viewH}
	b.hTrack = core.Rect{X: r.X, Y: r.Y + r.H - scrollbarW, W: b.viewW, H: scrollbarW}
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
func (d *barDrag) update(c *core.Ctx, b bars, contentW, contentH int, sx, sy *int) {
	in := c.In
	if d.bar != barNone {
		if !d.grab.held(in) {
			d.bar = barNone
			return
		}
		if d.bar == barV {
			*sy = posFromOffset(b.vTrack.H, contentH, b.viewH, in.Y-b.vTrack.Y-d.grab.off)
		} else {
			*sx = posFromOffset(b.hTrack.W, contentW, b.viewW, in.X-b.hTrack.X-d.grab.off)
		}
		return
	}
	if !in.Click {
		return
	}
	if b.vOK && b.vTrack.Contains(in.X, in.Y) {
		d.press(barV, b.vTrack.H, contentH, b.viewH, in.Y-b.vTrack.Y, sy)
	}
	if b.hOK && b.hTrack.Contains(in.X, in.Y) {
		d.press(barH, b.hTrack.W, contentW, b.viewW, in.X-b.hTrack.X, sx)
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
func drawBars(c *core.Ctx, dst *ebiten.Image, b bars, d barDrag, contentW, contentH, sx, sy int) {
	if b.vOK {
		drawScrollbar(c, dst, b.vTrack, contentH, b.viewH, sy, d.bar == barV || c.Hovered(b.vTrack))
	}
	if b.hOK {
		drawScrollbar(c, dst, b.hTrack, contentW, b.viewW, sx, d.bar == barH || c.Hovered(b.hTrack))
	}
}
