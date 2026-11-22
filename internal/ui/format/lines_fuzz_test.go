package format

import (
	"bytes"
	"testing"
)

// Lines never panics, and a body of n newlines gives at most n+1 lines
// plus those the indenting adds, which are bounded by the body's size.
// The seeds are synthetic.
func FuzzLines(fuzz *testing.F) {
	fuzz.Add([]byte(`{"a":[1,2,{"b":null}],"c":"d\n"}`))
	fuzz.Add([]byte(`[]`))
	fuzz.Add([]byte("plain\ntext\n"))
	fuzz.Add([]byte(`{"a":`))
	fuzz.Add(bytes.Repeat([]byte("["), 600))
	fuzz.Fuzz(func(t *testing.T, body []byte) {
		lines := Lines(body)
		if len(lines) > 2*len(body)+1 {
			t.Fatalf("%d lines from %d bytes", len(lines), len(body))
		}
	})
}
