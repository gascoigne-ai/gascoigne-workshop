package widget

import (
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Regression: a tab in a monospace view (a raw body, a TextArea) was one
// column wide and drawn as the font's missing-glyph box, since the fonts
// have no glyph for U+0009 and every surface counted it as one rune. The
// text after a tab did not move to the next tab stop, and the caret, the
// selection, hit-testing and the horizontal extent all counted it as one
// column.
func TestATabInAnEditorAdvancesToTheNextTabStop(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		cursor int
		col    int // display column of the caret
	}{
		{"after a leading tab", "\tb", 1, 4},
		{"after a tab inside a word", "ab\tc", 3, 4},
		{"a tab at a stop takes a whole stop", "abcd\te", 5, 8},
		{"two tabs", "\t\tx", 2, 8},
		{"text after a tab", "a\tbc", 4, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testCtx(t)
			area := NewTextArea(tt.text)
			area.Cursor = tt.cursor
			area.Focused = true
			r := core.Rect{W: 600, H: 200}
			c.In = core.Input{X: 5000, Y: 5000}
			area.Update(c, r)
			g := area.geometry(c, r)
			x, _, _ := area.CaretAt()
			if want := g.textX + g.colX(tt.col); x != want {
				t.Fatalf("caret x = %d (column %.1f), want %d (column %d)", x, float64(x-g.textX)/g.adv, want, tt.col)
			}
		})
	}
}

func TestAnEditorsWidthCountsTabsToTheirStops(t *testing.T) {
	c := testCtx(t)
	area := NewTextArea("\t\tx\nab")
	r := core.Rect{W: 600, H: 200}
	g := area.geometry(c, r)
	if got, want := g.contentW(r), g.textX-r.X+g.colX(9)+core.Gutter; got != want {
		t.Fatalf("contentW = %d, want %d (9 columns)", got, want)
	}
}

func TestClickingInAnEditorFindsTheRuneAfterATab(t *testing.T) {
	tests := []struct {
		name string
		col  float64 // where the click lands, in columns
		want int     // the cursor after the click
	}{
		{"in the first half of the tab's gap", 1, 0},
		{"in the second half of the tab's gap", 3, 1},
		{"on the character after the tab", 4.2, 1},
		{"past the character after the tab", 4.8, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testCtx(t)
			area := NewTextArea("\tb")
			r := core.Rect{W: 600, H: 200}
			g := area.geometry(c, r)
			c.In = core.Input{X: g.textX + int(tt.col*g.adv), Y: r.Y + g.lineH/2, Click: true, Held: true, Clicks: 1}
			area.Update(c, r)
			if area.Cursor != tt.want {
				t.Fatalf("cursor = %d, want %d", area.Cursor, tt.want)
			}
		})
	}
}

func TestMovingDownFromAfterATabKeepsTheDisplayColumn(t *testing.T) {
	c := testCtx(t)
	area := NewTextArea("\tx\nabcdef")
	area.Cursor = 1 // after the tab: column 4
	area.Focused = true
	r := core.Rect{W: 600, H: 200}
	c.In = core.Input{X: 5000, Y: 5000, Edits: []core.Edit{core.EditDown}}
	area.Update(c, r)
	if line, col := area.LineCol(area.Cursor); line != 1 || col != 4 {
		t.Fatalf("after Down the cursor is at line %d rune %d, want line 1 rune 4", line, col)
	}
}

func tabbedView(t *testing.T) (*CodeView, *core.Ctx, *memClip, core.Rect) {
	t.Helper()
	v, c, clip, r := viewport(t)
	v.SetLines([]Line{{Spans: []Span{{Text: "\tkey", Style: core.Text}, {Text: "\tvalue", Style: core.Text}}}})
	return v, c, clip, r
}

func TestACodeViewsWidthCountsTabsToTheirStops(t *testing.T) {
	v, c, _, r := tabbedView(t)
	g := v.geometry(c, r)
	// "\tkey\tvalue": key at 4..7, the tab to 8, value at 8..13
	if got, want := v.contentWidth(c, r), g.textX-r.X+g.colX(13)+core.Gutter; got != want {
		t.Fatalf("contentWidth = %d, want %d (13 columns)", got, want)
	}
}

func TestSelectingInACodeViewByDisplayColumnCopiesTheRealTab(t *testing.T) {
	v, c, clip, r := tabbedView(t)
	y := rowY(c, v, r, 0)
	// from the start of "key" (column 4) to the start of "value" (column 8)
	press(v, c, r, colX(c, v, r, 4)+1, y, 1)
	dragTo(v, c, r, colX(c, v, r, 8)+1, y)
	release(v, c, r, 0, 0)
	if got := v.SelectedText(); got != "key\t" {
		t.Fatalf("selected %q, want %q", got, "key\t")
	}
	c.In = core.Input{X: 5000, Y: 5000, Ctrl: true, Copy: true}
	v.Update(c, r)
	if clip.text != "key\t" {
		t.Fatalf("copied %q, want the real tab byte in %q", clip.text, "key\t")
	}
}

// Counting a match's columns from the row's start for every match made a
// frame of a long unwrapped line with many hits take time quadratic in the
// line's length.
func TestDrawingManyMatchesOnALongLineIsFast(t *testing.T) {
	v, c, _, r := viewport(t)
	line := strings.Repeat("key\tvalue ", 30000) // 300k runes
	v.SetLines([]Line{{Spans: []Span{{Text: line, Style: core.Text}}}})
	v.SetMatches(v.Find("e"), 0)
	dst := ebiten.NewImage(r.W, r.H)
	start := time.Now()
	v.Draw(c, dst, r)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("one frame with %d matches took %v", len(v.Matches()), elapsed)
	}
}
