package view

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func drawn(t *testing.T, c *core.Ctx, root *El, w, h int) {
	t.Helper()
	Layout(c, root, w, h)
	Draw(c, root, ebiten.NewImage(w, h))
}

func TestHoverIsOnForElementsUnderThePointer(t *testing.T) {
	cell := box(W(100), HoverBg(core.Hover))
	root := Row(cell, box(W(100)))
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 10}
	drawn(t, c, root, 200, 50)
	if !cell.hover {
		t.Fatal("the pointer is over the cell")
	}
}

func TestAnOpenModalOverlayTurnsHoverOffBelowItButNotInsideIt(t *testing.T) {
	cell := box(W(100), HoverBg(core.Hover))
	item := box(W(60), H(20), HoverBg(core.Hover))
	menu := Col(W(60), H(40), FloatAt(120, 5), Modal(), OnDismiss(func() {}), item)
	root := Stack(Row(cell, box(W(100))), menu)
	c := testCtx(t)

	c.In = core.Input{X: 50, Y: 10} // over the cell below
	drawn(t, c, root, 200, 50)
	if cell.hover {
		t.Fatal("the cell under a modal overlay must not light up")
	}

	c.In = core.Input{X: 130, Y: 10} // over the overlay's own item
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
	cell := box(W(100), HoverBg(core.Hover))
	floating := Col(W(60), H(40), FloatAt(120, 5), box()) // not modal: a toast, say
	root := Stack(Row(cell), floating)
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 10}
	drawn(t, c, root, 200, 50)
	if !cell.hover {
		t.Fatal("a non-modal floating element must not block hover")
	}
}
