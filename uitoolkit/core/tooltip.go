package core

import "github.com/hajimehoshi/ebiten/v2"

// TipDelayTicks is how long the pointer rests on an element before its
// tooltip shows (half a second at 60 ticks per second).
const TipDelayTicks = 30

// Tooltip box metrics.
const (
	tipPadX   = 8
	tipPadY   = 5
	tipOffset = 16 // below and right of the pointer, clear of the cursor
)

// DrawTooltip updates the delay bookkeeping for the element under the
// pointer (TipText, set by the views while they draw) and, once it is
// due, draws its tooltip over everything, kept inside screen.
func (c *Ctx) DrawTooltip(dst *ebiten.Image, screen Rect) {
	c.TipShown, c.TipWaiting = false, false
	if c.TipText == "" {
		c.TipFor = ""
		return
	}
	if c.TipText != c.TipFor || c.In.Click {
		c.TipFor, c.TipSince = c.TipText, c.Tick // a new element, or a click: wait again
	}
	if c.Tick-c.TipSince < TipDelayTicks {
		c.TipWaiting = true
		return
	}
	c.TipShown = true
	w := c.Width(c.F.Small, c.TipText) + 2*tipPadX
	h := c.LineH(c.F.Small) + 2*tipPadY
	x := max(screen.X, min(c.In.X+tipOffset, screen.X+screen.W-w))
	y := c.In.Y + tipOffset
	if y+h > screen.Y+screen.H {
		y = c.In.Y - tipOffset - h // no room below: above the pointer
	}
	box := Rect{X: x, Y: max(screen.Y, y), W: w, H: h}
	st := c.Th.Style("tooltip")
	c.FillStyled(dst, box, c.Color(Frame), c.Color(Frame), st.Radius)
	border := st.Border
	if border == "" {
		border = Border
	}
	c.OutlineStyled(dst, box, c.Color(border), st.Radius)
	c.TextMid(dst, c.F.Small, c.TipText, box.X+tipPadX, box, c.Color(Text))
}
