package widget

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestRowSpansCutsByRune(t *testing.T) {
	two := []Span{{Text: "ab", Style: core.Text}, {Text: "żółw", Style: core.Muted}}
	icon := []Span{{Text: "x", Style: core.Text}, {Text: "yz", Style: core.Muted, Icon: "link"}}
	tests := []struct {
		name     string
		spans    []Span
		from, to int
		want     []Span
	}{
		{"whole", two, 0, 6, two},
		{"ascii prefix", two, 0, 1, []Span{{Text: "a", Style: core.Text}}},
		{"cut inside the second span", two, 1, 4, []Span{{Text: "b", Style: core.Text}, {Text: "żó", Style: core.Muted}}},
		{"multibyte middle", two, 3, 5, []Span{{Text: "ół", Style: core.Muted}}},
		{"multibyte tail", two, 5, 6, []Span{{Text: "w", Style: core.Muted}}},
		{"keeps the icon", icon, 0, 3, icon},
		{"cut inside an icon span", icon, 1, 2, []Span{{Text: "y", Style: core.Muted, Icon: "link"}}},
		{"empty range", two, 2, 2, nil},
		{"no spans", nil, 0, 3, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendRowSpans(nil, tt.spans, tt.from, tt.to)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRowSpansAppendToTheGivenBuffer(t *testing.T) {
	buf := []Span{{Text: "stale"}}
	got := appendRowSpans(buf[:0], []Span{{Text: "abc"}}, 1, 3)
	if len(got) != 1 || got[0].Text != "bc" || &got[0] != &buf[0] {
		t.Fatalf("got %+v, want bc written into the buffer", got)
	}
}

func TestSpanStringCountsLikeRunes(t *testing.T) {
	cols := monoColumns(8)
	tests := []string{"", "abc", "a\tb", "\t\t", "żółw\tx", "ab\t\tcd\te"}
	for _, text := range tests {
		for col := range 10 {
			if got, want := cols.spanString(text, col), cols.span([]rune(text), 0, len([]rune(text)), col); got != want {
				t.Fatalf("spanString(%q, %d) = %d, want %d", text, col, got, want)
			}
		}
	}
}
