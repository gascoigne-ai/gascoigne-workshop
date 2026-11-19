package widget

import (
	"reflect"
	"testing"
)

// fakeMeasure stands in for a font: one unit per rune, four for a tab and
// two for the CJK runes that a real font draws double wide.
func fakeMeasure(char rune) float64 {
	switch {
	case char == '\t':
		return 4
	case char >= 0x2E80:
		return 2
	}
	return 1
}

func oneLine(text string) []wrapLine { return []wrapLine{{line: 0, text: []rune(text)}} }

func TestLayoutWrapBreaksLinesIntoVisualRows(t *testing.T) {
	tests := []struct {
		name  string
		lines []wrapLine
		width float64
		want  []visualRow
	}{
		{"an empty line keeps one row", oneLine(""), 10,
			[]visualRow{{0, 0, 0, 0}}},
		{"an exactly fitting line is one row", oneLine("abcdefghij"), 10,
			[]visualRow{{0, 0, 10, 0}}},
		{"a line breaks before the word that does not fit", oneLine("hello world foo"), 8,
			[]visualRow{{0, 0, 6, 0}, {0, 6, 12, 6}, {0, 12, 15, 12}}},
		{"a word wider than the view is cut at the width", oneLine("abcdefghijklmnop"), 5,
			[]visualRow{{0, 0, 5, 0}, {0, 5, 10, 5}, {0, 10, 15, 10}, {0, 15, 16, 15}}},
		{"spaces at a break hang at the end of the row", oneLine("ab    cd"), 3,
			[]visualRow{{0, 0, 6, 0}, {0, 6, 8, 6}}},
		{"trailing spaces add no row", oneLine("abc   "), 3,
			[]visualRow{{0, 0, 6, 0}}},
		{"punctuation stays at the end of its row", oneLine(`{"key":"value"}`), 10,
			[]visualRow{{0, 0, 8, 0}, {0, 8, 15, 8}}},
		{"a tab is measured and hangs like a space", oneLine("a\tb"), 4,
			[]visualRow{{0, 0, 2, 0}, {0, 2, 3, 5}}},
		{"a tab ends the row before the word", oneLine("\tabcd"), 6,
			[]visualRow{{0, 0, 1, 0}, {0, 1, 5, 4}}},
		{"wide runes are measured by their width", oneLine("日本語テキスト"), 5,
			[]visualRow{{0, 0, 2, 0}, {0, 2, 4, 4}, {0, 4, 6, 8}, {0, 6, 7, 12}}},
		{"a wide rune that does not fit goes to the next row whole", oneLine("ab日"), 3,
			[]visualRow{{0, 0, 2, 0}, {0, 2, 3, 2}}},
		{"a width smaller than one glyph still shows a rune per row", oneLine("abc"), 0.5,
			[]visualRow{{0, 0, 1, 0}, {0, 1, 2, 1}, {0, 2, 3, 2}}},
		{"no width means no wrap", oneLine("hello world foo"), 0,
			[]visualRow{{0, 0, 15, 0}}},
		{"a whole line (a folded one) is never broken",
			[]wrapLine{{line: 0, text: []rune("hello world foo"), whole: true}}, 3,
			[]visualRow{{0, 0, 15, 0}}},
		{"rows keep the original line numbers of the lines given",
			[]wrapLine{{line: 2, text: []rune("ab cd")}, {line: 7, text: []rune("ef")}}, 3,
			[]visualRow{{2, 0, 3, 0}, {2, 3, 5, 3}, {7, 0, 2, 0}}},
		{"no lines, no rows", nil, 10, []visualRow{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layoutWrap(tt.lines, tt.width, fakeMeasure).rows
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// sample lays out three lines into rows 0..2 ("hello ", "world ", "foo"),
// 3 (the empty line 2) and 4 ("a\tb", where the tab spans x 1..5); line 1
// is folded away.
func sample() wrapLayout {
	return layoutWrap([]wrapLine{
		{line: 0, text: []rune("hello world foo")},
		{line: 2, text: nil},
		{line: 3, text: []rune("a\tb")},
	}, 8, fakeMeasure)
}

func TestWrapLayoutRowOfAPositionTakesTheBreakToTheNextRow(t *testing.T) {
	tests := []struct {
		name string
		pos  textPos
		want int
	}{
		{"the start of a line", textPos{0, 0}, 0},
		{"inside the first row", textPos{0, 5}, 0},
		{"the break belongs to the row it starts", textPos{0, 6}, 1},
		{"the end of the line is on its last row", textPos{0, 15}, 2},
		{"a column past the end clamps to the last row", textPos{0, 99}, 2},
		{"an empty line", textPos{2, 0}, 3},
		{"a later line", textPos{3, 2}, 4},
		{"a line folded away has no row", textPos{1, 0}, -1},
		{"a line after the last has no row", textPos{9, 0}, -1},
	}
	layout := sample()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := layout.rowOf(tt.pos); got != tt.want {
				t.Fatalf("rowOf(%v) = %d, want %d", tt.pos, got, tt.want)
			}
		})
	}
}

func TestWrapLayoutMapsPositionsToPixelsAndBack(t *testing.T) {
	tests := []struct {
		name string
		pos  textPos
		row  int
		x    float64
	}{
		{"the start of the text", textPos{0, 0}, 0, 0},
		{"inside a continuation row", textPos{0, 8}, 1, 2},
		{"the end of the line", textPos{0, 15}, 2, 3},
		{"after a tab", textPos{3, 2}, 4, 5},
		{"before a tab", textPos{3, 1}, 4, 1},
	}
	layout := sample()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, x := layout.xOf(tt.pos)
			if row != tt.row || x != tt.x {
				t.Fatalf("xOf(%v) = row %d x %v, want row %d x %v", tt.pos, row, x, tt.row, tt.x)
			}
			if back := layout.posAt(row, x); back != tt.pos {
				t.Fatalf("posAt(%d, %v) = %v, want %v back", row, x, back, tt.pos)
			}
		})
	}
}

