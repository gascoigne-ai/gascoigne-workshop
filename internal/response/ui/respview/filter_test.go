package respview

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"testing"
)

func TestFilteredLinesKeepTheirRealLineNumbers(t *testing.T) {
	lines := CompactLines([]byte(compactBody))
	for i := range lines {
		if lines[i].Num != i+1 {
			t.Fatalf("line %d has Num %d: compact lines are numbered from the start", i, lines[i].Num)
		}
	}
	filtered := Filter(lines, "cy")
	if len(filtered) == 0 || len(filtered) >= len(lines) {
		t.Fatalf("filtered %d of %d", len(filtered), len(lines))
	}
	for _, l := range filtered {
		if l.Num == 0 || lineText(lines[l.Num-1]) != lineText(l) {
			t.Fatalf("filtered line %q claims number %d, which is %q", lineText(l), l.Num, lineText(lines[max(l.Num-1, 0)]))
		}
	}
	var v view.CodeView
	v.SetLines(filtered)
	last := filtered[len(filtered)-1].Num
	if v.LineNumber(len(filtered)-1) != last || last <= len(filtered) {
		t.Fatalf("the last shown line is %d of the original, shown as %d", last, v.LineNumber(len(filtered)-1))
	}
}

const compactBody = `{"total":3,"items":[` +
	`{"id":1,"name":"alpha","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"ann","site":"https://a.example.org/very/long/path/to/ann"}},` +
	`{"id":2,"name":"beta","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"bob","site":"https://b.example.org/very/long/path/to/bob"}},` +
	`{"id":3,"name":"gamma","tags":["cyan","magenta","teal","olive","navy","maroon","lime","aqua"],"owner":{"login":"cy","site":"https://c.example.org/very/long/path/to/cy"}}]}`
