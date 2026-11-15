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
	wrap bool // the Wrap option
}

func (b *codeB) FocusTarget() Focuser { return b.view }
func (*codeB) KeepsTab() bool         { return false }

func (b *codeB) wrapLines() { b.wrap = true }

func (b *codeB) Update(e *El, c *core.Ctx) {
	b.view.Wrap = b.wrap
	b.view.Update(c, e.Rect())
}

func (b *codeB) Draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	b.view.Wrap = b.wrap
	b.view.Draw(c, dst, e.Rect())
	if b.view.Focused {
		c.Outline(dst, e.Rect(), c.Color(core.AccentLine))
	}
}

// Code is a scrollable, foldable code viewer; give it a Bg.
func Code(v *CodeView, opts ...Option) *El {
	return Custom(&codeB{view: v}, with(nil, opts)...)
}

// wrapper is an element whose long lines can wrap.
type wrapper interface{ wrapLines() }

// Wrap makes a Code break long lines before a word at its width
// instead of scrolling sideways. Without it the element does not wrap,
// whatever its state held, so a template switches wrap off by leaving it
// out (wrap="{{on}}").
func Wrap() Option { return func(e *El) { e.Behavior().(wrapper).wrapLines() } }
