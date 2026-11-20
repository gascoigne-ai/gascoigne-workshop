package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// checkB is the checkbox: a 14px box, an accent square with a check when
// on, a plain outline when off.
type checkB struct {
	Base
	on bool
}

func (checkB) Measure(*El, *core.Ctx, int) (int, int) { return core.CheckboxSize, core.CheckboxSize }

// Update runs OnChange when the box is clicked; the caller owns the value.
func (checkB) Update(element *El, ctx *core.Ctx) {
	if ctx.In.Hit(element.Rect()) {
		element.Changed()
	}
}

func (b checkB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	r := element.Rect()
	drawCheckbox(ctx, dst, core.Rect{X: r.X + (r.W-core.CheckboxSize)/2, Y: r.Y + (r.H-core.CheckboxSize)/2, W: core.CheckboxSize, H: core.CheckboxSize}, b.on, ctx.Hovered(r))
}

// drawCheckbox draws a 14px box: an accent square with a white check
// when on, a plain outline when off (darker under the pointer).
func drawCheckbox(ctx *core.Ctx, dst *ebiten.Image, box core.Rect, on, hover bool) {
	if !on {
		edge := core.Rule
		if hover {
			edge = core.Muted
		}
		ctx.OutlineStyled(dst, box, ctx.Color(edge), ctx.Radius("checkbox"))
		return
	}
	accent := ctx.Color(core.Accent)
	ctx.FillStyled(dst, box, accent, accent, ctx.Radius("checkbox"))
	drawIconAt(ctx, dst, "check", box, core.AccentText)
}

// Checkbox is a square that shows an accent fill and a check when on.
// OnChange runs when it is clicked; the caller flips the value it passes in.
func Checkbox(on bool, opts ...Option) *El {
	return Custom(checkB{on: on}, With(nil, opts)...)
}
