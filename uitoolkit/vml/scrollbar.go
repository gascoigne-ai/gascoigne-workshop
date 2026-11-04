package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

const (
	// ScrollbarW is the width of a scrollbar track in pixels.
	ScrollbarW  = 8
	minThumbLen = 24 // the thumb never gets shorter than this
)

// clampScroll keeps pos inside [0, contentLen-viewLen].
func clampScroll(pos, contentLen, viewLen int) int {
	return min(max(pos, 0), max(contentLen-viewLen, 0))
}

// Thumb returns the offset and length of a scrollbar thumb inside a
// track of trackLen pixels, for content of contentLen shown in a view of
// viewLen scrolled to pos. ok is false when everything fits.
func Thumb(trackLen, contentLen, viewLen, pos int) (off, length int, ok bool) {
	if trackLen <= 0 || contentLen <= viewLen {
		return 0, 0, false
	}
	length = min(max(trackLen*viewLen/contentLen, minThumbLen), trackLen)
	pos = clampScroll(pos, contentLen, viewLen)
	return (trackLen - length) * pos / (contentLen - viewLen), length, true
}

// posFromOffset is the scroll position that puts the thumb at off, so a
// dragged thumb can drive the content.
func posFromOffset(trackLen, contentLen, viewLen, off int) int {
	_, length, ok := Thumb(trackLen, contentLen, viewLen, 0)
	if !ok || trackLen-length <= 0 {
		return 0
	}
	maxOff := trackLen - length
	off = min(max(off, 0), maxOff)
	return off * (contentLen - viewLen) / maxOff
}

// Scrollbar draws the thumb in track for the given scroll state; the
// track is vertical or horizontal by its shape. active brightens it
// (hover or drag). It draws nothing when everything fits.
func drawScrollbar(c *core.Ctx, dst *ebiten.Image, track core.Rect, contentLen, viewLen, pos int, active bool) {
	vertical := track.H >= track.W
	trackLen := track.W
	if vertical {
		trackLen = track.H
	}
	off, length, ok := Thumb(trackLen, contentLen, viewLen, pos)
	if !ok {
		return
	}
	clr := c.Color(core.Muted)
	if active {
		clr = c.Color(core.Text)
	}
	thumb := core.Rect{X: track.X + 1, Y: track.Y + off, W: track.W - 2, H: length}
	if !vertical {
		thumb = core.Rect{X: track.X + off, Y: track.Y + 1, W: length, H: track.H - 2}
	}
	c.FillStyled(dst, thumb, clr, clr, c.Radius("scrollbar"))
}

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
	b.hOK = contentW > r.W-btoi(b.vOK)*ScrollbarW
	b.vOK = contentH > r.H-btoi(b.hOK)*ScrollbarW
	b.viewW, b.viewH = r.W-btoi(b.vOK)*ScrollbarW, r.H-btoi(b.hOK)*ScrollbarW
	b.vTrack = core.Rect{X: r.X + r.W - ScrollbarW, Y: r.Y, W: ScrollbarW, H: b.viewH}
	b.hTrack = core.Rect{X: r.X, Y: r.Y + r.H - ScrollbarW, W: b.viewW, H: ScrollbarW}
	return b
}

const (
	barNone = iota
	barV
	barH
)

// barDrag tracks a scrollbar thumb being dragged.
type barDrag struct {
	bar  int // barNone, barV or barH
	grab int // pointer offset inside the thumb when the drag began
}

func (d *barDrag) active() bool { return d.bar != barNone }

// update runs a thumb drag, or starts one (or pages) on a press in a
// track, moving *sx and *sy.
func (d *barDrag) update(c *core.Ctx, b bars, contentW, contentH int, sx, sy *int) {
	in := c.In
	switch d.bar {
	case barV:
		if !in.Held {
			d.bar = barNone
			return
		}
		*sy = posFromOffset(b.vTrack.H, contentH, b.viewH, in.Y-b.vTrack.Y-d.grab)
		return
	case barH:
		if !in.Held {
			d.bar = barNone
			return
		}
		*sx = posFromOffset(b.hTrack.W, contentW, b.viewW, in.X-b.hTrack.X-d.grab)
		return
	}
	if !in.Click {
		return
	}
	if b.vOK && b.vTrack.Contains(in.X, in.Y) {
		d.press(b.vTrack.H, contentH, b.viewH, *sy, in.Y-b.vTrack.Y, barV, sy)
	}
	if b.hOK && b.hTrack.Contains(in.X, in.Y) {
		d.press(b.hTrack.W, contentW, b.viewW, *sx, in.X-b.hTrack.X, barH, sx)
	}
}

// press starts dragging the thumb when the press is on it, else pages
// toward the press.
func (d *barDrag) press(track, content, view, pos, at, bar int, scroll *int) {
	off, length, ok := Thumb(track, content, view, pos)
	switch {
	case !ok:
	case at >= off && at < off+length:
		d.bar, d.grab = bar, at-off
	case at < off:
		*scroll = clampScroll(pos-view, content, view)
	default:
		*scroll = clampScroll(pos+view, content, view)
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
