// Package logic holds the decisions of the font list that need no toolkit:
// where it opens, which rows are built, which one is selected and what it says
// when it has no rows. Plain ints and strings in and out, tested without a
// display.
package logic

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/form"
)

const (
	ListW = 320 // the width of the font list
	ListH = 240 // the height of its scrolling part
	Gap   = 8   // kept free between the list and the window edge
	Drop  = 16  // how far below the pointer the list opens
)

// Opened is the open list: the setting it picks a family for and where it
// floats.
type Opened struct {
	Role form.FontRole
	X, Y int
}

// Place is where the list opens: below the pointer, centered on it, and kept
// inside the window with Gap to spare. rowH is the height of the search row.
func Place(pointerX, pointerY, windowW, windowH, rowH int) (x, y int) {
	x = min(max(pointerX-ListW/2, Gap), windowW-ListW-Gap)
	y = min(pointerY+Drop, windowH-ListH-rowH-Gap)
	return x, y
}

// OpenScroll is the scroll offset that puts the selected row in the middle of
// the view, so the family in use is on screen at once. rows counts every row,
// Built-in included.
func OpenScroll(selected, rows, rowH, viewH int) int {
	centered := selected*rowH - (viewH-rowH)/2
	return min(max(centered, 0), max(rows*rowH-viewH, 0))
}

// Visible is the range [first, last) of rows to build: those in view plus two,
// so a scroll step never shows an unbuilt row. Only they load their fonts.
func Visible(scrollY, rowH, count, viewH int) (first, last int) {
	first = min(max(scrollY/rowH, 0), max(count-1, 0))
	last = min(first+viewH/rowH+2, count)
	return first, last
}

// Selected is the row of the family in use among the unfiltered rows: 0 for
// Built-in, or for a name that is not installed, else the index of the name
// plus one.
func Selected(names []string, current string) int {
	for i, name := range names {
		if strings.EqualFold(name, current) {
			return i + 1
		}
	}
	return 0
}

// ShowsBuiltIn reports whether the Built-in row is listed: only while the
// search is empty, as it has no name to match.
func ShowsBuiltIn(query string) bool { return strings.TrimSpace(query) == "" }

// Note is what the list says in place of rows: that the scan is pending, or
// that the search matched nothing. "" when there are rows.
func Note(installed, matching int) string {
	if installed == 0 {
		return "Looking for installed fonts..."
	}
	if matching == 0 {
		return "No font matches"
	}
	return ""
}
