package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestRowFixedGrowGapPad(t *testing.T) {
	a, b, c3 := box(W(50)), box(Grow(1)), box(W(30))
	root := Row(Pad(10), Gap(5), a, b, c3)
	Layout(testCtx(t), root, 300, 100)

	want := []core.Rect{
		{X: 10, Y: 10, W: 50, H: 80},
		{X: 65, Y: 10, W: 300 - 20 - 50 - 30 - 10, H: 80},
		{X: 300 - 10 - 30, Y: 10, W: 30, H: 80},
	}
	for i, k := range []*El{a, b, c3} {
		if k.rect != want[i] {
			t.Errorf("child %d rect = %+v, want %+v", i, k.rect, want[i])
		}
	}
}

func TestGrowSharesFreeSpaceByWeightWithoutLosingPixels(t *testing.T) {
	a, b := box(Grow(1)), box(Grow(2))
	root := Row(a, b)
	Layout(testCtx(t), root, 100, 10)
	if a.rect.W+b.rect.W != 100 {
		t.Fatalf("widths %d + %d must fill 100", a.rect.W, b.rect.W)
	}
	if a.rect.W != 33 || b.rect.X != 33 {
		t.Fatalf("a = %+v b = %+v, want a.W 33 and b starting at 33", a.rect, b.rect)
	}
}

func TestColFixedHeightAndFixedWidthChild(t *testing.T) {
	top, rest, narrow := box(H(40)), box(Grow(1)), box(W(20), H(10))
	root := Col(top, rest, narrow)
	Layout(testCtx(t), root, 200, 100)

	if top.rect != (core.Rect{X: 0, Y: 0, W: 200, H: 40}) {
		t.Errorf("top = %+v", top.rect)
	}
	if rest.rect != (core.Rect{X: 0, Y: 40, W: 200, H: 50}) {
		t.Errorf("rest = %+v", rest.rect)
	}
	if narrow.rect != (core.Rect{X: 0, Y: 90, W: 20, H: 10}) {
		t.Errorf("narrow = %+v", narrow.rect)
	}
}

func TestOptionalChildIsDroppedOnlyWhenTooNarrow(t *testing.T) {
	for _, tt := range []struct {
		w          int
		wantHidden bool
	}{{200, false}, {100, true}} {
		extra := box(W(80), Optional())
		root := Row(box(W(60)), extra)
		Layout(testCtx(t), root, tt.w, 10)
		if extra.hidden != tt.wantHidden {
			t.Errorf("width %d: hidden = %v, want %v", tt.w, extra.hidden, tt.wantHidden)
		}
	}
}

// A row that overflows takes the overflow from its Shrink children, first
// to last, each down to its MinW; a row that fits is laid out as without
// Shrink.
func TestShrinkChildrenGiveUpOnlyTheOverflowFirstToLast(t *testing.T) {
	cases := []struct {
		name            string
		width           int
		firstW, secondW int
		lastX           int
	}{
		{"fits", 300, 100, 80, 200},
		{"first gives the overflow", 200, 60, 80, 160},
		{"first stops at its min, second gives the rest", 140, 30, 50, 100},
		{"both at their min", 50, 30, 0, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := Row(Shrink(), MinW(30), box(W(100)))
			second := Row(Shrink(), box(W(80)))
			fixed := box(W(20))
			last := box(W(40))
			root := Row(first, second, fixed, last)
			Layout(testCtx(t), root, tc.width, 10)
			if first.rect.W != tc.firstW || second.rect.W != tc.secondW {
				t.Errorf("shrinking children %d and %d wide, want %d and %d", first.rect.W, second.rect.W, tc.firstW, tc.secondW)
			}
			if fixed.rect.W != 20 {
				t.Errorf("a child without Shrink is %d wide, want 20", fixed.rect.W)
			}
			if last.rect.X != tc.lastX {
				t.Errorf("last child at x=%d, want %d", last.rect.X, tc.lastX)
			}
		})
	}
}

func TestStackFloatPinsTopRight(t *testing.T) {
	under, menu := box(), box(W(60), H(40), FloatTR(30, 10))
	root := Stack(under, menu)
	Layout(testCtx(t), root, 400, 300)

	if under.rect != (core.Rect{W: 400, H: 300}) {
		t.Errorf("under = %+v", under.rect)
	}
	if menu.rect != (core.Rect{X: 330, Y: 30, W: 60, H: 40}) {
		t.Errorf("menu = %+v", menu.rect)
	}
}

