package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// tipDelayTicks is how long the pointer rests on an element before its
// tooltip shows (half a second at 60 ticks per second).
const tipDelayTicks = 30

// Tooltip box metrics.
const (
	tipPadX   = 8
	tipPadY   = 5
	tipOffset = 16 // below and right of the pointer, clear of the cursor
)

// TooltipState is the delay bookkeeping of the tooltip layer. The window
// owns it, so it can keep frames coming while a tooltip waits to show.
type TooltipState struct {
	shown, waiting bool
	text           string // the text the delay is counted for
	since          int    // tick at which the pointer came to rest on it
}

// Shown reports whether the last frame drew a tooltip.
func (s *TooltipState) Shown() bool { return s.shown }

// Waiting reports whether the pointer rests on an element with a tooltip
// that is not due yet: the window must draw again to show it.
func (s *TooltipState) Waiting() bool { return s.waiting }

// TooltipLayer draws the tooltip of the element under the pointer (set
// with view.Tooltip) over everything drawn before it, kept inside the
// window. Mount it last in the window's Stack.
func TooltipLayer(state *TooltipState) *El { return Custom(tooltipB{state: state}) }

type tooltipB struct {
	Base
	state *TooltipState
}

func (b tooltipB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	b.state.draw(ctx, dst, element.Rect())
}

// draw updates the delay bookkeeping for the tip of the element under the
// pointer and, once it is due, draws it inside screen.
func (s *TooltipState) draw(ctx *core.Ctx, dst *ebiten.Image, screen core.Rect) {
	s.shown, s.waiting = false, false
	if ctx.TipText == "" {
		s.text = ""
		return
	}
	if ctx.TipText != s.text || ctx.In.Click {
		s.text, s.since = ctx.TipText, ctx.Tick // a new element, or a click: wait again
	}
	if ctx.Tick-s.since < tipDelayTicks {
		s.waiting = true
		return
	}
	s.shown = true
	w := ctx.Width(ctx.F.Small, ctx.TipText) + 2*tipPadX
	h := ctx.LineH(ctx.F.Small) + 2*tipPadY
	x := max(screen.X, min(ctx.In.X+tipOffset, screen.X+screen.W-w))
	y := ctx.In.Y + tipOffset
	if y+h > screen.Y+screen.H {
		y = ctx.In.Y - tipOffset - h // no room below: above the pointer
	}
	box := core.Rect{X: x, Y: max(screen.Y, y), W: w, H: h}
	st := ctx.Th.Style("tooltip")
	ctx.FillStyled(dst, box, ctx.Color(core.Frame), ctx.Color(core.Frame), st.Radius)
	border := st.Border
	if border == "" {
		border = core.Border
	}
	ctx.OutlineStyled(dst, box, ctx.Color(border), st.Radius)
	ctx.TextMid(dst, ctx.F.Small, ctx.TipText, box.X+tipPadX, box, ctx.Color(core.Text))
}
