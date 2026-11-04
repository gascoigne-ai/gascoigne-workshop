package vml

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

func drawn(t *testing.T, c *widget.Ctx, root *El, w, h int) {
	t.Helper()
	Layout(c, root, w, h)
	Draw(c, root, ebiten.NewImage(w, h))
}

func TestHoverIsOnForElementsUnderThePointer(t *testing.T) {
	cell := box(W(100), HoverBg(widget.Hover))
	root := Row(cell, box(W(100)))
	c := testCtx(t)
	c.In = widget.Input{X: 50, Y: 10}
	drawn(t, c, root, 200, 50)
	if !cell.hover {
		t.Fatal("the pointer is over the cell")
	}
}

func TestAnOpenModalOverlayTurnsHoverOffBelowItButNotInsideIt(t *testing.T) {
	cell := box(W(100), HoverBg(widget.Hover))
	item := box(W(60), H(20), HoverBg(widget.Hover))
	menu := Col(W(60), H(40), FloatAt(120, 5), Modal(func() {}), item)
	root := Stack(Row(cell, box(W(100))), menu)
	c := testCtx(t)

	c.In = widget.Input{X: 50, Y: 10} // over the cell below
	drawn(t, c, root, 200, 50)
	if cell.hover {
		t.Fatal("the cell under a modal overlay must not light up")
	}

	c.In = widget.Input{X: 130, Y: 10} // over the overlay's own item
	drawn(t, c, root, 200, 50)
	if !item.hover {
		t.Fatal("items of the overlay itself do light up")
	}
	if cell.hover {
		t.Fatal("and the cell below still does not")
	}
	if c.HoverOff {
		t.Fatal("HoverOff must not stay set after drawing")
	}
}

func TestWithoutAModalOverlayHoverWorksEverywhere(t *testing.T) {
	cell := box(W(100), HoverBg(widget.Hover))
	floating := Col(W(60), H(40), FloatAt(120, 5), box()) // not modal: a toast, say
	root := Stack(Row(cell), floating)
	c := testCtx(t)
	c.In = widget.Input{X: 50, Y: 10}
	drawn(t, c, root, 200, 50)
	if !cell.hover {
		t.Fatal("a non-modal floating element must not block hover")
	}
}

func TestAModalOverlaySwallowsTheWheelAndTheCursorBelowIt(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	split := &SplitState{}
	first, second := box(), box()
	menu := box(W(40), H(40), FloatAt(160, 0), Modal(func() {}))
	root := Stack(
		Col(
			scroller(st, 100, rows(10, &hits)...),
			Split(split, 100, H(50), first, second),
		),
		menu,
	)
	c := testCtx(t)
	Layout(c, root, 200, 200)

	c.In = widget.Input{X: 50, Y: 50, Wheel: -3} // wheel over the scroller, outside the menu
	Update(c, root)
	if st.Y != 0 {
		t.Fatalf("the scroller below a modal moved to %d", st.Y)
	}

	c.Cursor = widget.CursorDefault
	c.In = widget.Input{X: 100, Y: 125} // on the split divider, outside the menu
	Update(c, root)
	if c.Cursor != widget.CursorDefault {
		t.Fatalf("the divider below a modal asked for cursor %v", c.Cursor)
	}
}

func TestWheelStillScrollsWhenNothingModalIsOpen(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Stack(scroller(st, 100, rows(20, &hits)...), box(W(40), H(40), FloatAt(160, 0))) // a non-modal float
	c := testCtx(t)
	Layout(c, root, 200, 200)
	c.In = widget.Input{X: 50, Y: 50, Wheel: -3}
	Update(c, root)
	if st.Y == 0 {
		t.Fatal("the scroller must scroll")
	}
}
