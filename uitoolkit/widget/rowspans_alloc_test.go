//go:build !race

package widget

import "testing"

// Each row of every frame cut its spans through rune slices and strings.
func TestRowSpansAllocateNothingIntoABuffer(t *testing.T) {
	spans := []Span{{Text: "    \"field\": "}, {Text: "\"żółw value\","}}
	buf := make([]Span, 0, 4)
	allocs := testing.AllocsPerRun(100, func() {
		buf = appendRowSpans(buf[:0], spans, 3, 18)
	})
	if allocs != 0 {
		t.Fatalf("%v allocations per cut, want 0", allocs)
	}
}

func TestSpanStringAllocatesNothing(t *testing.T) {
	cols := monoColumns(8)
	allocs := testing.AllocsPerRun(100, func() {
		cols.spanString("żółw\tvalue", 2)
	})
	if allocs != 0 {
		t.Fatalf("%v allocations per count, want 0", allocs)
	}
}
