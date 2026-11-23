package theme

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// VarSpans cuts text at its {{variables}}: a variable is drawn in the
// success color when vars defines it and in the error color when not, the
// text around it in base.
func VarSpans(text string, base core.Token, vars []request.KV) []widget.Span {
	var spans []widget.Span
	for _, segment := range request.SplitVars(text) {
		style := base
		if segment.IsVar {
			style = varToken(segment.Text, vars)
		}
		spans = append(spans, widget.Span{Text: segment.Text, Style: style})
	}
	return spans
}

// varToken is the color of the variable run "{{name}}".
func varToken(run string, vars []request.KV) core.Token {
	name := strings.TrimSuffix(strings.TrimPrefix(run, "{{"), "}}")
	if request.Defined(name, vars) {
		return core.OK
	}
	return core.Error
}
