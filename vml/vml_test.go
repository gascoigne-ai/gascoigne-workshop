package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

func testCtx(t *testing.T) *widget.Ctx {
	t.Helper()
	c, err := widget.NewCtx(widget.Dark())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func box(items ...Item) *El { return newEl(plain{}, items) }

func TestRowFixedGrowGapPad(t *testing.T) {
	a, b, c3 := box(W(50)), box(Grow(1)), box(W(30))
	root := Row(Pad(10), Gap(5), a, b, c3)
	Layout(testCtx(t), root, 300, 100)

	want := []widget.Rect{
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

	if top.rect != (widget.Rect{X: 0, Y: 0, W: 200, H: 40}) {
		t.Errorf("top = %+v", top.rect)
	}
	if rest.rect != (widget.Rect{X: 0, Y: 40, W: 200, H: 50}) {
		t.Errorf("rest = %+v", rest.rect)
	}
	if narrow.rect != (widget.Rect{X: 0, Y: 90, W: 20, H: 10}) {
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

func TestStackFloatPinsTopRight(t *testing.T) {
	under, menu := box(), box(W(60), H(40), Float(30, 10))
	root := Stack(under, menu)
	Layout(testCtx(t), root, 400, 300)

	if under.rect != (widget.Rect{W: 400, H: 300}) {
		t.Errorf("under = %+v", under.rect)
	}
	if menu.rect != (widget.Rect{X: 330, Y: 30, W: 60, H: 40}) {
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
	if got := Either(false, box(W(1)), box(W(2))).w; got != 2 {
		t.Fatalf("Either picked the wrong branch")
	}
}

func click(c *widget.Ctx, x, y int) { c.In = widget.Input{X: x, Y: y, Click: true} }

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

func TestModalSwallowsClicksAndDismissesOnOutsideClick(t *testing.T) {
	var under, inside, dismissed int
	menu := box(W(40), H(40), Float(0, 0), OnClick(func() { inside++ }), Modal(func() { dismissed++ }))
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

func TestTableHeightAndEditHandlers(t *testing.T) {
	var toggled, deleted []int
	spec := TableSpec{
		Columns: []string{"Name", "Value"}, Weights: []float64{1, 1},
		Rows:        []TableRow{{Cells: []string{"q", "x"}, Checked: true}, {Cells: []string{"o", "y"}}},
		Placeholder: []string{"Name", "Value"},
		Editable:    true,
		OnToggle:    func(i int) { toggled = append(toggled, i) },
		OnDelete:    func(i int) { deleted = append(deleted, i) },
	}
	tbl := Table(spec)
	root := Col(tbl)
	c := testCtx(t)
	Layout(c, root, 400, 400)

	// header + 2 rows + placeholder
	if got, want := tbl.rect.H, 4*widget.RowH; got != want {
		t.Fatalf("table height = %d, want %d", got, want)
	}
	rowY := func(i int) int { return (i+1)*widget.RowH + widget.RowH/2 }
	click(c, 10, rowY(1))
	Update(c, root)
	click(c, 395, rowY(0))
	Update(c, root)
	click(c, 10, rowY(2)) // placeholder row has no actions
	Update(c, root)

	if len(toggled) != 1 || toggled[0] != 1 || len(deleted) != 1 || deleted[0] != 0 {
		t.Fatalf("toggled %v deleted %v; want [1] and [0]", toggled, deleted)
	}
}

func TestTabsSelect(t *testing.T) {
	sel := -1
	tabs := Tabs([]TabItem{{Label: "Params", Badge: 3}, {Label: "Body"}}, 0, func(i int) { sel = i }, H(40))
	root := Row(tabs)
	c := testCtx(t)
	Layout(c, root, 400, 40)

	second := tabs.kids[1].rect
	click(c, second.X+second.W/2, 20)
	Update(c, root)
	if sel != 1 {
		t.Fatalf("selected %d, want 1", sel)
	}
}
