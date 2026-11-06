package vml

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestWrapRowsCutsLongLinesAndKeepsEmptyOnes(t *testing.T) {
	tests := []struct {
		name string
		vis  []int
		lens []int // rune length of each original line
		cols int
		want []wrapRow
	}{
		{"short lines stay whole", []int{0, 1}, []int{3, 5}, 10, []wrapRow{{0, 0, 3}, {1, 0, 5}}},
		{"a long line is cut at the width", []int{0}, []int{25}, 10, []wrapRow{{0, 0, 10}, {0, 10, 20}, {0, 20, 25}}},
		{"an exact fit is one row", []int{0}, []int{10}, 10, []wrapRow{{0, 0, 10}}},
		{"an empty line keeps its row", []int{0, 1}, []int{0, 4}, 10, []wrapRow{{0, 0, 0}, {1, 0, 4}}},
		{"only visible lines count", []int{1}, []int{40, 4}, 10, []wrapRow{{1, 0, 4}}},
		{"no room means no wrap", []int{0}, []int{25}, 0, []wrapRow{{0, 0, 25}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapRows(tt.vis, func(line int) int { return tt.lens[line] }, tt.cols)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// longLine is 200 runes: the digits 0-9 twenty times, so a column shows
// in the text itself.
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
	cols := v.wrapCols(v.geometry(c, r), r)
	if cols <= 0 || cols >= 200 {
		t.Fatalf("a 600px view fits %d columns, expected some but fewer than 200", cols)
	}
	return v, c, r, cols
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
	x := g.textX - v.ScrollX + 3*g.charW // three columns into that row
	press(v, c, r, x, y, 1)
	dragTo(v, c, r, x+4*g.charW, y)
	release(v, c, r, x+4*g.charW, y)
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
	cols := v.wrapCols(g, r)
	rowsPerLine := (200 + cols - 1) / cols
	v.RevealLine(50)
	c.In = core.Input{X: 5000, Y: 5000}
	v.Update(c, r)
	top, bottom := 50*rowsPerLine*g.lineH, (50*rowsPerLine+rowsPerLine)*g.lineH
	if v.ScrollY > top || v.ScrollY+r.H < bottom-g.lineH*(rowsPerLine-1) {
		t.Fatalf("ScrollY %d does not show line 50 (rows start at %d..%d)", v.ScrollY, top, bottom)
	}
}
