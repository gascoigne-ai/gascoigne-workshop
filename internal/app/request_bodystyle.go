package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/ui/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/jsonview"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// plainStyler draws a line in the plain text color.
func plainStyler(line string) []Span {
	return []Span{{Text: line, Style: respview.Tokens[jsonview.Plain]}}
}

// withVariables wraps a line styler so every {{variable}} is drawn in the
// success color when req resolves it and in the error color when it does
// not, over whatever color the base styler gave the text.
func (a *App) withVariables(base func(string) []Span, req *request.Request) func(string) []Span {
	vars := a.varsFor(req)
	return func(line string) []Span {
		spans := base(line)
		var out []Span
		for _, sp := range spans {
			out = append(out, splitSpanVars(sp, vars)...)
		}
		return out
	}
}

// splitSpanVars cuts a span at its {{variables}} and colors those.
func splitSpanVars(sp Span, vars []request.KV) []Span {
	if !containsVar(sp.Text) {
		return []Span{sp}
	}
	var out []Span
	for _, seg := range request.SplitVars(sp.Text) {
		style := sp.Style
		if seg.IsVar {
			style = w.OK
			if !request.Defined(varName(seg.Text), vars) {
				style = w.Error
			}
		}
		out = append(out, Span{Text: seg.Text, Style: style})
	}
	return out
}

func containsVar(text string) bool {
	for i := 0; i+1 < len(text); i++ {
		if text[i] == '{' && text[i+1] == '{' {
			return true
		}
	}
	return false
}

// varName is the name inside {{name}}.
func varName(text string) string {
	if len(text) < 4 {
		return ""
	}
	return text[2 : len(text)-2]
}

// cellStyler colors the {{variables}} in a table cell the way the URL
// field does: green when req resolves them, red when not. Cells without a
// variable are drawn as they are.
func (a *App) cellStyler(req *request.Request) func(string) []Span {
	vars := a.varsFor(req)
	return func(text string) []Span {
		if !containsVar(text) {
			return nil
		}
		var out []Span
		for _, seg := range request.SplitVars(text) {
			sp := Span{Text: seg.Text}
			if seg.IsVar {
				sp.Style = w.OK
				if !request.Defined(varName(seg.Text), vars) {
					sp.Style = w.Error
				}
			}
			out = append(out, sp)
		}
		return out
	}
}
