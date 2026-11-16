package widget

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// wrapCols is how many columns fit in a wrapped view's rows.
func wrapCols(code *CodeView, ctx *core.Ctx, rect core.Rect) int {
	g := code.geometry(ctx, rect)
	return int(code.wrapWidth(g, rect) / g.adv)
}

// longLine is 200 runes: the digits 0-9 twenty times, so a column shows
// in the text itself. It is one word, so it is cut at the width.
func wrapped(t *testing.T) (*CodeView, *core.Ctx, core.Rect, int) {
	t.Helper()
	v, c, _, r := viewport(t)
	v.SetLines([]Line{
		{Spans: []Span{{Text: strings.Repeat("0123456789", 20)}}},
		{Spans: []Span{{Text: "tail"}}},
	})
	v.Wrap = true
	c.In = core.Input{X: 5000, Y: 5000}
	v.Update(c, r)
	cols := wrapCols(v, c, r)
	if cols <= 0 || cols >= 200 {
		t.Fatalf("a 600px view fits %d columns, expected some but fewer than 200", cols)
	}
	return v, c, r, cols
}

// words is a line of 60 seven-rune words ("word00 word01 ..."), so a break
// by columns would fall inside a word.
func words() string {
	parts := make([]string, 60)
	for i := range parts {
		parts[i] = "word" + string(rune('0'+i/10)) + string(rune('0'+i%10))
	}
	return strings.Join(parts, " ")
}

func TestWrappedViewBreaksRowsBeforeWords(t *testing.T) {
	v, c, _, r := viewport(t)
	v.SetLines([]Line{{Spans: []Span{{Text: words()}}}})
	v.Wrap = true
	layout := v.layout(v.geometry(c, r), r)
	if len(layout.rows) < 3 {
		t.Fatalf("%d rows: a 419-rune line should wrap in a 600px view", len(layout.rows))
	}
	text := []rune(words())
	for _, row := range layout.rows[1:] {
		if got := string(text[row.from:min(row.from+4, len(text))]); got != "word" {
			t.Fatalf("a row starts at column %d with %q, want the start of a word", row.from, got)
		}
	}
}

func TestClickOnAWrappedRowLandsInTheWordUnderThePointer(t *testing.T) {
	v, c, _, r := viewport(t)
	v.SetLines([]Line{{Spans: []Span{{Text: words()}}}})
	v.Wrap = true
	g := v.geometry(c, r)
	second := v.layout(g, r).rows[1]
	press(v, c, r, g.textX+g.colX(2), rowY(c, v, r, 1), 2)
	want := string([]rune(words())[second.from : second.from+6])
	if got := v.SelectedText(); got != want {
		t.Fatalf("double click on the second row selected %q, want %q", got, want)
	}
}

func TestFoldedLineIsOneRowAndTheLinesItHidesHaveNone(t *testing.T) {
	v, c, _, r := viewport(t)
	v.SetLines([]Line{
		{Spans: []Span{{Text: "{ " + words()}}, Foldable: true},
		{Spans: []Span{{Text: "  " + words()}}},
		{Spans: []Span{{Text: "}"}}},
		{Spans: []Span{{Text: "after"}}},
	})
	v.Wrap = true
	v.ToggleFold(0)
	rows := v.layout(v.geometry(c, r), r).rows
	if len(rows) != 2 || rows[0].line != 0 || rows[1].line != 3 {
		t.Fatalf("rows %v, want the folded line whole and then line 3", rows)
	}
}

func TestWrappedViewNeverScrollsSideways(t *testing.T) {
	v, c, r, _ := wrapped(t)
	c.In = core.Input{X: 100, Y: 100, Wheel: 3, Shift: true}
	v.Update(c, r)
	if v.ScrollX != 0 {
		t.Fatalf("ScrollX = %d: wrapped text fits the width", v.ScrollX)
	}
}

func TestClickingASecondVisualRowSelectsFromThatPartOfTheLine(t *testing.T) {
	v, c, r, cols := wrapped(t)
	g := v.geometry(c, r)
	y := r.Y + g.lineH + g.lineH/2       // the second visual row, the continuation of line 0
	x := g.textX - v.ScrollX + g.colX(3) // three columns into that row
	press(v, c, r, x, y, 1)
	dragTo(v, c, r, x+g.colX(4), y)
	release(v, c, r, x+g.colX(4), y)
	want := strings.Repeat("0123456789", 20)[cols+3 : cols+7]
	if got := v.SelectedText(); got != want {
		t.Fatalf("selection %q, want %q (columns %d..%d of the line)", got, want, cols+3, cols+7)
	}
}

