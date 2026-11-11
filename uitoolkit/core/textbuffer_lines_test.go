package core

import "testing"

const poem = "abcdef\nab\nabcdef"

func area(text string) *TextBuffer {
	f := NewTextBuffer(text)
	f.Multiline = true
	f.ViewLines = 2
	return &f
}

func TestMultilineInsertKeepsLineBreaks(t *testing.T) {
	f := area("")
	f.Insert("a\r\nb\rc\nd")
	if f.Text != "a\nb\nc\nd" {
		t.Fatalf("text = %q (CRLF and CR become LF)", f.Text)
	}
	single := NewTextBuffer("")
	single.Insert("a\nb")
	if single.Text != "a b" {
		t.Fatalf("a single-line field still flattens: %q", single.Text)
	}
}

func TestLineColAndPosAt(t *testing.T) {
	f := area(poem)
	cases := []struct{ pos, line, col int }{{0, 0, 0}, {6, 0, 6}, {7, 1, 0}, {9, 1, 2}, {10, 2, 0}, {16, 2, 6}}
	for _, c := range cases {
		if l, col := f.LineCol(c.pos); l != c.line || col != c.col {
			t.Errorf("LineCol(%d) = %d,%d want %d,%d", c.pos, l, col, c.line, c.col)
		}
		if got := f.PosAt(c.line, c.col); got != c.pos {
			t.Errorf("PosAt(%d,%d) = %d want %d", c.line, c.col, got, c.pos)
		}
	}
	if got := f.PosAt(1, 50); got != 9 {
		t.Errorf("a column past the line end clamps to it, got %d", got)
	}
	if got := f.PosAt(99, 0); got != len([]rune(poem)) {
		t.Errorf("a line past the end clamps to the end, got %d", got)
	}
	if f.LineCount() != 3 {
		t.Errorf("LineCount = %d", f.LineCount())
	}
}

func TestVerticalMovementKeepsTheGoalColumn(t *testing.T) {
	f := area(poem)
	f.Cursor = 5 // line 0, col 5
	f.Apply(nil, []Edit{EditDown}, false)
	if l, c := f.LineCol(f.Cursor); l != 1 || c != 2 {
		t.Fatalf("down: %d,%d want 1,2 (clamped to the short line)", l, c)
	}
	f.Apply(nil, []Edit{EditDown}, false)
	if l, c := f.LineCol(f.Cursor); l != 2 || c != 5 {
		t.Fatalf("down again: %d,%d want 2,5 (the goal column comes back)", l, c)
	}
	f.Apply(nil, []Edit{EditUp, EditUp}, false)
	if l, c := f.LineCol(f.Cursor); l != 0 || c != 5 {
		t.Fatalf("up twice: %d,%d", l, c)
	}
	f.Apply(nil, []Edit{EditLeft, EditDown}, false) // a horizontal move resets the goal
	if l, c := f.LineCol(f.Cursor); l != 1 || c != 2 {
		t.Fatalf("after Left, Down: %d,%d want 1,2", l, c)
	}
}

func TestVerticalMovementAtTheEdges(t *testing.T) {
	f := area(poem)
	f.Cursor = 3
	f.Apply(nil, []Edit{EditUp}, false)
	if f.Cursor != 0 {
		t.Fatalf("Up on the first line goes to the start, got %d", f.Cursor)
	}
	f.Cursor = 12
	f.Apply(nil, []Edit{EditDown}, false)
	if f.Cursor != len([]rune(poem)) {
		t.Fatalf("Down on the last line goes to the end, got %d", f.Cursor)
	}
	one := NewTextBuffer("single line")
	one.Cursor = 3
	one.Apply(nil, []Edit{EditUp, EditDown}, false)
	if one.Cursor != 3 {
		t.Fatalf("a single-line field ignores Up and Down, cursor %d", one.Cursor)
	}
}

func TestShiftDownSelectsAcrossLines(t *testing.T) {
	f := area(poem)
	f.Cursor = 2
	f.Apply(nil, []Edit{EditDown}, true)
	if f.SelectedText() != "cdef\nab" {
		t.Fatalf("selection %q", f.SelectedText())
	}
}

func TestHomeEndAreLineWiseAndDocWiseWithCtrl(t *testing.T) {
	f := area(poem)
	f.Cursor = 8 // inside "ab"
	f.Apply(nil, []Edit{EditHome}, false)
	if f.Cursor != 7 {
		t.Fatalf("Home = %d, want 7", f.Cursor)
	}
	f.Apply(nil, []Edit{EditEnd}, false)
	if f.Cursor != 9 {
		t.Fatalf("End = %d, want 9", f.Cursor)
	}
	f.Apply(nil, []Edit{EditDocStart}, false)
	if f.Cursor != 0 {
		t.Fatalf("DocStart = %d", f.Cursor)
	}
	f.Apply(nil, []Edit{EditDocEnd}, false)
	if f.Cursor != 16 {
		t.Fatalf("DocEnd = %d", f.Cursor)
	}
}

