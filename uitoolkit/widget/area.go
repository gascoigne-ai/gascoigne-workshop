package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// areaB is a multi-line editor over a TextArea.
type areaB struct {
	Base
	area *TextArea
}

func (b areaB) FocusTarget() Focuser { return b.area }

// KeepsTab: an area uses Tab to indent, so Escape is how focus leaves it.
func (areaB) KeepsTab() bool { return true }

func (b areaB) Update(e *El, c *core.Ctx) {
	if !b.area.Update(c, e.Rect()) {
		return
	}
	if onChange := e.ChangeHandler(); onChange != nil {
		onChange()
	}
}

func (b areaB) Draw(e *El, c *core.Ctx, dst *ebiten.Image) { b.area.Draw(c, dst, e.Rect()) }

// Area is a multi-line editor bound to a (v-model style). It keeps
// Tab for indenting, so Escape is how focus leaves it with the keyboard.
func Area(a *TextArea, opts ...Option) *El {
	return Custom(areaB{area: a}, with(nil, opts)...)
}
