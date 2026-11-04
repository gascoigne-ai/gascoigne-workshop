package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// areaB is a multi-line editor over a core.TextArea.
type areaB struct{ plain }

func (areaB) measure(*El, *core.Ctx, int) (int, int) { return 0, 0 }

func (areaB) update(e *El, c *core.Ctx) {
	if e.area.Update(c, e.rect) && e.onChange != nil {
		e.onChange()
	}
}

func (areaB) draw(e *El, c *core.Ctx, dst *ebiten.Image) { e.area.Draw(c, dst, e.rect) }

// TextArea is a multi-line editor bound to a (v-model style). It keeps
// Tab for indenting, so Escape is how focus leaves it with the keyboard.
func TextArea(a *core.TextArea, opts ...Option) *El {
	e := newEl(areaB{}, with(nil, opts))
	e.area = a
	return e
}
