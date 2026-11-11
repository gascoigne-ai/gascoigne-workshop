package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// codeB shows a CodeView.
type codeB struct {
	Base
	view *CodeView
}

func (b codeB) FocusTarget() Focuser { return b.view }
func (codeB) KeepsTab() bool         { return false }

func (b codeB) Update(e *El, c *core.Ctx) { b.view.Update(c, e.Rect()) }

func (b codeB) Draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	b.view.Draw(c, dst, e.Rect())
	if b.view.Focused {
		c.Outline(dst, e.Rect(), c.Color(core.AccentLine))
	}
}

// Code is a scrollable, foldable code viewer; give it a Bg.
func Code(v *CodeView, opts ...Option) *El {
	return Custom(codeB{view: v}, with(nil, opts)...)
}
