package widget

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// Span is a run of text colored by a theme token. A code view paints a span
// with an Icon as that icon in the span's cells, in the Style color, and
// keeps Text for search and copy; other widgets show only Text.
type Span struct {
	Text  string
	Style core.Token
	Icon  string
}

// Styler cuts a text into colored spans; nil or an empty result draws the
// text plain. What the text is depends on the widget that takes it: a
// field's whole text, a table cell's text, a text area's single line.
type Styler func(text string) []Span
