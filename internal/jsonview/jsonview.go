// Package jsonview turns a response body into display lines: pretty
// printed JSON split into colorable spans, or plain text lines when the
// body is not JSON.
package jsonview

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Kind classifies a span for coloring.
type Kind int

const (
	Plain Kind = iota
	Punct
	Key
	String
	Number
	Literal // true, false, null
)

// Span is a run of text of one Kind.
type Span struct {
	Text string
	Kind Kind
}

// Line is one display line. Foldable lines open an object or array.
type Line struct {
	Spans    []Span
	Foldable bool
}

// Lines pretty-prints body when it is valid JSON and splits it into
// lines; otherwise it returns the body's own lines as Plain spans.
func Lines(body []byte) []Line {
	var buf bytes.Buffer
	if json.Valid(body) && json.Indent(&buf, body, "", "  ") == nil {
		return tokenize(buf.String())
	}
	return plain(string(body))
}

func plain(text string) []Line {
	var lines []Line
	for _, l := range strings.Split(text, "\n") {
		lines = append(lines, Line{Spans: []Span{{Text: strings.TrimRight(l, "\r")}}})
	}
	return lines
}

func tokenize(text string) []Line {
	var lines []Line
	for _, l := range strings.Split(text, "\n") {
		spans := tokenizeLine(l)
		last := strings.TrimRight(l, " ,")
		foldable := strings.HasSuffix(last, "{") || strings.HasSuffix(last, "[")
		lines = append(lines, Line{Spans: spans, Foldable: foldable})
	}
	return lines
}

// tokenizeLine splits one line of json.Indent output. Strings never
// span lines there, so a single left-to-right pass is enough.
func tokenizeLine(l string) []Span {
	var spans []Span
	add := func(text string, k Kind) { spans = append(spans, Span{Text: text, Kind: k}) }
	for i := 0; i < len(l); {
		c := l[i]
		switch {
		case c == '"':
			end := stringEnd(l, i)
			kind := String
			if strings.HasPrefix(strings.TrimLeft(l[end:], " "), ":") {
				kind = Key
			}
			add(l[i:end], kind)
			i = end
		case strings.IndexByte("{}[],: ", c) >= 0:
			j := i + 1
			for j < len(l) && strings.IndexByte("{}[],: ", l[j]) >= 0 && (c == ' ') == (l[j] == ' ') {
				j++
			}
			kind := Punct
			if c == ' ' {
				kind = Plain
			}
			add(l[i:j], kind)
			i = j
		default:
			j := i
			for j < len(l) && strings.IndexByte("{}[],: ", l[j]) < 0 {
				j++
			}
			kind := Number
			if w := l[i:j]; w == "true" || w == "false" || w == "null" {
				kind = Literal
			}
			add(l[i:j], kind)
			i = j
		}
	}
	return spans
}

// stringEnd returns the index just past the closing quote of the string
// starting at l[start].
func stringEnd(l string, start int) int {
	for i := start + 1; i < len(l); i++ {
		switch l[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return len(l)
}
