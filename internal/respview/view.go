// Package respview turns a response body into the colored, foldable
// lines the code viewer shows: pretty-printed, compact or raw, and
// filters the compact lines for a search.
package respview

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Tokens colors the spans of a JSON response.
var Tokens = map[jsonview.Kind]core.Token{
	jsonview.Plain:   "json.plain",
	jsonview.Punct:   "json.punct",
	jsonview.Key:     "json.key",
	jsonview.String:  "json.string",
	jsonview.Number:  "json.number",
	jsonview.Literal: "json.literal",
}

// CodeLines turns a response body into colored code-view lines.
func CodeLines(body []byte) []core.Line {
	src := jsonview.Lines(body)
	lines := make([]core.Line, len(src))
	for i, l := range src {
		spans := make([]core.Span, len(l.Spans))
		for j, s := range l.Spans {
			spans[j] = core.Span{Text: s.Text, Style: Tokens[s.Kind]}
		}
		lines[i] = core.Line{Spans: spans, Foldable: l.Foldable}
	}
	return lines
}

// CompactWidth is how wide a block may be and still stay on one line in
// the compact view.
const CompactWidth = 90

// CompactFoldDepth is the nesting level from which blocks start folded
// in the compact view: the top level and its children are open.
const CompactFoldDepth = 2

// CompactLines is the response as compact JSON lines; the opening line of
// every block carries its one-line preview for when it is folded.
func CompactLines(body []byte) []core.Line {
	src := jsonview.Compact(body, CompactWidth)
	lines := make([]core.Line, len(src))
	for i, l := range src {
		lines[i] = core.Line{Spans: toSpans(l.Spans), Foldable: l.Foldable, FoldedSpans: toSpans(l.Folded), Depth: l.Depth, Num: i + 1}
	}
	return lines
}

func toSpans(src []jsonview.Span) []core.Span {
	if src == nil {
		return nil
	}
	spans := make([]core.Span, len(src))
	for j, s := range src {
		spans[j] = core.Span{Text: s.Text, Style: Tokens[s.Kind]}
	}
	return spans
}

// FoldDeep folds every block that opens at nesting depth or deeper.
func FoldDeep(v *core.CodeView, depth int) {
	for i, l := range v.Lines {
		if l.Foldable && l.Depth >= depth {
			v.Folded[i] = true
		}
	}
}

// LooksBinary reports whether a body is not text: it has a NUL byte or
// is not valid UTF-8 (judged on the first 8 KiB).
func LooksBinary(body []byte) bool {
	sample := body
	if len(body) > binarySample {
		sample = body[:binarySample]
		for i := 0; i < utf8.UTFMax-1 && !utf8.Valid(sample); i++ {
			sample = sample[:len(sample)-1] // the cut may split a character
		}
	}
	return bytes.IndexByte(sample, 0) >= 0 || !utf8.Valid(sample)
}

const binarySample = 8 << 10

// RawLines shows a body exactly as received, one plain line per line.
func RawLines(body []byte) []core.Line {
	texts := strings.Split(string(body), "\n")
	lines := make([]core.Line, len(texts))
	for i, text := range texts {
		lines[i] = core.Line{Spans: []core.Span{{Text: strings.TrimRight(text, "\r"), Style: Tokens[jsonview.Plain]}}}
	}
	return lines
}
