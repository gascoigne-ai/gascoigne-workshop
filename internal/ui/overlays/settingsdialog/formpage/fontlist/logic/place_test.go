package logic

import "testing"

func TestPlaceCentersTheListOnThePointerAndKeepsItInTheWindow(t *testing.T) {
	cases := []struct {
		name               string
		pointerX, pointerY int
		windowW, windowH   int
		wantX, wantY       int
	}{
		{"centered below the pointer", 400, 300, 1280, 800, 400 - ListW/2, 316},
		{"pushed in from the left edge", 10, 300, 1280, 800, Gap, 316},
		{"pushed in from the right edge", 1270, 300, 1280, 800, 1280 - ListW - Gap, 316},
		{"lifted from the bottom edge", 400, 790, 1280, 800, 400 - ListW/2, 800 - ListH - 24 - Gap},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := Place(tc.pointerX, tc.pointerY, tc.windowW, tc.windowH, 24)
			if x != tc.wantX || y != tc.wantY {
				t.Fatalf("got (%d, %d), want (%d, %d)", x, y, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestOpenScrollCentersTheSelectedRowWithinTheScrollableRange(t *testing.T) {
	cases := []struct {
		name                        string
		selected, rows, rowH, viewH int
		want                        int
	}{
		{"first row stays at the top", 0, 300, 24, 240, 0},
		{"a middle row is centered", 200, 301, 24, 240, 200*24 - (240-24)/2},
		{"the last row stops at the bottom", 300, 301, 24, 240, 301*24 - 240},
		{"a list shorter than the view does not scroll", 2, 3, 24, 240, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := OpenScroll(tc.selected, tc.rows, tc.rowH, tc.viewH); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestVisibleBuildsTheRowsInViewPlusTwo(t *testing.T) {
	cases := []struct {
		name                        string
		scrollY, rowH, count, viewH int
		first, last                 int
	}{
		{"top of a long list", 0, 24, 301, 240, 0, 12},
		{"scrolled", 4800, 24, 301, 240, 200, 212},
		{"short list ends at its count", 0, 24, 3, 240, 0, 3},
		{"empty list", 0, 24, 0, 240, 0, 0},
		{"scrolled past the end is clamped", 99999, 24, 301, 240, 300, 301},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first, last := Visible(tc.scrollY, tc.rowH, tc.count, tc.viewH)
			if first != tc.first || last != tc.last {
				t.Fatalf("got [%d, %d), want [%d, %d)", first, last, tc.first, tc.last)
			}
		})
	}
}

func TestSelectedIsTheRowOfTheFamilyInUseIgnoringCase(t *testing.T) {
	names := []string{"Go", "Go Mono"}
	cases := []struct {
		current string
		want    int
	}{
		{"", 0},
		{"go mono", 2},
		{"Go", 1},
		{"Not Installed", 0},
	}
	for _, tc := range cases {
		if got := Selected(names, tc.current); got != tc.want {
			t.Errorf("Selected(%q) = %d, want %d", tc.current, got, tc.want)
		}
	}
}

func TestBuiltInShowsUnlessTheSearchHasText(t *testing.T) {
	cases := map[string]bool{"": true, "   ": true, "mono": false, " mono ": false}
	for query, want := range cases {
		if got := ShowsBuiltIn(query); got != want {
			t.Errorf("ShowsBuiltIn(%q) = %v, want %v", query, got, want)
		}
	}
}

func TestNoteSaysWhyThereAreNoRows(t *testing.T) {
	cases := []struct {
		name                string
		installed, matching int
		want                string
	}{
		{"scan pending", 0, 0, "Looking for installed fonts..."},
		{"nothing matches", 5, 0, "No font matches"},
		{"rows shown", 5, 2, ""},
	}
	for _, tc := range cases {
		if got := Note(tc.installed, tc.matching); got != tc.want {
			t.Errorf("%s: Note(%d, %d) = %q, want %q", tc.name, tc.installed, tc.matching, got, tc.want)
		}
	}
}
