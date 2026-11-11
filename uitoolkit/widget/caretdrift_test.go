package widget

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Regression: the caret, the selection and the click mapping placed a
// column at col*charW, where charW is the width of one "0" rounded to whole
// pixels, while the text is drawn by the width of whole strings. The two
// disagree by a fraction of a pixel per character, so on a long line the
// caret ended up several characters away from where the text was.
func TestTheCaretStandsAtTheEndOfTheTextOnALongLine(t *testing.T) {
	c := testCtx(t)
	for _, scale := range []float64{1, 1.25, 1.5, 2} {
		if err := c.SetScale(scale); err != nil {
			t.Fatal(err)
		}
		line := strings.Repeat("0", 120)
		area := NewTextArea(line)
		area.Cursor = 120
		g := area.geometry(c, core.Rect{W: 2000, H: 200})
		caretX := g.colX(120)
		textEnd := c.Width(c.F.Mono, line)
		if d := caretX - textEnd; d < -1 || d > 1 {
			t.Errorf("scale %.2f: the caret is at %d but the text ends at %d (%d px apart)", scale, caretX, textEnd, d)
		}
	}
}

func TestClickingOnALongLineLandsOnTheCharacterUnderThePointer(t *testing.T) {
	c := testCtx(t)
	if err := c.SetScale(1.5); err != nil {
		t.Fatal(err)
	}
	area := NewTextArea(strings.Repeat("0123456789", 12))
	r := core.Rect{W: 2000, H: 200}
	g := area.geometry(c, r)
	for _, col := range []int{10, 50, 100, 119} {
		x := g.colX(col) + 1 // inside the left half of that character: the caret goes before it
		if got := g.colAt(x); got != col {
			t.Errorf("a click in the left half of column %d chose column %d", col, got)
		}
	}
}
