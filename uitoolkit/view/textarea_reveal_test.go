package view

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Regression: the horizontal scroll that keeps the cursor in view measured
// the visible width from the window's left edge, not from the editor's, so an
// editor placed 300 px from the left scrolled as soon as the cursor passed
// the third column and the text jumped under the caret.
func TestAnEditorNotAtTheLeftEdgeScrollsOnlyWhenTheCursorReachesItsRightEdge(t *testing.T) {
	c := testCtx(t)
	area := NewTextArea("")
	area.Focused = true
	r := core.Rect{X: 300, Y: 40, W: 400, H: 200}
	g := area.geometry(c, r)
	b := layoutBars(r, g.contentW(r), g.contentH())
	fits := g.colAt(b.viewW - (g.textX - r.X)) // columns that fit before the right edge
	for i := 0; i < fits-3; i++ {
		c.In = core.Input{X: 5000, Y: 5000, Typed: []rune("x")}
		area.Update(c, r)
	}
	if area.ScrollX != 0 {
		t.Fatalf("scrolled sideways (%d px) after %d of %d columns that fit", area.ScrollX, fits-3, fits)
	}
	for i := 0; i < 10; i++ {
		c.In = core.Input{X: 5000, Y: 5000, Typed: []rune(strings.Repeat("y", 1))}
		area.Update(c, r)
	}
	if area.ScrollX == 0 {
		t.Fatal("the cursor ran past the right edge without scrolling")
	}
	line, col := area.LineCol(area.Cursor)
	_ = line
	right := g.textX - area.ScrollX + g.colX(col+1)
	if right > r.X+b.viewW {
		t.Fatalf("the cursor at x=%d is outside the editor (right edge %d)", right, r.X+b.viewW)
	}
}
