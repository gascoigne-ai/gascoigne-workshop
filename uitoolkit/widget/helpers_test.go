package widget

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	c, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func click(c *core.Ctx, x, y int) { c.In = core.Input{X: x, Y: y, Click: true} }

// props is the first line of the element's Dump: its own attributes
// without its children, for asserting what a widget set.
func props(element *El) string {
	first, _, _ := strings.Cut(Dump(element), "\n")
	return first
}

// hasProps fails unless the element's own attributes include every part.
func hasProps(t *testing.T, element *El, parts ...string) {
	t.Helper()
	line := props(element)
	for _, part := range parts {
		if !strings.Contains(line, part) {
			t.Errorf("%q is missing from %q", part, line)
		}
	}
}

type fakeClipboard struct {
	text string
	err  error
}

func (f *fakeClipboard) Read() (string, error) { return f.text, f.err }
func (f *fakeClipboard) Write(text string) error {
	if f.err != nil {
		return f.err
	}
	f.text = text
	return nil
}

func contains(text, part string) bool { return strings.Contains(text, part) }