func TestPageMovesByViewLines(t *testing.T) {
	f := area("1\n2\n3\n4\n5\n6")
	f.Cursor = 0
	f.Apply(nil, []Edit{EditPageDown}, false)
	if l, _ := f.LineCol(f.Cursor); l != 2 {
		t.Fatalf("PageDown with a 2-line view: line %d, want 2", l)
	}
	f.Apply(nil, []Edit{EditPageUp}, false)
	if l, _ := f.LineCol(f.Cursor); l != 0 {
		t.Fatalf("PageUp: line %d", l)
	}
}

func TestEnterKeepsTheIndentation(t *testing.T) {
	tests := []struct {
		name, text string
		cursor     int
		want       string
		wantCursor int
	}{
		{"indented line", "  foo", 5, "  foo\n  ", 8},
		{"mid line keeps the tail", "  foo", 4, "  fo\n  o", 7},
		{"no indent", "foo", 3, "foo\n", 4},
		{"cursor inside the indent only copies up to the cursor", "    x", 2, "  \n    x", 5},
		{"tab indent", "\tx", 2, "\tx\n\t", 4},
	}
	for _, tt := range tests {
		f := area(tt.text)
		f.Cursor = tt.cursor
		f.Apply(nil, []Edit{EditNewline}, false)
		if f.Text != tt.want || f.Cursor != tt.wantCursor {
			t.Errorf("%s: %q cursor %d; want %q %d", tt.name, f.Text, f.Cursor, tt.want, tt.wantCursor)
		}
	}
}

func TestEnterReplacesTheSelectionAndUndoes(t *testing.T) {
	f := area("a-b")
	f.Anchor, f.Cursor, f.HasSel = 1, 2, true
	f.Apply(nil, []Edit{EditNewline}, false)
	if f.Text != "a\nb" {
		t.Fatalf("text = %q", f.Text)
	}
	f.Undo()
	if f.Text != "a-b" {
		t.Fatalf("undo = %q", f.Text)
	}
}

func TestTabInsertsSpacesToTheNextStop(t *testing.T) {
	f := area("ab")
	f.Apply(nil, []Edit{EditIndent}, false) // col 2 is on a stop: a full tab width
	if f.Text != "ab  " {
		t.Fatalf("text = %q", f.Text)
	}
	g := area("a")
	g.Apply(nil, []Edit{EditIndent}, false) // col 1: one space reaches the stop
	if g.Text != "a " {
		t.Fatalf("text = %q", g.Text)
	}
}

func TestTabOnASelectionIndentsEveryLineAndShiftTabDedents(t *testing.T) {
	f := area("one\ntwo\nthree")
	f.Anchor, f.Cursor, f.HasSel = 1, 9, true // from "one" into "two"... through the second line
	f.Apply(nil, []Edit{EditIndent}, false)
	if f.Text != "  one\n  two\n  three" {
		t.Fatalf("indent: %q", f.Text)
	}
	if lo, hi, ok := f.Selection(); !ok || lo != 3 || hi != 15 {
		t.Fatalf("the selection must follow the text, got %d..%d", lo, hi)
	}
	f.Apply(nil, []Edit{EditDedent}, false)
	if f.Text != "one\ntwo\nthree" {
		t.Fatalf("dedent: %q", f.Text)
	}
}

func TestShiftTabDedentsTheCurrentLineAndStopsAtZero(t *testing.T) {
	f := area("    x\ny")
	f.Cursor = 5 // end of the indented line
	f.Apply(nil, []Edit{EditDedent}, false)
	if f.Text != "  x\ny" || f.Cursor != 3 {
		t.Fatalf("text %q cursor %d", f.Text, f.Cursor)
	}
	f.Apply(nil, []Edit{EditDedent, EditDedent}, false)
	if f.Text != "x\ny" {
		t.Fatalf("text %q", f.Text)
	}
	f.Cursor = 0
	if f.Apply(nil, []Edit{EditDedent}, false); f.Text != "x\ny" {
		t.Fatal("dedenting an unindented line changes nothing")
	}
	g := area("\tx")
	g.Cursor = 2
	g.Apply(nil, []Edit{EditDedent}, false)
	if g.Text != "x" {
		t.Fatalf("a tab is removed whole: %q", g.Text)
	}
}
