package ui

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

func TestFilteredLinesKeepTheirRealLineNumbers(t *testing.T) {
	lines := compactLines([]byte(compactBody))
	for i := range lines {
		if lines[i].Num != i+1 {
			t.Fatalf("line %d has Num %d: compact lines are numbered from the start", i, lines[i].Num)
		}
	}
	filtered := filterCompact(lines, "cy")
	if len(filtered) == 0 || len(filtered) >= len(lines) {
		t.Fatalf("filtered %d of %d", len(filtered), len(lines))
	}
	for _, l := range filtered {
		if l.Num == 0 || lineText(lines[l.Num-1]) != lineText(l) {
			t.Fatalf("filtered line %q claims number %d, which is %q", lineText(l), l.Num, lineText(lines[max(l.Num-1, 0)]))
		}
	}
	var v widget.CodeView
	v.SetLines(filtered)
	last := filtered[len(filtered)-1].Num
	if v.LineNumber(len(filtered)-1) != last || last <= len(filtered) {
		t.Fatalf("the last shown line is %d of the original, shown as %d", last, v.LineNumber(len(filtered)-1))
	}
}
