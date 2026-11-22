package core

import (
	"strings"
	"testing"
)

// Regression: the fonts have no glyph for a tab, so text with one was drawn
// with the missing-glyph box in its place, one narrow cell wide, in labels
// and fields as much as in code.
func TestATabInProportionalTextIsAGapOfFourSpaces(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		text string
	}{{"inside", "a\tb"}, {"leading", "\tb"}, {"two", "a\t\tb"}} {
		t.Run(tt.name, func(t *testing.T) {
			spaced := strings.ReplaceAll(tt.text, "\t", strings.Repeat(" ", TabStop))
			for _, face := range []struct {
				name string
				got  int
				want int
			}{
				{"interface", ctx.Width(ctx.F.UI, tt.text), ctx.Width(ctx.F.UI, spaced)},
				{"small", ctx.Width(ctx.F.Small, tt.text), ctx.Width(ctx.F.Small, spaced)},
			} {
				if face.got != face.want {
					t.Errorf("%s: width %d, want %d (a gap of %d spaces)", face.name, face.got, face.want, TabStop)
				}
			}
		})
	}
}

func TestTextColumnExpandsTabsToTheNextStop(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		index int
		want  int
	}{
		{"the start", "\tab", 0, 0},
		{"after a leading tab", "\tab", 1, 4},
		{"a rune after a tab", "\tab", 2, 5},
		{"a tab inside a word", "ab\tc", 3, 4},
		{"a tab at a stop takes a whole stop", "abcd\te", 5, 8},
		{"two tabs", "\t\tx", 2, 8},
		{"no tabs", "abc", 3, 3},
		{"past the end is the end", "a\t", 9, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TextColumn([]rune(tt.line), tt.index); got != tt.want {
				t.Fatalf("TextColumn(%q, %d) = %d, want %d", tt.line, tt.index, got, tt.want)
			}
		})
	}
}

func TestIndexAtIsTheRuneNearestADisplayColumn(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want int
	}{
		{"the start", "\tab", 0, 0},
		{"the first half of a tab's gap", "\tab", 1, 0},
		{"the middle of a tab's gap goes after it", "\tab", 2, 1},
		{"the second half of a tab's gap", "\tab", 3, 1},
		{"the rune after a tab", "\tab", 4, 1},
		{"a later rune", "\tab", 5, 2},
		{"past the end is the end", "\tab", 40, 3},
		{"before the start is the start", "ab", -3, 0},
		{"no tabs", "abc", 2, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IndexAt([]rune(tt.line), tt.col); got != tt.want {
				t.Fatalf("IndexAt(%q, %d) = %d, want %d", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestIndexAtUndoesTextColumn(t *testing.T) {
	line := []rune("a\tbc\t\td\te")
	for i := 0; i <= len(line); i++ {
		if back := IndexAt(line, TextColumn(line, i)); back != i {
			t.Errorf("IndexAt(TextColumn(%d)) = %d", i, back)
		}
	}
}

func TestExpandTabsFillsEachTabWithSpacesToItsStop(t *testing.T) {
	tests := []struct {
		name string
		text string
		col  int
		want string
	}{
		{"a leading tab", "\tb", 0, "    b"},
		{"a tab inside a word", "ab\tc", 0, "ab  c"},
		{"text that starts mid-line", "\tc", 2, "  c"},
		{"text that starts at a stop", "\tc", 4, "    c"},
		{"no tabs is unchanged", "abc", 3, "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpandTabs(tt.text, tt.col); got != tt.want {
				t.Fatalf("ExpandTabs(%q, %d) = %q, want %q", tt.text, tt.col, got, tt.want)
			}
		})
	}
}
