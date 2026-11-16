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

func (b *codeB) Update(element *El, ctx *core.Ctx) {
	b.view.Wrap = b.wrap
	b.view.Update(ctx, element.Rect())
}

func (b *codeB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	b.view.Wrap = b.wrap
	b.view.Draw(ctx, dst, element.Rect())
	if b.view.Focused {
		ctx.Outline(dst, element.Rect(), ctx.Color(core.AccentLine))
	}
}

// Code is a scrollable, foldable code viewer; give it a Bg.
func Code(code *CodeView, opts ...Option) *El {
	return Custom(&codeB{view: code}, with(nil, opts)...)
}

// wrapper is an element whose long lines can wrap: Code and Area.
type wrapper interface{ wrapLines() }

// Wrap makes a Code or Area break long lines before a word at its width
// instead of scrolling sideways. Without it the element does not wrap,
// whatever its state held, so a template switches wrap off by leaving it
// out (wrap="{{on}}").
func Wrap() Option { return func(element *El) { element.Behavior().(wrapper).wrapLines() } }
