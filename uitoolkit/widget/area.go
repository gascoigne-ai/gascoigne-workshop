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
	wrap bool // the Wrap option
}

func (b *areaB) FocusTarget() Focuser { return b.area }

// KeepsTab: an area uses Tab to indent, so Escape is how focus leaves it.
func (*areaB) KeepsTab() bool { return true }

func (b *areaB) wrapLines() { b.wrap = true }

func (b *areaB) Update(element *El, ctx *core.Ctx) {
	b.area.Wrap = b.wrap
	if !b.area.Update(ctx, element.Rect()) {
		return
	}
	element.Changed()
}

func (b *areaB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	b.area.Wrap = b.wrap
	b.area.Draw(ctx, dst, element.Rect())
}

// Area is a multi-line editor bound to a (v-model style). It keeps
// Tab for indenting, so Escape is how focus leaves it with the keyboard.
func Area(textArea *TextArea, opts ...Option) *El {
	return Custom(&areaB{area: textArea}, With(nil, opts)...)
}