func TestIfForAndFragmentsFlatten(t *testing.T) {
	items := []string{"a", "b", "c"}
	root := Row(
		If(false, box()),
		If(true, box()),
		For(items, func(int, string) Node { return box() }),
		nil,
		Frag(box(), box()),
	)
	if got := len(root.kids); got != 1+3+2 {
		t.Fatalf("children = %d, want 6", got)
	}
}

func TestClickReachesOnlyTheElementUnderThePointer(t *testing.T) {
	var hits []string
	left := box(W(50), OnClick(func() { hits = append(hits, "left") }))
	right := box(W(50), OnClick(func() { hits = append(hits, "right") }))
	root := Row(left, right)
	c := testCtx(t)
	Layout(c, root, 100, 20)

	click(c, 10, 5)
	Update(c, root)
	click(c, 70, 5)
	Update(c, root)
	click(c, 500, 5)
	Update(c, root)

	if len(hits) != 2 || hits[0] != "left" || hits[1] != "right" {
		t.Fatalf("hits = %v, want [left right]", hits)
	}
}

func TestDoubleClickRunsOnTheSecondClickOverTheElement(t *testing.T) {
	var clicks, doubles int
	root := Row(box(W(50), OnClick(func() { clicks++ }), OnDoubleClick(func() { doubles++ })), box(W(50)))
	c := testCtx(t)
	Layout(c, root, 100, 20)
	step := func(in core.Input) { c.In = in; Update(c, root) }

	step(core.Input{X: 10, Y: 5, Click: true, Clicks: 1})
	if clicks != 1 || doubles != 0 {
		t.Fatalf("a single click: clicks %d doubles %d", clicks, doubles)
	}
	step(core.Input{X: 10, Y: 5, Click: true, Clicks: 2})
	if clicks != 2 || doubles != 1 {
		t.Fatalf("the second click is a click and a double click: clicks %d doubles %d", clicks, doubles)
	}
	step(core.Input{X: 70, Y: 5, Click: true, Clicks: 2})
	if doubles != 1 {
		t.Fatal("a double click beside the element ran it")
	}
}

func TestModalSwallowsClicksAndDismissesOnOutsideClick(t *testing.T) {
	var under, inside, dismissed int
	menu := box(W(40), H(40), FloatTR(0, 0), OnClick(func() { inside++ }), Modal(), OnDismiss(func() { dismissed++ }))
	root := Stack(box(OnClick(func() { under++ })), menu)
	c := testCtx(t)
	Layout(c, root, 200, 100)

	click(c, 190, 10) // inside the menu (top right)
	Update(c, root)
	click(c, 10, 90) // outside it
	Update(c, root)

	if inside != 1 || dismissed != 1 || under != 0 {
		t.Fatalf("inside %d dismissed %d under %d; want 1, 1, 0", inside, dismissed, under)
	}
}

type fakeClipboard struct {
	text    string
	readErr error
}

func (f *fakeClipboard) Read() (string, error)     { return f.text, f.readErr }
func (f *fakeClipboard) Write(string string) error { f.text = string; return nil }

func TestSpacerAndRulesTakeTheirSizes(t *testing.T) {
	h, v, gap := HRule(), VRule(), Spacer()
	inset := VRuleInset(8)
	root := Col(h, Row(H(40), v, gap, inset))
	Layout(testCtx(t), root, 200, 100)

	if h.rect.H != 1 || h.rect.W != 200 {
		t.Errorf("HRule = %+v", h.rect)
	}
	if v.rect.W != 1 || v.rect.H != 40 {
		t.Errorf("VRule = %+v", v.rect)
	}
	if inset.rect.W != 1 || inset.kids[0].rect.H != 40-16 {
		t.Errorf("VRuleInset line = %+v inside %+v", inset.kids[0].rect, inset.rect)
	}
	if gap.rect.W != 200-1-1 {
		t.Errorf("Spacer must take the free width, got %d", gap.rect.W)
	}
}

func TestLinesMakesOneRowPerLine(t *testing.T) {
	lines := Lines("a\nb\nc", core.Muted)
	Layout(testCtx(t), Col(lines), 100, 200)
	if len(lines.kids) != 3 || lines.rect.H != 3*22 {
		t.Fatalf("rows %d height %d, want 3 rows of 22px", len(lines.kids), lines.rect.H)
	}
	for _, k := range lines.kids {
		if k.fg != core.Muted {
			t.Fatalf("line color = %q", k.fg)
		}
	}
}

