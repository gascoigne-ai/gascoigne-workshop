package body

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// kindToken is the color of a kind of JSON span: the token named like the
// kind's style, which is how the response view colors it too.
func kindToken(kind format.Kind) core.Token { return core.Token(kind.Style()) }

// plainStyler draws a line in the plain text color.
func plainStyler(line string) []widget.Span {
	return []widget.Span{{Text: line, Style: kindToken(format.Plain)}}
}

// jsonStyler colors one line of a JSON body.
func jsonStyler(line string) []widget.Span {
	src := format.Tokenize(line)
	spans := make([]widget.Span, len(src))
	for i, token := range src {
		spans[i] = widget.Span{Text: token.Text, Style: kindToken(token.Kind)}
	}
	return spans
}

// withVariables wraps a line styler so every {{variable}} is drawn in the
// success color when vars define it and in the error color when not, over
// whatever color the base styler gave the text.
func withVariables(base func(string) []widget.Span, vars []request.KV) func(string) []widget.Span {
	return func(line string) []widget.Span {
		var out []widget.Span
		for _, span := range base(line) {
			out = append(out, splitSpanVars(span, vars)...)
		}
		return out
	}
}

// splitSpanVars cuts a span at its {{variables}} and colors those.
func splitSpanVars(span widget.Span, vars []request.KV) []widget.Span {
	if !strings.Contains(span.Text, "{{") {
		return []widget.Span{span}
	}
	return theme.VarSpans(span.Text, span.Style, vars)
}
