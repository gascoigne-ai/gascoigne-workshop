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

func (checkB) Measure(*El, *core.Ctx, int) (int, int) { return 14, 14 }

// update runs OnChange when the box is clicked; the caller owns the value.
func (checkB) Update(e *El, c *core.Ctx) {
	if onChange := e.ChangeHandler(); onChange != nil && c.In.Hit(e.Rect()) {
		onChange()
	}
}

func (b checkB) Draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	r := e.Rect()
	drawCheckbox(c, dst, core.Rect{X: r.X + (r.W-14)/2, Y: r.Y + (r.H-14)/2, W: 14, H: 14}, b.on, c.Hovered(r))
}

// drawCheckbox draws a 14px box: an accent square with a white check
// when on, a plain outline when off (darker under the pointer).
func drawCheckbox(c *core.Ctx, dst *ebiten.Image, box core.Rect, on, hover bool) {
	if !on {
		edge := core.Rule
		if hover {
			edge = core.Muted
		}
		c.OutlineStyled(dst, box, c.Color(edge), c.Radius("checkbox"))
		return
	}
	accent := c.Color(core.Accent)
	c.FillStyled(dst, box, accent, accent, c.Radius("checkbox"))
	x, y := float32(box.X), float32(box.Y)
	mark := c.Color(core.AccentText)
	c.Line(dst, x+3.5, y+7.5, x+6, y+10, 1.8, mark)
	c.Line(dst, x+6, y+10, x+10.5, y+4.5, 1.8, mark)
}

// Checkbox is a square that shows an accent fill and a check when on.
// OnChange runs when it is clicked; the caller flips the value it passes in.
func Checkbox(on bool, opts ...Option) *El {
	return Custom(checkB{on: on}, with(nil, opts)...)
}
