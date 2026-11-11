package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// resizeRig is a 300x200 resizable element at the origin of a context.
func resizeRig(t *testing.T, st *ResizeState) (*El, *core.Ctx) {
	t.Helper()
	root := Resizable(st, 120, 80, 500, 400, Row())
	c := testCtx(t)
	Layout(c, root, 300, 200)
	return root, c
}

func resizeStep(c *core.Ctx, root *El, in core.Input) {
	c.In = in
	Update(c, root)
}

func TestDraggingAnEdgeResizesTheElementAroundItsCenter(t *testing.T) {
	tests := []struct {
		name           string
		pressX, pressY int
		dragX, dragY   int
		wantW, wantH   int
		wantCursor     core.Cursor
	}{
		{"right edge outward", 298, 100, 318, 100, 340, 200, core.CursorResizeH},
		{"right edge inward", 298, 100, 278, 100, 260, 200, core.CursorResizeH},
		{"left edge outward", 1, 100, -19, 100, 340, 200, core.CursorResizeH},
		{"bottom edge outward", 150, 198, 150, 213, 300, 230, core.CursorResizeV},
		{"top edge outward", 150, 1, 150, -9, 300, 220, core.CursorResizeV},
		{"bottom right corner", 298, 198, 308, 208, 320, 220, core.CursorResizeNWSE},
		{"top left corner", 1, 1, -9, -9, 320, 220, core.CursorResizeNWSE},
		{"top right corner", 298, 1, 308, -9, 320, 220, core.CursorResizeNESW},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := &ResizeState{}
			root, c := resizeRig(t, st)
			resizeStep(c, root, core.Input{X: test.pressX, Y: test.pressY, Click: true, Held: true})
			resizeStep(c, root, core.Input{X: test.dragX, Y: test.dragY, Held: true})
			if st.W != test.wantW || st.H != test.wantH {
				t.Fatalf("size %dx%d, want %dx%d", st.W, st.H, test.wantW, test.wantH)
			}
			if c.Cursor != test.wantCursor {
				t.Fatalf("cursor %v, want %v", c.Cursor, test.wantCursor)
			}
		})
	}
}

func TestTheSizeStaysWithinTheMinimumAndMaximum(t *testing.T) {
	st := &ResizeState{}
	root, c := resizeRig(t, st)
	resizeStep(c, root, core.Input{X: 298, Y: 100, Click: true, Held: true})
	resizeStep(c, root, core.Input{X: 2000, Y: 100, Held: true})
	if st.W != 500 {
		t.Fatalf("W = %d, want the maximum 500", st.W)
	}
	resizeStep(c, root, core.Input{X: -2000, Y: 100, Held: true})
	if st.W != 120 {
		t.Fatalf("W = %d, want the minimum 120", st.W)
	}
}

func TestPressingInTheMiddleOrReleasingDoesNotResize(t *testing.T) {
	st := &ResizeState{}
	root, c := resizeRig(t, st)
	resizeStep(c, root, core.Input{X: 150, Y: 100, Click: true, Held: true})
	resizeStep(c, root, core.Input{X: 200, Y: 100, Held: true})
	if st.W != 0 || st.H != 0 {
		t.Fatalf("a press in the middle resized to %dx%d", st.W, st.H)
	}
	resizeStep(c, root, core.Input{X: 298, Y: 100, Click: true, Held: true})
	resizeStep(c, root, core.Input{X: 308, Y: 100}) // released
	resizeStep(c, root, core.Input{X: 400, Y: 100})
	if st.W != 0 && st.W != 300 { // a press fixes the size it has; a release moves nothing
		t.Fatalf("after the release W = %d", st.W)
	}
}

func TestDoubleClickingAnEdgeGoesBackToTheDefaultSize(t *testing.T) {
	st := &ResizeState{W: 450, H: 380}
	root, c := resizeRig(t, st)
	resizeStep(c, root, core.Input{X: 298, Y: 100, Click: true, Held: true, Clicks: 2})
	if st.W != 0 || st.H != 0 {
		t.Fatalf("size %dx%d, want 0x0 meaning the default", st.W, st.H)
	}
}

func TestTheCursorChangesOverAnEdgeEvenWithoutPressing(t *testing.T) {
	st := &ResizeState{}
	root, c := resizeRig(t, st)
	resizeStep(c, root, core.Input{X: 298, Y: 100})
	if c.Cursor != core.CursorResizeH {
		t.Fatalf("cursor %v over the right edge", c.Cursor)
	}
	c.Cursor = core.CursorDefault
	resizeStep(c, root, core.Input{X: 150, Y: 100})
	if c.Cursor != core.CursorDefault {
		t.Fatalf("cursor %v in the middle", c.Cursor)
	}
}
