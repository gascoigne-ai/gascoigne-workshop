package logic

import "testing"

func TestRowGeometryRange(t *testing.T) {
	tests := []struct {
		name    string
		geom    RowGeometry
		scrollY int
		want    RowRange
	}{
		{"empty", RowGeometry{Count: 0, RowH: 28, ViewH: 400}, 0, RowRange{0, 0}},
		{"fits", RowGeometry{Count: 5, RowH: 28, ViewH: 400}, 0, RowRange{0, 5}},
		{"top", RowGeometry{Count: 1000, RowH: 28, ViewH: 400}, 0, RowRange{0, 16}},
		{"middle", RowGeometry{Count: 1000, RowH: 28, ViewH: 400}, 28 * 500, RowRange{500, 516}},
		{"not a multiple of the row height", RowGeometry{Count: 1000, RowH: 28, ViewH: 400}, 28*500 + 27, RowRange{500, 516}},
		{"bottom clamps", RowGeometry{Count: 1000, RowH: 28, ViewH: 400}, 28 * 985, RowRange{985, 1000}},
		{"scroll past the content", RowGeometry{Count: 1000, RowH: 28, ViewH: 400}, 28 * 5000, RowRange{985, 1000}},
		{"negative scroll", RowGeometry{Count: 1000, RowH: 28, ViewH: 400}, -50, RowRange{0, 16}},
		{"no view height builds all", RowGeometry{Count: 1000, RowH: 28, ViewH: 0}, 28 * 500, RowRange{0, 1000}},
		{"no row height builds all", RowGeometry{Count: 1000, RowH: 0, ViewH: 400}, 28 * 500, RowRange{0, 1000}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.geom.Range(test.scrollY)
			if got != test.want {
				t.Errorf("Range(%d) of %+v = %+v, want %+v", test.scrollY, test.geom, got, test.want)
			}
			if got.First < 0 || got.First > got.Last || got.Last > test.geom.Count {
				t.Errorf("%+v breaks 0 <= First <= Last <= Count (%d)", got, test.geom.Count)
			}
		})
	}
}