func TestSmallLinesUseTheSmallFontAtNoteHeight(t *testing.T) {
	lines := Lines("a\nb", core.Muted, Small(), Gap(2))
	Layout(testCtx(t), Col(lines), 100, 200)
	if len(lines.kids) != 2 || lines.rect.H != 2*core.NoteH+2 {
		t.Fatalf("rows %d height %d, want 2 rows of %dpx and a 2px gap", len(lines.kids), lines.rect.H, core.NoteH)
	}
	for _, k := range lines.kids {
		if !k.small || k.fg != core.Muted {
			t.Fatalf("line small=%v color=%q, want a small muted text", k.small, k.fg)
		}
	}
}

func TestFloatAtKeepsTheElementInsideTheStack(t *testing.T) {
	tests := []struct {
		name  string
		x, y  int
		wantX int
		wantY int
	}{
		{"fits where asked", 100, 50, 100, 50},
		{"sticks out on the right: moved left", 380, 50, 400 - 60, 50},
		{"sticks out at the bottom: opens above", 100, 90, 100, 90 - 40},
		{"both", 390, 95, 400 - 60, 95 - 40},
		{"never above the top", 100, 20, 100, 20},
	}
	for _, tt := range tests {
		menu := box(W(60), H(40), FloatAt(tt.x, tt.y))
		root := Stack(box(), menu)
		Layout(testCtx(t), root, 400, 100)
		if menu.rect.X != tt.wantX || menu.rect.Y != tt.wantY {
			t.Errorf("%s: at %d,%d want %d,%d", tt.name, menu.rect.X, menu.rect.Y, tt.wantX, tt.wantY)
		}
	}
}

func TestFloatCenterCentersTheElement(t *testing.T) {
	dialog := box(W(100), H(40), FloatCenter())
	root := Stack(box(), dialog)
	Layout(testCtx(t), root, 400, 200)
	if dialog.rect != (core.Rect{X: 150, Y: 80, W: 100, H: 40}) {
		t.Fatalf("dialog = %+v", dialog.rect)
	}
}

func TestMovedShiftsACenteredElementAndKeepsItInsideTheStack(t *testing.T) {
	tests := []struct {
		name string
		move MoveState
		want core.Rect
	}{
		{"zero is the center", MoveState{}, core.Rect{X: 150, Y: 80, W: 100, H: 40}},
		{"offset from the center", MoveState{X: -50, Y: 30}, core.Rect{X: 100, Y: 110, W: 100, H: 40}},
		{"stops at the left and top edge", MoveState{X: -999, Y: -999}, core.Rect{X: 0, Y: 0, W: 100, H: 40}},
		{"stops at the right and bottom edge", MoveState{X: 999, Y: 999}, core.Rect{X: 300, Y: 160, W: 100, H: 40}},
	}
	for _, tt := range tests {
		dialog := box(W(100), H(40), FloatCenter(), Moved(&tt.move))
		Layout(testCtx(t), Stack(box(), dialog), 400, 200)
		if dialog.rect != tt.want {
			t.Errorf("%s: dialog = %+v, want %+v", tt.name, dialog.rect, tt.want)
		}
	}
}

// dragRig is a 100x40 centered dialog in a 400x200 stack whose top 10 rows
// are a drag handle with a 20 px button at its left end.
func dragRig(t *testing.T, moveState *MoveState, dismissed *int) (*El, *core.Ctx) {
	t.Helper()
	handle := Row(H(20), DragHandle(), box(W(20), OnClick(func() {})))
	dialog := Col(W(100), H(40), FloatCenter(), Moved(moveState), Modal(), OnDismiss(func() { *dismissed++ }), handle, box(Grow(1)))
	root := Stack(box(), dialog)
	c := testCtx(t)
	Layout(c, root, 400, 200)
	return root, c
}

