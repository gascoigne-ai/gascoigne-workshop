package core

import (
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
func caretBlock(row Rect, x, w, lineH int) Rect {
	height := min(lineH, row.H)
	return Rect{X: x, Y: row.Y + (row.H-height)/2, W: w, H: height}
}

// DrawBlockCaret paints a solid accent-colored block over the character
// of runes at cursor, which is drawn at x, and redraws that character in
// the accent text color so it stays readable. row is the line the text
// is centered in.
func (c *Ctx) DrawBlockCaret(dst *ebiten.Image, face *text.GoTextFace, runes []rune, cursor, x int, row Rect) {
	glyph := caretGlyph(runes, cursor)
	c.Fill(dst, caretBlock(row, x, c.Width(face, glyph), c.LineH(face)), c.Color(Accent))
	c.TextMid(dst, face, glyph, x, row, c.Color(AccentText))
}
