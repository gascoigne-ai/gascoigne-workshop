package widget

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

// Scrollbar draws the thumb in track for the given scroll state; the
// track is vertical or horizontal by its shape. active brightens it
// (hover or drag). It draws nothing when everything fits.
func (c *Ctx) Scrollbar(dst *ebiten.Image, track Rect, contentLen, viewLen, pos int, active bool) {
	vertical := track.H >= track.W
	trackLen := track.W
	if vertical {
		trackLen = track.H
	}
	off, length, ok := Thumb(trackLen, contentLen, viewLen, pos)
	if !ok {
		return
	}
	clr := c.Col(Muted)
	if active {
		clr = c.Col(Text)
	}
	thumb := Rect{X: track.X + 1, Y: track.Y + off, W: track.W - 2, H: length}
	if !vertical {
		thumb = Rect{X: track.X + off, Y: track.Y + 1, W: length, H: track.H - 2}
	}
	c.Fill(dst, thumb, clr)
}
