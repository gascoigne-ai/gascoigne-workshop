package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Layout is rects in, rects out: it places the children for the state it
// finds and never changes it. Update settles the state afterwards.
func TestLayoutLeavesTheScrollStateAloneAndUpdateClampsIt(t *testing.T) {
	var hits []int
	st := &ScrollState{Y: 500} // far past the end of 200px of rows in 100px
	root := Col(scroller(st, 100, rows(10, &hits)...))
	c := testCtx(t)
	Layout(c, root, 200, 100)
	if st.Y != 500 {
		t.Fatalf("Layout wrote Y = %d", st.Y)
	}
	c.In = core.Input{}
	Update(c, root)
	if st.Y != 100 {
		t.Fatalf("Update left Y = %d, want the clamp 100", st.Y)
	}
}

func TestLayoutLeavesTheHScrollStateAloneAndUpdateSettlesIt(t *testing.T) {
	var hits []int
	st := &ScrollState{X: 999}
	root := Col(strip(st, 5, &hits)) // 200px of boxes in 100px
	c := testCtx(t)
	Layout(c, root, 300, 100)
	if st.X != 999 || st.Overflowing {
		t.Fatalf("Layout wrote X = %d, Overflowing = %v", st.X, st.Overflowing)
	}
	c.In = core.Input{}
	Update(c, root)
	if st.X != 136 || !st.Overflowing {
		t.Fatalf("Update left X = %d, Overflowing = %v", st.X, st.Overflowing)
	}
}

func TestUpdateBringsTheKeptChildIntoView(t *testing.T) {
	st := &ScrollState{}
	var items []Item
	for i := range 5 {
		if i == 4 {
			items = append(items, box(W(40), KeepInView()))
			continue
		}
		items = append(items, box(W(40)))
	}
	root := Col(HScroll(st, append([]Item{W(100), H(20)}, items...)...))
	c := testCtx(t)
	Layout(c, root, 300, 100)
	if st.X != 0 {
		t.Fatalf("Layout moved X to %d", st.X)
	}
	c.In = core.Input{}
	Update(c, root)
	if st.X == 0 {
		t.Fatal("Update must scroll the kept child into view")
	}
}
