package model

import (
	"reflect"
	"strings"
	"testing"
)

// Bulk text pasted into a table parses to at most one row per line, never
// panics, and formatting the rows and parsing them again gives the same rows.
func FuzzParseBulk(fuzz *testing.F) {
	fuzz.Add("q: gioui\n# off: x\nurl: https://a.test:8080/p?x=1\npad:   two spaces\nempty: ")
	fuzz.Add("a: 1\r\nb: 2\r\n")
	fuzz.Add("  k  : v\nlonely\nk: a: b")
	fuzz.Fuzz(func(t *testing.T, text string) {
		rows := ParseBulk(text, nil)
		if lines := strings.Count(text, "\n") + 1; len(rows) > lines {
			t.Fatalf("%d rows from %d lines", len(rows), lines)
		}
		if again := ParseBulk(FormatBulk(rows), nil); !reflect.DeepEqual(again, rows) {
			t.Fatalf("round trip changed %#v into %#v", rows, again)
		}
	})
}
