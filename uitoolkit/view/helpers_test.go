package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
)

// Helpers shared by the tests of this package: a context to lay out and
// update views with, a view of sample lines with the pointer helpers that
// press, drag and release on it, and the keyboard helpers for a focused
// code view. A helper used by one test file stays in that file.

func testCtx(t *testing.T) *core.Ctx {
	t.Helper()
	c, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func box(items ...Item) *El { return newEl(plain{}, items) }

func click(c *core.Ctx, x, y int) { c.In = core.Input{X: x, Y: y, Click: true} }

type memClip struct{ text string }

func (m *memClip) Read() (string, error) { return m.text, nil }

func (m *memClip) Write(s string) error { m.text = s; return nil }
