package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func splitRoot(t *testing.T, st *SplitState, w, h int, items ...Item) (*El, *El, *El, *core.Ctx) {
	t.Helper()
	first, second := Row(), Row()
	root := Split(st, 200, append(items, first, second)...)
	c := testCtx(t)
	Layout(c, root, w, h)
	return root, first, second, c
}

func TestSplitPlacesTwoChildrenAroundAOnePixelDivider(t *testing.T) {
	_, first, second, _ := splitRoot(t, &SplitState{}, 600, 100)
	if first.Rect() != (core.Rect{W: 200, H: 100}) {
		t.Errorf("first = %+v", first.Rect())
	}
	if second.Rect() != (core.Rect{X: 201, W: 399, H: 100}) {
		t.Errorf("second = %+v (the divider takes one pixel)", second.Rect())
	}
}

func TestSplitVerticalStacksTheChildren(t *testing.T) {
	_, first, second, _ := splitRoot(t, &SplitState{}, 100, 600, Vertical())
	if first.Rect() != (core.Rect{W: 100, H: 200}) || second.Rect() != (core.Rect{Y: 201, W: 100, H: 399}) {
		t.Fatalf("first %+v second %+v", first.Rect(), second.Rect())
	}
}

func TestSplitRatioIsAFractionOfTheContainer(t *testing.T) {
	st := &SplitState{}
	first, second := Row(), Row()
	root := Split(st, 0.25, AsRatio(), first, second)
	c := testCtx(t)
	Layout(c, root, 801, 50)
	if first.Rect().W != 200 { // 25% of the 800px left after the divider
		t.Fatalf("first = %+v", first.Rect())
	}
	Layout(c, root, 401, 50)
	if first.Rect().W != 100 {
		t.Fatalf("a smaller window keeps the proportion, first = %+v", first.Rect())
	}
	_ = second
}

func TestSplitPixelSizeStaysWhenTheWindowChanges(t *testing.T) {
	st := &SplitState{}
	root, first, _, c := splitRoot(t, st, 600, 50)
	Layout(c, root, 900, 50)
	if first.Rect().W != 200 {
		t.Fatalf("first = %+v", first.Rect())
	}
}

func TestSplitMinimumsClampBothSides(t *testing.T) {
	tests := []struct {
		name  string
		size  float64
		width int
		want  int
	}{
		{"below the minimum", 10, 600, 120},
		{"above what leaves the second its minimum", 590, 600, 600 - 1 - 150},
		{"inside the limits", 300, 600, 300},
		{"a window too small for both minimums favors the first", 200, 200, 120},
	}
	for _, tt := range tests {
		st := &SplitState{Size: tt.size}
		first, second := Row(), Row()
		root := Split(st, 200, MinFirst(120), MinSecond(150), first, second)
		Layout(testCtx(t), root, tt.width, 50)
		if first.Rect().W != tt.want {
			t.Errorf("%s: first.W = %d, want %d", tt.name, first.Rect().W, tt.want)
		}
	}
}

func TestDraggingTheDividerResizesTheFirstChild(t *testing.T) {
	st := &SplitState{}
	root, first, _, c := splitRoot(t, st, 600, 100, MinFirst(50))

	c.In = core.Input{X: 200, Y: 40, Click: true, Held: true}
	Update(c, root)
	c.In = core.Input{X: 320, Y: 40, Held: true}
	Update(c, root)
	Layout(c, root, 600, 100)
	if first.Rect().W != 320 {
		t.Fatalf("after dragging to x=320 the first child is %d wide", first.Rect().W)
	}
	c.In = core.Input{X: 320, Y: 40} // release
	Update(c, root)
	c.In = core.Input{X: 500, Y: 40, Held: true}
	Update(c, root)
	Layout(c, root, 600, 100)
	if first.Rect().W != 320 {
		t.Fatalf("after release the divider must not follow the pointer, first = %d", first.Rect().W)
	}
}

func TestDragKeepsTheGrabOffset(t *testing.T) {
	st := &SplitState{}
	root, first, _, c := splitRoot(t, st, 600, 100)
	c.In = core.Input{X: 202, Y: 40, Click: true, Held: true} // two pixels right of the divider
	Update(c, root)
	c.In = core.Input{X: 302, Y: 40, Held: true}
	Update(c, root)
	Layout(c, root, 600, 100)
	if first.Rect().W != 300 {
		t.Fatalf("the divider moves by the pointer's travel, not jumping to it: %d", first.Rect().W)
	}
}

func TestDoubleClickOnTheDividerResetsTheSize(t *testing.T) {
	st := &SplitState{Size: 340}
	root, first, _, c := splitRoot(t, st, 600, 100)
	c.In = core.Input{X: 340, Y: 40, Click: true, Held: true, Clicks: 2}
	Update(c, root)
	Layout(c, root, 600, 100)
	if first.Rect().W != 200 {
		t.Fatalf("first = %d, want the default 200", first.Rect().W)
	}
}

func TestClicksNextToTheDividerStillReachTheChildren(t *testing.T) {
	st := &SplitState{}
	var hits []string
	left := Row(OnClick(func() { hits = append(hits, "left") }))
	right := Row(OnClick(func() { hits = append(hits, "right") }))
	root := Split(st, 200, left, right)
	c := testCtx(t)
	Layout(c, root, 600, 100)
	click(c, 100, 50)
	Update(c, root)
	click(c, 400, 50)
	Update(c, root)
	if len(hits) != 2 || st.drag.on {
		t.Fatalf("hits %v dragging %v", hits, st.drag.on)
	}
}

func TestDividerAsksForAResizeCursor(t *testing.T) {
	st := &SplitState{}
	root, _, _, c := splitRoot(t, st, 600, 100)
	for _, tt := range []struct {
		name string
		in   core.Input
		want core.Cursor
	}{
		{"over the divider", core.Input{X: 200, Y: 40}, core.CursorResizeH},
		{"a little off it", core.Input{X: 202, Y: 40}, core.CursorResizeH},
		{"elsewhere", core.Input{X: 50, Y: 40}, core.CursorDefault},
	} {
		c.Cursor = core.CursorDefault
		c.In = tt.in
		Update(c, root)
		if c.Cursor != tt.want {
			t.Errorf("%s: cursor %v, want %v", tt.name, c.Cursor, tt.want)
		}
	}
	vroot, _, _, vc := splitRoot(t, &SplitState{}, 100, 600, Vertical())
	vc.In = core.Input{X: 40, Y: 200}
	Update(vc, vroot)
	if vc.Cursor != core.CursorResizeV {
		t.Errorf("a vertical split asks for the up/down cursor, got %v", vc.Cursor)
	}
}

func TestDraggingKeepsTheResizeCursorEvenOffTheDivider(t *testing.T) {
	st := &SplitState{}
	root, _, _, c := splitRoot(t, st, 600, 100)
	c.In = core.Input{X: 200, Y: 40, Click: true, Held: true}
	Update(c, root)
	c.Cursor = core.CursorDefault
	c.In = core.Input{X: 450, Y: 40, Held: true}
	Update(c, root)
	if c.Cursor != core.CursorResizeH {
		t.Fatalf("cursor %v while dragging", c.Cursor)
	}
}
