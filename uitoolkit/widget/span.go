package widget

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// Span is a run of text colored by a theme token.
type Span struct {
	Text  string
	Style core.Token
}
