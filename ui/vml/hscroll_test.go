package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// strip is an HScroll 100px wide with n 40px boxes that record clicks.
func strip(st *ScrollState, n int, hits *[]int) *El {
	var items []Item
	for i := 0; i < n; i++ {
		i := i
		items = append(items, box(W(40), OnClick(func() { *hits = append(*hits, i) })))
	}
	return HScroll(st, append([]Item{W(100), H(20)}, items...)...)
}

func TestHScrollWheelScrollsSidewaysAndClamps(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(strip(st, 5, &hits)) // 200px of boxes in 100px
	c := testCtx(t)
	Layout(c, root, 300, 100)
	for range 20 {
		wheel(c, 50, 10, -1)
		Update(c, root)
		Layout(c, root, 300, 100)
	}
	if st.X != 136 { // content 200 + an arrow's room at each end - the box 100
		t.Fatalf("X = %d, want the clamp 136", st.X)
	}
	for range 20 {
		wheel(c, 50, 10, 1)
		Update(c, root)
	}
	if st.X != 0 {
		t.Fatalf("X = %d, want 0", st.X)
	}
}

func TestHScrollThatFitsDoesNotScrollAndLeavesWheelAlone(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(strip(st, 2, &hits)) // 80px in 100px
	c := testCtx(t)
	Layout(c, root, 300, 100)
	wheel(c, 50, 10, -1)
	Update(c, root)
	if st.X != 0 || c.In.Wheel == 0 {
		t.Fatalf("X = %d, wheel left = %v", st.X, c.In.Wheel)
	}
}

func TestHScrolledContentIsClickableAtItsNewPosition(t *testing.T) {
	var hits []int
	st := &ScrollState{X: 100}
	root := Col(strip(st, 5, &hits))
	c := testCtx(t)
	Layout(c, root, 300, 100)
	// Box 4 is the last 40px; with X=100 it sits at 60..100. The arrows
	// cover the edges, so click the middle of box 3 (20..60 -> 40).
	c.In = widget.Input{X: 40, Y: 10, Click: true, Held: true}
	Update(c, root)
	c.In = widget.Input{X: 40, Y: 10}
	Update(c, root)
	if len(hits) != 1 || hits[0] != 3 {
		t.Fatalf("hits %v, want box 3", hits)
	}
}

func TestHScrollArrowsStepTheStrip(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(strip(st, 5, &hits))
	c := testCtx(t)
	Layout(c, root, 300, 100)
	right := widget.Rect{X: 100 - hscrollArrowW, Y: 0, W: hscrollArrowW, H: 20}
	c.In = widget.Input{X: right.X + 2, Y: 10, Click: true, Held: true}
	Update(c, root)
	if st.X <= 0 {
		t.Fatalf("the right arrow must scroll right, X = %d", st.X)
	}
	if len(hits) != 0 {
		t.Fatal("an arrow click must not reach the tab under it")
	}
	Layout(c, root, 300, 100)
	c.In = widget.Input{X: 2, Y: 10, Click: true, Held: true}
	Update(c, root)
	if st.X != 0 {
		t.Fatalf("the left arrow must scroll back, X = %d", st.X)
	}
}

func TestHScrollRevealsTheFlaggedChildOnceWhenItChanges(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	build := func(flagged int) *El {
		var items []Item
		for i := 0; i < 5; i++ {
			opts := []Item{W(40)}
			if i == flagged {
				opts = append(opts, Reveal())
			}
			items = append(items, box(opts...))
		}
		return HScroll(st, append([]Item{W(100), H(20)}, items...)...)
	}
	_ = hits
	c := testCtx(t)
	root := Col(build(4))
	Layout(c, root, 300, 100)
	if st.X != 136 { // the last box ends at 200; clamped to the maximum
		t.Fatalf("X = %d, want 136", st.X)
	}
	st.X = 0 // the user scrolls away; the same flagged child must not pull it back
	root = Col(build(4))
	Layout(c, root, 300, 100)
	if st.X != 0 {
		t.Fatalf("X = %d, a repeated reveal must not fight the user", st.X)
	}
	root = Col(build(0)) // a different child is flagged now
	st.X = 100
	Layout(c, root, 300, 100)
	if st.X >= 100 || st.X > 0 {
		t.Fatalf("X = %d, want the first box revealed", st.X)
	}
}

func TestHScrollFollowsTheHorizontalWheelOfATouchpad(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(strip(st, 5, &hits))
	c := testCtx(t)
	Layout(c, root, 300, 100)
	c.In = widget.Input{X: 50, Y: 10, WheelX: -1} // two fingers swiping left
	Update(c, root)
	if st.X <= 0 {
		t.Fatalf("a sideways swipe must scroll the strip, X = %d", st.X)
	}
	if c.In.WheelX != 0 {
		t.Fatal("the swipe is consumed")
	}
	x := st.X
	c.In = widget.Input{X: 50, Y: 10, WheelX: 1}
	Update(c, root)
	if st.X >= x {
		t.Fatalf("swiping back must scroll back, X = %d", st.X)
	}
}

