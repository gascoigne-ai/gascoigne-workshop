package widget

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// paragraphB is a muted small text that breaks into lines at the width its
// parent gives it, so a description sits beside a control without a
// column count guessed in advance.
type paragraphB struct {
	Base
	content string
}

// wrapParagraph breaks content before the word that would pass width. A word
// wider than width stands alone on its line.
func wrapParagraph(ctx *core.Ctx, face *text.GoTextFace, content string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(content) {
		joined := word
		if line != "" {
			joined = line + " " + word
		}
		if line != "" && ctx.Width(face, joined) > width {
			lines = append(lines, line)
			joined = word
		}
		line = joined
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// Describe puts the whole text in a Dump, where a test reads what a row says.
func (p paragraphB) Describe() string { return fmt.Sprintf("text=%q", p.content) }

func (p paragraphB) Measure(element *El, ctx *core.Ctx, maxW int) (int, int) {
	face := ctx.F.Small
	lines := wrapParagraph(ctx, face, p.content, max(maxW, 1))
	widest := 0
	for _, line := range lines {
		widest = max(widest, ctx.Width(face, line))
	}
	return min(widest, max(maxW, 1)), len(lines) * ctx.LineH(face)
}

func (p paragraphB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	face := ctx.F.Small
	rect := element.Rect()
	clr := ctx.Color(core.Muted)
	for i, line := range wrapParagraph(ctx, face, p.content, rect.W) {
		ctx.Text(dst, face, line, rect.X, rect.Y+i*ctx.LineH(face), clr)
	}
}

// Paragraph is muted small text that wraps at the width it is laid out in.
func Paragraph(content string, opts ...Option) *El {
	return Custom(paragraphB{content: content}, With(nil, opts)...)
}
