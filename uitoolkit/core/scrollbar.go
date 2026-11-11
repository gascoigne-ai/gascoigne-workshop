package core

import "github.com/hajimehoshi/ebiten/v2"

const (
	// ScrollbarW is the width of a scrollbar track in pixels.
	ScrollbarW  = 8
	minThumbLen = 24 // the thumb never gets shorter than this
)

// ClampScroll keeps pos inside [0, contentLen-viewLen].
func ClampScroll(pos, contentLen, viewLen int) int {
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
	pos = ClampScroll(pos, contentLen, viewLen)
	return (trackLen - length) * pos / (contentLen - viewLen), length, true
}

// PosFromOffset is the scroll position that puts the thumb at off, so a
// dragged thumb can drive the content.
func PosFromOffset(trackLen, contentLen, viewLen, off int) int {
	_, length, ok := Thumb(trackLen, contentLen, viewLen, 0)
	if !ok || trackLen-length <= 0 {
		return 0
	}
	maxOff := trackLen - length
	off = min(max(off, 0), maxOff)
	return off * (contentLen - viewLen) / maxOff
}

// DrawScrollbar draws the thumb in track for the given scroll state; the
// track is vertical or horizontal by its shape. active brightens it
// (hover or drag). It draws nothing when everything fits.
func (c *Ctx) DrawScrollbar(dst *ebiten.Image, track Rect, contentLen, viewLen, pos int, active bool) {
	vertical := track.H >= track.W
	trackLen := track.W
	if vertical {
		trackLen = track.H
	}
	off, length, ok := Thumb(trackLen, contentLen, viewLen, pos)
	if !ok {
		return
	}
	clr := c.Color(Muted)
	if active {
		clr = c.Color(Text)
	}
	thumb := Rect{X: track.X + 1, Y: track.Y + off, W: track.W - 2, H: length}
	if !vertical {
		thumb = Rect{X: track.X + off, Y: track.Y + 1, W: length, H: track.H - 2}
	}
	c.FillStyled(dst, thumb, clr, clr, c.Radius("scrollbar"))
}

// PressThumb is a press at `at` in a scrollbar track for content shown
// through a view scrolled to pos: it either grabs the thumb (grabbed, with
// the offset inside it) or pages toward the press (the new pos).
func PressThumb(track, content, view, pos, at int) (grabbed bool, off, newPos int) {
	thumb, length, ok := Thumb(track, content, view, pos)
	switch {
	case !ok:
		return false, 0, pos
	case at >= thumb && at < thumb+length:
		return true, at - thumb, pos
	case at < thumb:
		return false, 0, ClampScroll(pos-view, content, view)
	}
	return false, 0, ClampScroll(pos+view, content, view)
}