func TestHScrollArrowsAreAlwaysShownWhenOverflowingAndAtALimitStillTakeTheClick(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(strip(st, 5, &hits))
	c := testCtx(t)
	Layout(c, root, 300, 100)
	if l, r := hscrollOf(root).arrowsShown(root.kids[0]); !l || !r {
		t.Fatalf("both arrows are drawn while the strip overflows, got %v %v", l, r)
	}
	// At the left end the left arrow is dim, but a click on it must not
	// fall through to the tab beneath.
	c.In = widget.Input{X: 3, Y: 10, Click: true, Held: true}
	Update(c, root)
	if st.X != 0 || len(hits) != 0 {
		t.Fatalf("X = %d hits %v", st.X, hits)
	}
	fit := Col(strip(&ScrollState{}, 2, &hits))
	Layout(c, fit, 300, 100)
	if l, r := hscrollOf(fit).arrowsShown(fit.kids[0]); l || r {
		t.Fatalf("no arrows when everything fits, got %v %v", l, r)
	}
}

func hscrollOf(root *El) *hscrollB { return root.kids[0].b.(*hscrollB) }

func TestHoldingAnArrowKeepsScrolling(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(strip(st, 20, &hits)) // 800px of boxes in 100px
	c := testCtx(t)
	Layout(c, root, 300, 100)
	press := widget.Input{X: 100 - hscrollArrowW + 2, Y: 10, Click: true, Held: true}
	c.In = press
	Update(c, root)
	first := st.X
	if first <= 0 {
		t.Fatal("the press scrolls once")
	}
	held := widget.Input{X: press.X, Y: 10, Held: true}
	for range hscrollHoldDelay - 2 { // not yet repeating
		c.In = held
		Update(c, root)
	}
	if st.X != first {
		t.Fatalf("repeated too early: %d -> %d", first, st.X)
	}
	for range 30 {
		c.In = held
		Update(c, root)
	}
	if st.X <= first {
		t.Fatalf("holding the arrow must keep scrolling: %d -> %d", first, st.X)
	}
	x := st.X
	c.In = widget.Input{X: press.X, Y: 10} // released
	Update(c, root)
	for range 30 {
		c.In = held // pressed elsewhere and dragged onto the arrow: not a hold
		Update(c, root)
	}
	if st.X != x {
		t.Fatalf("a drag onto the arrow must not scroll: %d -> %d", x, st.X)
	}
}

func TestHScrollPansWhenDraggedWithTheLeftButton(t *testing.T) {
	var hits []int
	st := &ScrollState{X: 50}
	root := Col(strip(st, 5, &hits))
	c := testCtx(t)
	Layout(c, root, 300, 100)
	at := func(x int, click bool) {
		c.In = widget.Input{X: x, Y: 10, Click: click, Held: true}
		Update(c, root)
		Layout(c, root, 300, 100)
	}
	at(60, true)
	at(58, false) // a twitch is not a drag
	if st.X != 50 {
		t.Fatalf("X = %d, a 2px twitch must not pan", st.X)
	}
	at(40, false) // 20px left: the content follows the pointer, so X grows
	if st.X != 70 {
		t.Fatalf("X = %d, want 70", st.X)
	}
	at(70, false) // 10px right of the start: X shrinks
	if st.X != 40 {
		t.Fatalf("X = %d, want 40", st.X)
	}
	at(-300, false) // the pointer leaves the strip; the pan keeps following it, clamped
	if st.X != 136 {
		t.Fatalf("X = %d, want the clamp 136", st.X)
	}
	c.In = widget.Input{X: -300, Y: 10}
	Update(c, root)
	if st.X != 136 || st.panning {
		t.Fatalf("release ends the pan: X = %d panning %v", st.X, st.panning)
	}
}

func TestHScrollThatFitsDoesNotPanAndReportsOverflow(t *testing.T) {
	var hits []int
	st := &ScrollState{}
	root := Col(strip(st, 2, &hits))
	c := testCtx(t)
	Layout(c, root, 300, 100)
	c.In = widget.Input{X: 50, Y: 10, Click: true, Held: true}
	Update(c, root)
	c.In = widget.Input{X: 10, Y: 10, Held: true}
	Update(c, root)
	if st.X != 0 || st.Overflowing {
		t.Fatalf("X = %d overflowing %v", st.X, st.Overflowing)
	}
	st2 := &ScrollState{}
	root2 := Col(strip(st2, 5, &hits))
	Layout(c, root2, 300, 100)
	if !st2.Overflowing {
		t.Fatal("an overflowing strip says so, so the app can tell panning from reordering")
	}
}
