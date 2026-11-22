package settingsui

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// testCtx is the context the views are built with in tests: the default
// faces, so the sizes are the default ones.
func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// findAll is every element under root whose own line of the dump holds all of
// parts.
func findAll(root *El, parts ...string) []*El {
	var found []*El
	line, _, _ := strings.Cut(Dump(root), "\n")
	matches := true
	for _, part := range parts {
		matches = matches && strings.Contains(line, part)
	}
	if matches {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findAll(child, parts...)...)
	}
	return found
}
