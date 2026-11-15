package widget

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// wrappedArea is a focused, wrapping 400x200 editor holding a long line
// of words and a short one, updated once with the pointer away.
func wrappedArea(t *testing.T) (*TextArea, *core.Ctx, core.Rect) {
	t.Helper()
	c := testCtx(t)
	area := NewTextArea(words() + "\nend")
	area.Wrap, area.Focused, area.Cursor = true, true, 0
	r := core.Rect{X: 10, Y: 20, W: 400, H: 200}
	c.In = core.Input{X: 5000, Y: 5000}
	area.Update(c, r)
	return &area, c, r
}

func TestWrappedAreaBreaksRowsBeforeWordsAndNeverScrollsSideways(t *testing.T) {
	area, c, r := wrappedArea(t)
	layout := area.layout(area.geometry(c, r), r)
	text := []rune(words())
	rows := 0
	for _, row := range layout.rows {
		if row.line != 0 {
			continue
		}
		rows++
		if row.from > 0 && string(text[row.from:row.from+4]) != "word" {
			t.Fatalf("a row starts at column %d with %q, want the start of a word", row.from, string(text[row.from:row.from+4]))
		}
	}
	if rows < 3 {
		t.Fatalf("the long line has %d rows in a 400px editor, want it wrapped", rows)
	}
	area.Cursor = len(text) // the end of the long line, far right when not wrapped
	c.In = core.Input{X: 5000, Y: 5000, Edits: []core.Edit{core.EditEnd}}
	area.Update(c, r)
	c.In = core.Input{X: 50, Y: 50, Wheel: 3, Shift: true}
	area.Update(c, r)
	if area.ScrollX != 0 {
		t.Fatalf("ScrollX = %d: wrapped text fits the width", area.ScrollX)
	}
}

func TestClickOnAWrappedRowOfTheAreaPutsTheCursorInThatRow(t *testing.T) {
	area, c, r := wrappedArea(t)
	g := area.geometry(c, r)
	second := area.layout(g, r).rows[1]
	c.In = core.Input{X: g.textX + g.colX(2), Y: r.Y + g.lineH + g.lineH/2, Click: true, Held: true, Clicks: 1}
	area.Update(c, r)
	if want := second.from + 2; area.Cursor != want {
		t.Fatalf("cursor %d after a click two columns into the second row, want %d", area.Cursor, want)
	}
}

func TestDownInAWrappedAreaMovesOneVisualRow(t *testing.T) {
	area, c, r := wrappedArea(t)
	second := area.layout(area.geometry(c, r), r).rows[1]
	c.In = core.Input{X: 5000, Y: 5000, Edits: []core.Edit{core.EditDown}}
	area.Update(c, r)
	if area.Cursor != second.from {
		t.Fatalf("Down from the start went to %d, want the start of the second row (%d)", area.Cursor, second.from)
	}
	c.In = core.Input{X: 5000, Y: 5000, Edits: []core.Edit{core.EditUp}}
	area.Update(c, r)
	if area.Cursor != 0 {
		t.Fatalf("Up went to %d, want back to 0", area.Cursor)
	}
}

func TestWrappedAreaReportsTheCaretOnItsRow(t *testing.T) {
	area, c, r := wrappedArea(t)
	g := area.geometry(c, r)
	layout := area.layout(g, r)
	area.Cursor = layout.rows[1].from + 3
	c.In = core.Input{X: 5000, Y: 5000}
	area.Update(c, r)
	x, y, ok := area.CaretAt()
	wantX, wantY := g.textX+g.colX(3), r.Y+2*g.lineH-area.ScrollY // the bottom of the second row
	if !ok || x != wantX || y != wantY {
		t.Fatalf("CaretAt = %d,%d,%v want %d,%d", x, y, ok, wantX, wantY)
	}
}

func TestTypingAtTheEndOfAWrappedAreaKeepsTheCaretRowInView(t *testing.T) {
	c := testCtx(t)
	area := NewTextArea(strings.Repeat(words()+"\n", 3))
	area.Wrap, area.Focused = true, true
	r := core.Rect{W: 400, H: 100}
	c.In = core.Input{X: 5000, Y: 5000, Typed: []rune("x")}
	area.Update(c, r)
	g := area.geometry(c, r)
	row, _ := area.layout(g, r).xOf(textPos{3, 1})
	if top := row * g.lineH; top < area.ScrollY || top+g.lineH > area.ScrollY+r.H {
		t.Fatalf("ScrollY %d does not show the caret's row %d", area.ScrollY, row)
	}
}

func TestAreaWrapOptionSwitchesTheEditorsWrap(t *testing.T) {
	c := testCtx(t)
	area := NewTextArea(words())
	drawn(t, c, Area(&area, Wrap()), 400, 200)
	if !area.Wrap {
		t.Fatal("Area with Wrap() left the editor unwrapped")
	}
	drawn(t, c, Area(&area), 400, 200)
	if area.Wrap {
		t.Fatal("Area without Wrap() left the editor wrapped")
	}
}
