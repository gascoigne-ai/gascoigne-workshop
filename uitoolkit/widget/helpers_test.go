package widget

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

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

func drawn(t *testing.T, c *core.Ctx, root *El, w, h int) {
	t.Helper()
	Layout(c, root, w, h)
	Draw(c, root, ebiten.NewImage(w, h))
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

// sampleLines is this JSON, pretty printed, as lines (index: text):
// 0 {  1   "a": 1,  2   "items": [  3     {  4       "id": 2,
// 5       "o": {  6         "x": 3  7       }  8     },  9     {
// 10      "id": 4  11    }  12  ],  13  "z": 5  14 }
func sampleLines() []Line {
	texts := []struct {
		text     string
		foldable bool
	}{
		{"{", true}, {`  "a": 1,`, false}, {`  "items": [`, true}, {"    {", true},
		{`      "id": 2,`, false}, {`      "o": {`, true}, {`        "x": 3`, false},
		{"      }", false}, {"    },", false}, {"    {", true}, {`      "id": 4`, false},
		{"    }", false}, {"  ],", false}, {`  "z": 5`, false}, {"}", false},
	}
	lines := make([]Line, len(texts))
	for i, t := range texts {
		lines[i] = Line{Spans: []Span{{Text: t.text}}, Foldable: t.foldable}
	}
	return lines
}

type memClip struct{ text string }

func (m *memClip) Read() (string, error) { return m.text, nil }

func (m *memClip) Write(s string) error { m.text = s; return nil }

// viewport is a 600x400 code view at the origin of a context with a memory clipboard.
func viewport(t *testing.T) (*CodeView, *core.Ctx, *memClip, core.Rect) {
	t.Helper()
	c, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	clip := &memClip{}
	c.Clipboard = clip
	v := &CodeView{}
	v.SetLines(sampleLines())
	return v, c, clip, core.Rect{W: 600, H: 400}
}

// rowY is the pixel row of the middle of visible row i.
func rowY(c *core.Ctx, v *CodeView, r core.Rect, i int) int {
	g := v.geometry(c, r)
	return r.Y + i*g.lineH + g.lineH/2
}

// colX is the pixel x of the boundary before column col of the text.
func colX(c *core.Ctx, v *CodeView, r core.Rect, col int) int {
	g := v.geometry(c, r)
	return g.textX - v.ScrollX + g.colX(col)
}

func press(v *CodeView, c *core.Ctx, r core.Rect, x, y, clicks int) {
	c.In = core.Input{X: x, Y: y, Click: true, Held: true, Clicks: clicks}
	v.Update(c, r)
}

func dragTo(v *CodeView, c *core.Ctx, r core.Rect, x, y int) {
	c.In = core.Input{X: x, Y: y, Held: true}
	v.Update(c, r)
}

func release(v *CodeView, c *core.Ctx, r core.Rect, x, y int) {
	c.In = core.Input{X: x, Y: y}
	v.Update(c, r)
}

// shiftKey sends one Shift+movement key to a view whose pointer is away.
func shiftKey(v *CodeView, c *core.Ctx, r core.Rect, edit core.Edit) {
	c.In = core.Input{X: 5000, Y: 5000, Shift: true, Edits: []core.Edit{edit}}
	v.Update(c, r)
}

// clickAt puts the caret before column col of line and releases.
func clickAt(v *CodeView, c *core.Ctx, r core.Rect, line, col int) {
	press(v, c, r, colX(c, v, r, col), rowY(c, v, r, line), 1)
	release(v, c, r, 0, 0)
}
