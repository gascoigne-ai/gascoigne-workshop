package core

import "strings"

// TabStop is the number of columns between tab stops: a tab in monospace
// text advances to the next multiple of it, and in proportional text it is
// a gap as wide as this many spaces of the face. The fonts have no glyph
// for U+0009, so a tab drawn as is shows the missing-glyph box one narrow
// cell wide. Tabs are expanded for display only: the text keeps the real
// tab, so copy, paste and saved files are unchanged. 4 is the common
// editor default.
const TabStop = 4

// ColumnAfter is the display column after char drawn at column col: the
// next tab stop for a tab, the next column for any other rune. It is the
// one rule every monospace surface counts columns by, so drawing, the caret,
// selection, hit-testing, wrapping and the horizontal extent agree.
func ColumnAfter(col int, char rune) int {
	if char == '\t' {
		return (col/TabStop + 1) * TabStop
	}
	return col + 1
}

// TextColumn is the display column of rune index in line, counted from the
// start of the line with tabs expanded (an index past the end is the end).
func TextColumn(line []rune, index int) int {
	col := 0
	for _, char := range line[:min(max(index, 0), len(line))] {
		col = ColumnAfter(col, char)
	}
	return col
}

// IndexAt is the inverse of TextColumn for hit-testing: the rune boundary in
// line nearest display column col. A column inside a tab's gap goes before
// the tab in the first half of the gap and after it from the middle on, as
// a click on any character does.
func IndexAt(line []rune, col int) int {
	left := 0
	for i, char := range line {
		right := ColumnAfter(left, char)
		if 2*col < left+right {
			return i
		}
		left = right
	}
	return len(line)
}

// ExpandTabs is text as drawn when it starts at display column col of its
// line: each tab replaced by the spaces up to its stop.
func ExpandTabs(text string, col int) string {
	if !strings.ContainsRune(text, '\t') {
		return text
	}
	var out strings.Builder
	for _, char := range text {
		next := ColumnAfter(col, char)
		if char == '\t' {
			out.WriteString(strings.Repeat(" ", next-col))
		} else {
			out.WriteRune(char)
		}
		col = next
	}
	return out.String()
}

// tabsAsGaps is proportional text as measured and drawn: each tab a gap of
// TabStop spaces of the face, since a proportional face has no columns to
// stop at.
func tabsAsGaps(text string) string {
	if !strings.ContainsRune(text, '\t') {
		return text
	}
	return strings.ReplaceAll(text, "\t", strings.Repeat(" ", TabStop))
}
