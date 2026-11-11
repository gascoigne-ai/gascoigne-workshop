package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// cursorProbe asks for a resize cursor while the pointer is over it.
type cursorProbe struct{ Base }

func (*cursorProbe) Update(e *El, c *core.Ctx) {
	if c.In.Over(e.Rect()) {
		c.Cursor = core.CursorResizeH
	}
}

func TestAModalOverlaySwallowsTheWheelAndTheCursorBelowIt(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	menu := Row(W(40), H(40), FloatAt(160, 0), Modal(), OnDismiss(func() {}))
	root := Stack(
		Col(
			scroller(st, 100, rows(10, &hits)...),
			Custom(&cursorProbe{}, H(50)), // wants a resize cursor while the pointer is on it, as a divider does
		),
		menu,
	)
	c := testCtx(t)
	Layout(c, root, 200, 200)

	c.In = core.Input{X: 50, Y: 50, Wheel: -3} // wheel over the scroller, outside the menu
	Update(c, root)
	if st.Y != 0 {
		t.Fatalf("the scroller below a modal moved to %d", st.Y)
	}

	c.Cursor = core.CursorDefault
	c.In = core.Input{X: 100, Y: 125} // on the divider, outside the menu
	Update(c, root)
	if c.Cursor != core.CursorDefault {
		t.Fatalf("the divider below a modal asked for cursor %v", c.Cursor)
	}
}

func TestWheelStillScrollsWhenNothingModalIsOpen(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Stack(scroller(st, 100, rows(20, &hits)...), Row(W(40), H(40), FloatAt(160, 0))) // a non-modal float
	c := testCtx(t)
	Layout(c, root, 200, 200)
	c.In = core.Input{X: 50, Y: 50, Wheel: -3}
	Update(c, root)
	if st.Y == 0 {
		t.Fatal("the scroller must scroll")
	}
}