func TestDraggingAHandleMovesTheDialog(t *testing.T) {
	m, dismissed := &MoveState{}, 0
	root, c := dragRig(t, m, &dismissed)
	step := func(in core.Input) { c.In = in; Update(c, root) }
	step(core.Input{X: 200, Y: 90, Click: true, Held: true}) // on the handle, right of the button
	step(core.Input{X: 230, Y: 100, Held: true})
	if m.X != 30 || m.Y != 10 {
		t.Fatalf("offset %d,%d, want 30,10", m.X, m.Y)
	}
	if c.Cursor != core.CursorMove {
		t.Fatalf("cursor %v while dragging", c.Cursor)
	}
	step(core.Input{X: 300, Y: 100}) // released: the dialog stays
	step(core.Input{X: 400, Y: 195})
	if m.X != 30 || m.Y != 10 || dismissed != 0 {
		t.Fatalf("after the release offset %d,%d, dismissed %d", m.X, m.Y, dismissed)
	}
}

func TestTheGrabCursorShowsOnlyWhileTheHandleIsPressed(t *testing.T) {
	m, dismissed := &MoveState{}, 0
	root, c := dragRig(t, m, &dismissed)
	step := func(in core.Input) core.Cursor {
		c.Cursor = core.CursorDefault
		c.In = in
		Update(c, root)
		return c.Cursor
	}
	if got := step(core.Input{X: 200, Y: 90}); got != core.CursorDefault {
		t.Errorf("hovering the handle: cursor %v, want the default", got)
	}
	if got := step(core.Input{X: 200, Y: 90, Click: true, Held: true}); got != core.CursorMove {
		t.Errorf("pressing the handle: cursor %v, want the grab cursor", got)
	}
	if got := step(core.Input{X: 210, Y: 95, Held: true}); got != core.CursorMove {
		t.Errorf("dragging: cursor %v, want the grab cursor", got)
	}
	if got := step(core.Input{X: 210, Y: 95}); got != core.CursorDefault {
		t.Errorf("after the release: cursor %v, want the default", got)
	}
	if got := step(core.Input{X: 155, Y: 90, Click: true, Held: true}); got != core.CursorDefault {
		t.Errorf("pressing a control in the handle: cursor %v, want the default", got)
	}
}

func TestPressingAControlInsideTheHandleDoesNotDrag(t *testing.T) {
	m, dismissed := &MoveState{}, 0
	root, c := dragRig(t, m, &dismissed)
	c.In = core.Input{X: 155, Y: 90, Click: true, Held: true} // on the button
	Update(c, root)
	c.In = core.Input{X: 185, Y: 100, Held: true}
	Update(c, root)
	if m.X != 0 || m.Y != 0 {
		t.Fatalf("dragged by the button to %d,%d", m.X, m.Y)
	}
}

func TestDoubleClickingAHandleCentersTheDialog(t *testing.T) {
	m, dismissed := &MoveState{X: 40, Y: -20}, 0
	root, c := dragRig(t, m, &dismissed)
	c.In = core.Input{X: 250, Y: 70, Click: true, Held: true, Clicks: 2} // the dialog is at 190,60
	Update(c, root)
	if m.X != 0 || m.Y != 0 {
		t.Fatalf("offset %d,%d, want the center", m.X, m.Y)
	}
}

func TestADragPastTheEdgeDoesNotPileUp(t *testing.T) {
	m, dismissed := &MoveState{}, 0
	root, c := dragRig(t, m, &dismissed)
	c.In = core.Input{X: 200, Y: 90, Click: true, Held: true}
	Update(c, root)
	c.In = core.Input{X: 2000, Y: 90, Held: true}
	Update(c, root)
	Layout(c, root, 400, 200) // the layout keeps the dialog inside and writes the offset back
	if m.X != 150 {
		t.Fatalf("X = %d, want 150 (the right edge)", m.X)
	}
}

func TestOnRightClickGetsThePointer(t *testing.T) {
	var gx, gy, n int
	root := Row(box(W(100), OnRightClick(func(posX, posY int) { gx, gy, n = posX, posY, n+1 })))
	c := testCtx(t)
	Layout(c, root, 200, 50)
	c.In = core.Input{X: 30, Y: 20, RightClick: true}
	Update(c, root)
	c.In = core.Input{X: 150, Y: 20, RightClick: true} // outside the element
	Update(c, root)
	c.In = core.Input{X: 30, Y: 20, Click: true} // a left click is not a right click
	Update(c, root)
	if n != 1 || gx != 30 || gy != 20 {
		t.Fatalf("calls %d at %d,%d", n, gx, gy)
	}
}

