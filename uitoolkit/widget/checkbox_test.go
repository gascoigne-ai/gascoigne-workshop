package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestCheckboxOnChangeRunsWhenTheBoxIsClicked(t *testing.T) {
	var changes int
	root := Row(Checkbox(true, W(20), H(20), OnChange(func() { changes++ })))
	c := testCtx(t)
	Layout(c, root, 100, 20)
	c.In = core.Input{X: 60, Y: 10, Click: true} // beside the box
	Update(c, root)
	c.In = core.Input{X: 10, Y: 10, Click: true}
	Update(c, root)
	if changes != 1 {
		t.Fatalf("OnChange ran %d times, want 1", changes)
	}
}

func TestCheckboxIsA14PixelSquare(t *testing.T) {
	box := Checkbox(false)
	Layout(testCtx(t), Row(AlignItems(AlignStart), box), 100, 50)
	if got := box.Rect(); got.W != 14 || got.H != 14 {
		t.Fatalf("size %dx%d, want 14x14", got.W, got.H)
	}
}
