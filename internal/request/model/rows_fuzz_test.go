package model

import (
	"strings"
	"testing"
)

// Bulk text pasted into a table parses to at most one row per line and
// never panics.
func FuzzParseBulk(f *testing.F) {
	f.Add("q: gioui\n# off: x\nurl: https://a.test:8080/p?x=1\npad:   two spaces\nempty: ")
	f.Add("a: 1\r\nb: 2\r\n")
	f.Add("  k  : v\nlonely\nk: a: b")
	f.Fuzz(func(t *testing.T, text string) {
		rows := ParseBulk(text, nil)
		if lines := strings.Count(text, "\n") + 1; len(rows) > lines {
			t.Fatalf("%d rows from %d lines", len(rows), lines)
		}
	})
}
