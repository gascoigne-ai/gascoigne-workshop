package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Scrollbar geometry shared by Scroll and HScroll: the track width and the
// thumb's size and position for the content shown.
const (
	// scrollbarW is the width of a scrollbar track in pixels.
	scrollbarW  = 8
	minThumbLen = 24 // the thumb never gets shorter than this
)

// clampScroll keeps pos inside [0, contentLen-viewLen].
func clampScroll(pos, contentLen, viewLen int) int {
	return min(max(pos, 0), max(contentLen-viewLen, 0))
}

// thumb returns the offset and length of a scrollbar thumb inside a
// track of trackLen pixels, for content of contentLen shown in a view of
// viewLen scrolled to pos. ok is false when everything fits.
func thumb(trackLen, contentLen, viewLen, pos int) (off, length int, ok bool) {
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
	_, length, ok := thumb(trackLen, contentLen, viewLen, 0)
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
	off, length, ok := thumb(trackLen, contentLen, viewLen, pos)
	if !ok {
		return
	}
	clr := c.Color(core.Muted)
	if active {
		clr = c.Color(core.Text)
	}
	bar := core.Rect{X: track.X + 1, Y: track.Y + off, W: track.W - 2, H: length}
	if !vertical {
		bar = core.Rect{X: track.X + off, Y: track.Y + 1, W: length, H: track.H - 2}
	}
	c.FillStyled(dst, bar, clr, clr, c.Radius("scrollbar"))
}

// pressThumb is a press at `at` in a scrollbar track for content shown
// through a view scrolled to pos: it either grabs the thumb (grabbed, with
// the offset inside it) or pages toward the press (the new pos).
func pressThumb(track, content, view, pos, at int) (grabbed bool, off, newPos int) {
	start, length, ok := thumb(track, content, view, pos)
	switch {
	case !ok:
		return false, 0, pos
	case at >= start && at < start+length:
		return true, at - start, pos
	case at < start:
		return false, 0, clampScroll(pos-view, content, view)
	}
	return false, 0, clampScroll(pos+view, content, view)
}
