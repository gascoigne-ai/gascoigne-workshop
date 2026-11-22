package settingsui

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
