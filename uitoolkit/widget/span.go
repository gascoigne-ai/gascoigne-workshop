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
