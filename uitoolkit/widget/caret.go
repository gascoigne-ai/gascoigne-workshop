package widget

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// caretGlyph is the character the block caret covers: the rune at cursor,
// or a space at the end of the text and before a line break or tab, so the
// block still has a cell to stand on.
func caretGlyph(runes []rune, cursor int) string {
	if cursor < 0 || cursor >= len(runes) || runes[cursor] == '\n' || runes[cursor] == '\t' {
		return " "
	}
	return string(runes[cursor])
}

// caretBlock is the block's rectangle: w wide at x, one text line high and
// centered in row (or the row's own height when it is shorter than a line).
func caretBlock(row core.Rect, posX, width, lineH int) core.Rect {
	height := min(lineH, row.H)
	return core.Rect{X: posX, Y: row.Y + (row.H-height)/2, W: width, H: height}
}

// drawBlockCaret paints a solid accent-colored block over the character at x and redraws
// that character in the accent text color so it stays readable. row is the
// line the text is centered in.
func drawBlockCaret(ctx *core.Ctx, dst *ebiten.Image, face *text.GoTextFace, glyph string, posX int, row core.Rect) {
	ctx.Fill(dst, caretBlock(row, posX, ctx.Width(face, glyph), ctx.LineH(face)), ctx.Color(core.Accent))
	ctx.TextMid(dst, face, glyph, posX, row, ctx.Color(core.AccentText))
}
