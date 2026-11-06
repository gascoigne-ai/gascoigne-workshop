package vml

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
func caretBlock(row core.Rect, x, w, lineH int) core.Rect {
	height := min(lineH, row.H)
	return core.Rect{X: x, Y: row.Y + (row.H-height)/2, W: w, H: height}
}

// drawBlockCaret paints a solid accent-colored block over the character at x and redraws
// that character in the accent text color so it stays readable. row is the
// line the text is centered in.
func drawBlockCaret(c *core.Ctx, dst *ebiten.Image, face *text.GoTextFace, glyph string, x int, row core.Rect) {
	c.Fill(dst, caretBlock(row, x, c.Width(face, glyph), c.LineH(face)), c.Color(core.Accent))
	c.TextMid(dst, face, glyph, x, row, c.Color(core.AccentText))
}
