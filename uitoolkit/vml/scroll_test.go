package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// rows builds n clickable 20px rows that record their index when clicked.
func rows(n int, hits *[]int) []Item {
	var items []Item
	for i := 0; i < n; i++ {
		i := i
		items = append(items, box(H(20), OnClick(func() { *hits = append(*hits, i) })))
	}
	return items
}

// scroller is a Scroll with a fixed height.
func scroller(st *ScrollState, h int, items ...Item) *El {
	return Scroll(st, append([]Item{H(h)}, items...)...)
}

func wheel(c *widget.Ctx, x, y int, dy float64) {
	c.In = widget.Input{X: x, Y: y, Wheel: dy}
}

func TestScrollWheelMovesContentAndClampsAtBothEnds(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(scroller(st, 100, rows(10, &hits)...)) // 200px of rows in 100px
	c := testCtx(t)
	Layout(c, root, 200, 100)

	wheel(c, 50, 50, -1)
	Update(c, root)
	if st.Y <= 0 {
		t.Fatalf("wheel down must scroll, Y = %d", st.Y)
	}
	for range 50 {
		wheel(c, 50, 50, -1)
		Update(c, root)
	}
	Layout(c, root, 200, 100)
	if want := 10*20 - 100; st.Y != want {
		t.Fatalf("Y = %d, want the clamp %d", st.Y, want)
	}
	for range 50 {
		wheel(c, 50, 50, 1)
		Update(c, root)
	}
	if st.Y != 0 {
		t.Fatalf("Y = %d, want 0", st.Y)
	}
}

func TestScrolledContentIsClickableAtItsNewPosition(t *testing.T) {
	var hits []int
	st := &ScrollState{Y: 60} // rows 0-2 are scrolled out
	root := Col(scroller(st, 100, rows(10, &hits)...))
	c := testCtx(t)
	Layout(c, root, 200, 100)

	click(c, 50, 10) // the first visible row is now row 3
	Update(c, root)
	if len(hits) != 1 || hits[0] != 3 {
		t.Fatalf("hits = %v, want [3]", hits)
	}
}

func TestRowsScrolledOutOfViewDoNotReceiveClicksOutsideTheScroller(t *testing.T) {
	var hits, below []int
	st := &ScrollState{Y: 0}
	root := Col(
		scroller(st, 100, rows(10, &hits)...), // rows 5..9 extend below the viewport
		box(H(100), OnClick(func() { below = append(below, 1) })),
	)
	c := testCtx(t)
	Layout(c, root, 200, 200)

	click(c, 50, 150) // on the box under the scroller, where row 7 would be
	Update(c, root)
	if len(hits) != 0 || len(below) != 1 {
		t.Fatalf("hidden rows got %v, the box under got %v", hits, below)
	}
}

func TestShortContentDoesNotScroll(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(scroller(st, 100, rows(2, &hits)...))
	c := testCtx(t)
	Layout(c, root, 200, 100)
	wheel(c, 50, 50, -3)
	Update(c, root)
	if st.Y != 0 {
		t.Fatalf("Y = %d, want 0", st.Y)
	}
}

func TestWheelScrollsTheInnermostScroller(t *testing.T) {
	var hits []int
	outer, inner := &ScrollState{}, &ScrollState{}
	root := Col(scroller(outer, 150,
		scroller(inner, 100, rows(10, &hits)...),
		box(H(200)),
	))
	c := testCtx(t)
	Layout(c, root, 200, 150)

	wheel(c, 50, 50, -1) // over the inner scroller
	Update(c, root)
	if inner.Y <= 0 || outer.Y != 0 {
		t.Fatalf("inner %d outer %d: the innermost scroller takes the wheel", inner.Y, outer.Y)
	}

	wheel(c, 50, 130, -1) // below the inner one, over the outer's own content
	Update(c, root)
	if outer.Y <= 0 {
		t.Fatalf("outer %d: the wheel over the outer's content scrolls the outer", outer.Y)
	}
}

func TestScrollbarThumbDragMovesTheContent(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(scroller(st, 100, rows(20, &hits)...)) // 400px in 100px
	c := testCtx(t)
	Layout(c, root, 200, 100)

	off, length, ok := widget.Thumb(100, 400, 100, 0)
	if !ok {
		t.Fatal("a thumb is expected")
	}
	x := 200 - widget.ScrollbarW/2
	click(c, x, off+length/2)
	Update(c, root)
	c.In = widget.Input{X: x, Y: off + length/2 + 30, Held: true}
	Update(c, root)
	if st.Y <= 0 {
		t.Fatalf("Y = %d after dragging the thumb", st.Y)
	}
	c.In = widget.Input{X: x, Y: 5} // released
	Update(c, root)
	y := st.Y
	c.In = widget.Input{X: x, Y: 90, Held: true}
	Update(c, root)
	if st.Y != y {
		t.Fatal("a released thumb must not follow the pointer")
	}
	if len(hits) != 0 {
		t.Fatalf("dragging the bar must not click rows, hits %v", hits)
	}
}