func TestLinesAfterAWrappedLineSitBelowItsLastRow(t *testing.T) {
	v, c, r, cols := wrapped(t)
	g := v.geometry(c, r)
	rows := (200 + cols - 1) / cols
	y := r.Y + rows*g.lineH + g.lineH/2 // the row of line 1
	press(v, c, r, g.textX+g.charW, y, 2)
	if got := v.SelectedText(); got != "tail" {
		t.Fatalf("double click on the row after the wrapped line selected %q, want tail", got)
	}
}

func TestRevealingALineCountsTheRowsOfWrappedLinesAbove(t *testing.T) {
	v, c, _, _ := viewport(t)
	lines := make([]Line, 60)
	for i := range lines {
		lines[i] = Line{Spans: []Span{{Text: strings.Repeat("0123456789", 20)}}}
	}
	v.SetLines(lines)
	v.Wrap = true
	r := core.Rect{W: 600, H: 100}
	g := v.geometry(c, r)
	cols := wrapCols(v, c, r)
	rowsPerLine := (200 + cols - 1) / cols
	v.RevealLine(50)
	c.In = core.Input{X: 5000, Y: 5000}
	v.Update(c, r)
	top, bottom := 50*rowsPerLine*g.lineH, (50*rowsPerLine+rowsPerLine)*g.lineH
	if v.ScrollY > top || v.ScrollY+r.H < bottom-g.lineH*(rowsPerLine-1) {
		t.Fatalf("ScrollY %d does not show line 50 (rows start at %d..%d)", v.ScrollY, top, bottom)
	}
}

// A minified body is one line of many rows: revealing a hit by its line
// alone would leave a hit far along the line below the view.
func TestRevealMatchScrollsToTheRowOfTheHit(t *testing.T) {
	v, c, _, _ := viewport(t)
	text := strings.Repeat(words()+" ", 10) + "needle"
	v.SetLines([]Line{{Spans: []Span{{Text: text}}}})
	v.Wrap = true
	r := core.Rect{W: 600, H: 100}
	g := v.geometry(c, r)
	hit := v.Find("needle")[0]
	v.RevealMatch(hit)
	c.In = core.Input{X: 5000, Y: 5000}
	v.Update(c, r)
	row := v.layout(g, r).rowOf(textPos{hit.Line, hit.Col})
	if top := row * g.lineH; top < v.ScrollY || top+g.lineH > v.ScrollY+r.H {
		t.Fatalf("ScrollY %d does not show row %d of the hit (y %d)", v.ScrollY, row, top)
	}
}

func TestTurningWrapOnOrOffKeepsTheTopLine(t *testing.T) {
	v, c, _, _ := viewport(t)
	lines := make([]Line, 40)
	for i := range lines {
		lines[i] = Line{Spans: []Span{{Text: words()}}}
	}
	v.SetLines(lines)
	r := core.Rect{W: 600, H: 200}
	g := v.geometry(c, r)
	v.ScrollY = 10 * g.lineH // line 10 at the top
	c.In = core.Input{X: 5000, Y: 5000}
	v.Wrap = true
	v.Update(c, r)
	if got := v.layout(g, r).rowStart(v.ScrollY / g.lineH); got != (textPos{10, 0}) {
		t.Fatalf("after wrapping, the top row starts at %v, want line 10", got)
	}
	v.Wrap = false
	v.Update(c, r)
	if v.ScrollY != 10*g.lineH {
		t.Fatalf("after unwrapping, ScrollY = %d, want line 10 at the top (%d)", v.ScrollY, 10*g.lineH)
	}
}

func TestShiftDownInAWrappedLineMovesOneRow(t *testing.T) {
	v, c, _, r := viewport(t)
	v.SetLines([]Line{{Spans: []Span{{Text: words()}}}, {Spans: []Span{{Text: "tail"}}}})
	v.Wrap = true
	v.Focused = true
	g := v.geometry(c, r)
	second := v.layout(g, r).rows[1]
	shiftKey(v, c, r, core.EditDown)
	if v.selB != (textPos{0, second.from}) {
		t.Fatalf("Shift+Down from the start went to %v, want the start of the second row (column %d)", v.selB, second.from)
	}
}

func TestCodeWrapOptionSwitchesTheViewsWrap(t *testing.T) {
	v, c, _, _ := viewport(t)
	drawn(t, c, Code(v, Wrap()), 600, 400)
	if !v.Wrap {
		t.Fatal("Code with Wrap() left the view unwrapped")
	}
	drawn(t, c, Code(v), 600, 400)
	if v.Wrap {
		t.Fatal("Code without Wrap() left the view wrapped: the option is how a template switches it off")
	}
}