func TestModalAlsoSwallowsAndDismissesOnRightClickOutside(t *testing.T) {
	var under, dismissed int
	menu := box(W(40), H(40), FloatAt(0, 0), Modal(), OnDismiss(func() { dismissed++ }))
	root := Stack(box(OnRightClick(func(int, int) { under++ })), menu)
	c := testCtx(t)
	Layout(c, root, 200, 100)
	c.In = core.Input{X: 150, Y: 80, RightClick: true}
	Update(c, root)
	if dismissed != 1 || under != 0 {
		t.Fatalf("dismissed %d, reached the element below %d", dismissed, under)
	}
}

func TestRightClickIsHandledByTheInnermostElementOnly(t *testing.T) {
	var inner, outer int
	root := Row(Row(W(100), OnRightClick(func(int, int) { outer++ }),
		Row(W(50), OnRightClick(func(int, int) { inner++ }))))
	c := testCtx(t)
	Layout(c, root, 200, 50)
	c.In = core.Input{X: 20, Y: 10, RightClick: true} // over the inner element
	Update(c, root)
	if inner != 1 || outer != 0 {
		t.Fatalf("inner %d outer %d: one right click opens one menu", inner, outer)
	}
	c.In = core.Input{X: 80, Y: 10, RightClick: true} // over the outer only
	Update(c, root)
	if inner != 1 || outer != 1 {
		t.Fatalf("inner %d outer %d", inner, outer)
	}
}

func TestBorderLeftOptionRecordsTheTokenAndInset(t *testing.T) {
	e := box(BorderL(core.BorderSoft), BorderInset(6))
	if e.borderL != core.BorderSoft || e.borderInset != 6 {
		t.Fatalf("%q %d", e.borderL, e.borderInset)
	}
}

// Every side of an element can be set the same way: border, padding and margin.
func TestSideOptionsAreConsistent(t *testing.T) {
	e := box(BorderL("a"), BorderR("b"), BorderT("c"), BorderB("d"))
	if e.borderL != "a" || e.borderR != "b" || e.borderT != "c" || e.borderB != "d" {
		t.Fatalf("border sides %q %q %q %q", e.borderL, e.borderR, e.borderT, e.borderB)
	}
	x, y := box(BorderX("x"), BorderY("y")), box(PadX(1), PadY(2), MarginX(3), MarginY(4))
	if x.borderL != "x" || x.borderR != "x" || x.borderT != "y" || x.borderB != "y" {
		t.Fatal("BorderX and BorderY")
	}
	if y.padL != 1 || y.padR != 1 || y.padT != 2 || y.padB != 2 || y.mL != 3 || y.mR != 3 || y.mT != 4 || y.mB != 4 {
		t.Fatal("axes")
	}
	m := box(MarginL(1), MarginT(2), MarginR(3), MarginB(4), PadL(5), PadT(6), PadR(7), PadB(8))
	if m.mL != 1 || m.mT != 2 || m.mR != 3 || m.mB != 4 || m.padL != 5 || m.padT != 6 || m.padR != 7 || m.padB != 8 {
		t.Fatal("single sides")
	}
}

// Event rule: a left click reaches the onClick of every element it is over
// and of every ancestor, innermost first; a right click only the innermost
// handler (TestRightClickIsHandledByTheInnermostElementOnly).
func TestLeftClickReachesEveryAncestorInnermostFirst(t *testing.T) {
	var order []string
	root := Row(W(100), OnClick(func() { order = append(order, "outer") }),
		Row(W(100), OnClick(func() { order = append(order, "inner") })))
	c := testCtx(t)
	Layout(c, root, 200, 50)
	c.In = core.Input{X: 30, Y: 20, Click: true}
	Update(c, root)
	if len(order) != 2 || order[0] != "inner" || order[1] != "outer" {
		t.Fatalf("order %v", order)
	}
}

func TestOnClickAtGetsThePointerOfTheClick(t *testing.T) {
	var gx, gy, n int
	root := Row(box(W(100), OnClickAt(func(posX, posY int) { gx, gy, n = posX, posY, n+1 })))
	c := testCtx(t)
	Layout(c, root, 200, 50)
	c.In = core.Input{X: 150, Y: 20, Click: true} // outside the element
	Update(c, root)
	c.In = core.Input{X: 30, Y: 20, Click: true}
	Update(c, root)
	if n != 1 || gx != 30 || gy != 20 {
		t.Fatalf("calls %d at %d,%d", n, gx, gy)
	}
}
