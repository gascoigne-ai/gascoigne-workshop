package vml

import (
	"errors"
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

type fakeClipboard struct {
	text    string
	readErr error
}

func (f *fakeClipboard) Read() (string, error) { return f.text, f.readErr }
func (f *fakeClipboard) Write(s string) error  { f.text = s; return nil }

func TestFieldCopyCutPasteSelectAll(t *testing.T) {
	changes := 0
	f := widget.NewTextField("hello world")
	f.Focused = true
	field := Field(&f, OnChange(func() { changes++ }))
	root := Row(field)
	c := testCtx(t)
	clip := &fakeClipboard{}
	c.Clipboard = clip
	Layout(c, root, 300, 30)

	step := func(in widget.Input) {
		c.In = in
		Update(c, root)
	}

	step(widget.Input{Ctrl: true, SelectAll: true})
	step(widget.Input{Ctrl: true, Copy: true})
	if clip.text != "hello world" || f.Text != "hello world" || changes != 0 {
		t.Fatalf("copy: clipboard %q text %q changes %d", clip.text, f.Text, changes)
	}

	step(widget.Input{Ctrl: true, Cut: true})
	if f.Text != "" || clip.text != "hello world" || changes != 1 {
		t.Fatalf("cut: clipboard %q text %q changes %d", clip.text, f.Text, changes)
	}

	clip.text = "new\r\nvalue"
	step(widget.Input{Ctrl: true, Paste: true})
	if f.Text != "new  value" || changes != 2 {
		t.Fatalf("paste: text %q changes %d", f.Text, changes)
	}

	// Ctrl+V must not also type the letter the key reports.
	step(widget.Input{Ctrl: true, Typed: []rune("v")})
	if f.Text != "new  value" {
		t.Fatalf("ctrl+letter typed text: %q", f.Text)
	}
}

func TestFieldPasteFailureLeavesTextAlone(t *testing.T) {
	f := widget.NewTextField("keep")
	f.Focused = true
	root := Row(Field(&f))
	c := testCtx(t)
	c.Clipboard = &fakeClipboard{text: "lost", readErr: errTest}
	Layout(c, root, 300, 30)
	c.In = widget.Input{Ctrl: true, Paste: true}
	Update(c, root)
	if f.Text != "keep" {
		t.Fatalf("text = %q", f.Text)
	}
}

func TestFieldDragSelectsAndReleaseEndsTheDrag(t *testing.T) {
	f := widget.NewTextField("hello")
	root := Row(Field(&f, Grow(1)))
	c := testCtx(t)
	Layout(c, root, 300, 30)

	// press at the start of the text, drag far past its end, release
	c.In = widget.Input{X: fieldPad, Y: 10, Click: true, Held: true}
	Update(c, root)
	c.In = widget.Input{X: fieldPad + 200, Y: 10, Held: true}
	Update(c, root)
	c.In = widget.Input{X: fieldPad + 200, Y: 10}
	Update(c, root)

	if f.SelectedText() != "hello" || f.Dragging {
		t.Fatalf("selection %q dragging %v", f.SelectedText(), f.Dragging)
	}
}

var errTest = errors.New("no clipboard")

func TestButtonsClickAndOverrideDefaults(t *testing.T) {
	var plain, primary int
	b1 := Button("Cancel", W(80), OnClick(func() { plain++ }))
	b2 := PrimaryButton("Send", W(80), Bg(widget.Error), OnClick(func() { primary++ }))
	root := Row(b1, b2)
	c := testCtx(t)
	Layout(c, root, 200, 30)

	click(c, 10, 10)
	click(c, 10, 10)
	Update(c, root)
	click(c, 100, 10)
	Update(c, root)
	if plain != 1 || primary != 1 {
		t.Fatalf("plain %d primary %d, want 1 and 1", plain, primary)
	}
	if b2.bg != widget.Error {
		t.Fatalf("caller options must override the button's defaults, bg = %q", b2.bg)
	}
	if b1.bg != widget.Chip {
		t.Fatalf("a plain button uses the chip color, bg = %q", b1.bg)
	}
}

func TestEnterInAFocusedFieldFiresOnEnterOnly(t *testing.T) {
	enters := 0
	f := widget.NewTextField("x")
	root := Row(Field(&f, Grow(1), Placeholder("p"), OnEnter(func() { enters++ })))
	c := testCtx(t)
	Layout(c, root, 200, 30)

	c.In = widget.Input{Enter: true}
	Update(c, root)
	if enters != 0 {
		t.Fatal("an unfocused field must ignore Enter")
	}
	f.Focused = true
	Update(c, root)
	if enters != 1 {
		t.Fatalf("enters = %d, want 1", enters)
	}
}

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
	lines := Lines("a\nb\nc", widget.Muted)
	Layout(testCtx(t), Col(lines), 100, 200)
	if len(lines.kids) != 3 || lines.rect.H != 3*22 {
		t.Fatalf("rows %d height %d, want 3 rows of 22px", len(lines.kids), lines.rect.H)
	}
	for _, k := range lines.kids {
		if k.fg != widget.Muted {
			t.Fatalf("line color = %q", k.fg)
		}
	}
}

func TestCodeNodeForwardsInputToTheView(t *testing.T) {
	v := &widget.CodeView{}
	v.SetLines([]widget.Line{{Spans: []widget.Span{{Text: "one"}}}, {Spans: []widget.Span{{Text: "two"}}}})
	root := Row(Code(v, Grow(1)))
	c := testCtx(t)
	Layout(c, root, 400, 100)

	click(c, 300, 15) // in the text area of the first line
	Update(c, root)
	if v.SelectedText() != "one" || !v.Focused {
		t.Fatalf("selection %q focused %v", v.SelectedText(), v.Focused)
	}
}

func TestEitherAndIconsMeasure(t *testing.T) {
	c := testCtx(t)
	icons := Row(Checkbox(true), Cross(), Triangle(false), Dots())
	Layout(c, icons, 200, 30)
	want := []int{14, 10, 10, 4}
	for i, k := range icons.kids {
		if k.rect.W != want[i] {
			t.Errorf("icon %d width %d, want %d", i, k.rect.W, want[i])
		}
	}
	if Either(true, box(W(1)), box(W(2))).w != 1 {
		t.Fatal("Either(true) picks the first")
	}
}