func TestWrapLayoutPosAtPicksTheNearestColumnInsideTheRow(t *testing.T) {
	tests := []struct {
		name string
		row  int
		x    float64
		want textPos
	}{
		{"left of the text is the row's start", 1, -3, textPos{0, 6}},
		{"just before a rune's middle", 1, 2.4, textPos{0, 8}},
		{"just after a rune's middle", 1, 2.6, textPos{0, 9}},
		{"right of a continued row stays before its break", 1, 100, textPos{0, 11}},
		{"right of the last row is the line's end", 2, 100, textPos{0, 15}},
		{"a wide tab rounds by its own width", 4, 2.9, textPos{3, 1}},
		{"past a tab's middle is after it", 4, 3.1, textPos{3, 2}},
		{"an empty line", 3, 5, textPos{2, 0}},
		{"a row above the first clamps to it", -2, 0, textPos{0, 0}},
		{"a row below the last clamps to it", 99, 100, textPos{3, 3}},
	}
	layout := sample()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := layout.posAt(tt.row, tt.x); got != tt.want {
				t.Fatalf("posAt(%d, %v) = %v, want %v", tt.row, tt.x, got, tt.want)
			}
		})
	}
}

func TestWrapLayoutPosAtStaysOnTheRowThatWasHit(t *testing.T) {
	layouts := map[string]wrapLayout{
		"word breaks": sample(),
		"hard breaks": layoutWrap(oneLine("abcdefghijklmnop"), 5, fakeMeasure),
		"wide runes":  layoutWrap(oneLine("日本語テキスト"), 5, fakeMeasure),
	}
	for name, layout := range layouts {
		for row := range layout.rows {
			for _, x := range []float64{-5, 1.5, 1000} {
				if got := layout.rowOf(layout.posAt(row, x)); got != row {
					t.Fatalf("%s: a hit on row %d at x %v lands on row %d", name, row, x, got)
				}
			}
		}
	}
}

func TestWrapLayoutMapsScrollRowsToLinesAndBack(t *testing.T) {
	layout := sample()
	tests := []struct {
		row  int
		want textPos
	}{
		{0, textPos{0, 0}}, {1, textPos{0, 6}}, {2, textPos{0, 12}}, {3, textPos{2, 0}}, {4, textPos{3, 0}},
	}
	for _, tt := range tests {
		if got := layout.rowStart(tt.row); got != tt.want {
			t.Fatalf("rowStart(%d) = %v, want %v", tt.row, got, tt.want)
		}
	}
}

func TestWrapLayoutOfNoLinesAnswersWithoutRows(t *testing.T) {
	layout := layoutWrap(nil, 10, fakeMeasure)
	if got := layout.rowOf(textPos{0, 0}); got != -1 {
		t.Fatalf("rowOf = %d, want -1", got)
	}
	if got := layout.posAt(0, 0); got != (textPos{}) {
		t.Fatalf("posAt = %v, want the zero position", got)
	}
	if got := layout.rowStart(0); got != (textPos{}) {
		t.Fatalf("rowStart = %v, want the zero position", got)
	}
}

// FuzzWrapRunes: any text at any width, including zero and negative, with
// a measure that gives every rune a width; it must terminate and keep
// every rune in some row.
func FuzzWrapRunes(f *testing.F) {
	f.Add("hello world, wrapped", 40.0, 8.0)
	f.Add("żółw", 0.0, 8.0)
	f.Add("a b", -5.0, 0.0)
	f.Fuzz(func(t *testing.T, text string, width, advance float64) {
		rows := wrapRunes(0, []rune(text), width, func(rune) float64 { return advance })
		covered := 0
		for _, row := range rows {
			covered += row.to - row.from
		}
		if len([]rune(text)) > 0 && covered != len([]rune(text)) {
			t.Fatalf("rows cover %d of %d runes", covered, len([]rune(text)))
		}
	})
}
