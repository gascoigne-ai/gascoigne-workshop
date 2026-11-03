package ui

import "testing"

func TestComputeCoversWindowWithoutOverlap(t *testing.T) {
	for _, size := range [][2]int{{1280, 800}, {900, 600}, {300, 200}, {10, 10}} {
		w, h := size[0], size[1]
		l := Compute(w, h)
		regions := []Rect{l.SidebarHead, l.Tree, l.Header, l.Tabs, l.URL, l.Request, l.Response, l.Status}
		area := 0
		for i, r := range regions {
			if r.W < 0 || r.H < 0 || r.X < 0 || r.Y < 0 || r.X+r.W > w || r.Y+r.H > h {
				t.Errorf("%dx%d: region %d out of window: %+v", w, h, i, r)
			}
			area += r.W * r.H
			for j := i + 1; j < len(regions); j++ {
				if overlaps(r, regions[j]) {
					t.Errorf("%dx%d: regions %d and %d overlap: %+v %+v", w, h, i, j, r, regions[j])
				}
			}
		}
		if area != w*h {
			t.Errorf("%dx%d: regions cover %d pixels, want %d", w, h, area, w*h)
		}
	}
}

func overlaps(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H && a.W > 0 && a.H > 0 && b.W > 0 && b.H > 0
}

func TestSplitsClamp(t *testing.T) {
	r := Rect{10, 20, 100, 50}
	tests := []struct {
		name       string
		got1, got2 Rect
		want1      Rect
		want2      Rect
	}{
		{"left", first(r.SplitLeft(30)), second(r.SplitLeft(30)), Rect{10, 20, 30, 50}, Rect{40, 20, 70, 50}},
		{"left too big", first(r.SplitLeft(500)), second(r.SplitLeft(500)), r, Rect{110, 20, 0, 50}},
		{"top negative", first(r.SplitTop(-5)), second(r.SplitTop(-5)), Rect{10, 20, 100, 0}, r},
		{"bottom", first(r.SplitBottom(20)), second(r.SplitBottom(20)), Rect{10, 20, 100, 30}, Rect{10, 50, 100, 20}},
		{"right", first(r.SplitRight(25)), second(r.SplitRight(25)), Rect{10, 20, 75, 50}, Rect{85, 20, 25, 50}},
	}
	for _, tt := range tests {
		if tt.got1 != tt.want1 || tt.got2 != tt.want2 {
			t.Errorf("%s: got %+v %+v, want %+v %+v", tt.name, tt.got1, tt.got2, tt.want1, tt.want2)
		}
	}
}

func first(a, _ Rect) Rect  { return a }
func second(_, b Rect) Rect { return b }

func TestContainsAndInset(t *testing.T) {
	r := Rect{10, 10, 20, 20}
	for _, tt := range []struct {
		x, y int
		want bool
	}{{10, 10, true}, {29, 29, true}, {30, 10, false}, {9, 15, false}} {
		if got := r.Contains(tt.x, tt.y); got != tt.want {
			t.Errorf("Contains(%d,%d) = %v", tt.x, tt.y, got)
		}
	}
	if got := r.Inset(30); got.W != 0 || got.H != 0 {
		t.Errorf("Inset beyond size = %+v", got)
	}
}
