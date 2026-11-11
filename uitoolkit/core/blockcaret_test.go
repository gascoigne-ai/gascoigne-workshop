package core

import "testing"

func TestCaretGlyphIsTheRuneUnderTheBlock(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		cursor int
		want   string
	}{
		{"inside text", "héllo", 1, "é"},
		{"at the start", "abc", 0, "a"},
		{"at the end shows a blank cell", "abc", 3, " "},
		{"empty text", "", 0, " "},
		{"before a line break shows a blank cell", "ab\ncd", 2, " "},
		{"before a tab shows a blank cell", "a\tb", 1, " "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := caretGlyph([]rune(c.text), c.cursor); got != c.want {
				t.Fatalf("caretGlyph(%q, %d) = %q, want %q", c.text, c.cursor, got, c.want)
			}
		})
	}
}

// A block sized from the row height minus insets shrank to a sliver in short
// table cells; its height must come from the font's line height, centered.
func TestCaretBlockIsLineHighWhateverTheRowHeight(t *testing.T) {
	cases := []struct {
		name string
		row  Rect
		want Rect
	}{
		{"tall row", Rect{X: 0, Y: 10, W: 200, H: 40}, Rect{X: 30, Y: 22, W: 8, H: 16}},
		{"short table cell", Rect{X: 0, Y: 10, W: 200, H: 20}, Rect{X: 30, Y: 12, W: 8, H: 16}},
		{"row shorter than the line", Rect{X: 0, Y: 10, W: 200, H: 12}, Rect{X: 30, Y: 10, W: 8, H: 12}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := caretBlock(c.row, 30, 8, 16); got != c.want {
				t.Fatalf("caretBlock = %+v, want %+v", got, c.want)
			}
		})
	}
}
