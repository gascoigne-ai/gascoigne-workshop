package view

import (
	"errors"
	"reflect"
	"strings"
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
	if got, want := tbl.rect.H, 4*core.RowH; got != want {
		t.Fatalf("table height = %d, want %d", got, want)
	}
	rowY := func(i int) int { return (i+1)*core.RowH + core.RowH/2 }
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

type fakeClipboard struct {
	text    string
	readErr error
}

func (f *fakeClipboard) Read() (string, error) { return f.text, f.readErr }
func (f *fakeClipboard) Write(s string) error  { f.text = s; return nil }

func TestFieldCopyCutPasteSelectAll(t *testing.T) {
	changes := 0
	f := NewTextField("hello world")
	f.Focused = true
	field := Field(&f, OnChange(func() { changes++ }))
	root := Row(field)
	c := testCtx(t)
	clip := &fakeClipboard{}
	c.Clipboard = clip
	Layout(c, root, 300, 30)

	step := func(in core.Input) {
		c.In = in
		Update(c, root)
	}

	step(core.Input{Ctrl: true, SelectAll: true})
	step(core.Input{Ctrl: true, Copy: true})
	if clip.text != "hello world" || f.Text != "hello world" || changes != 0 {
		t.Fatalf("copy: clipboard %q text %q changes %d", clip.text, f.Text, changes)
	}

	step(core.Input{Ctrl: true, Cut: true})
	if f.Text != "" || clip.text != "hello world" || changes != 1 {
		t.Fatalf("cut: clipboard %q text %q changes %d", clip.text, f.Text, changes)
	}

	clip.text = "new\r\nvalue"
	step(core.Input{Ctrl: true, Paste: true})
	if f.Text != "new  value" || changes != 2 {
		t.Fatalf("paste: text %q changes %d", f.Text, changes)
	}

	// Ctrl+V must not also type the letter the key reports.
	step(core.Input{Ctrl: true, Typed: []rune("v")})
	if f.Text != "new  value" {
		t.Fatalf("ctrl+letter typed text: %q", f.Text)
	}
}

func TestFieldPasteFailureLeavesTextAlone(t *testing.T) {
	f := NewTextField("keep")
	f.Focused = true
	root := Row(Field(&f))
	c := testCtx(t)
	c.Clipboard = &fakeClipboard{text: "lost", readErr: errTest}
	Layout(c, root, 300, 30)
	c.In = core.Input{Ctrl: true, Paste: true}
	Update(c, root)
	if f.Text != "keep" {
		t.Fatalf("text = %q", f.Text)
	}
}

func TestFieldDragSelectsAndReleaseEndsTheDrag(t *testing.T) {
	f := NewTextField("hello")
	root := Row(Field(&f, Grow(1)))
	c := testCtx(t)
	Layout(c, root, 300, 30)

	// press at the start of the text, drag far past its end, release
	c.In = core.Input{X: fieldPad, Y: 10, Click: true, Held: true}
	Update(c, root)
	c.In = core.Input{X: fieldPad + 200, Y: 10, Held: true}
	Update(c, root)
	c.In = core.Input{X: fieldPad + 200, Y: 10}
	Update(c, root)

	if f.SelectedText() != "hello" || f.Dragging {
		t.Fatalf("selection %q dragging %v", f.SelectedText(), f.Dragging)
	}
}

var errTest = errors.New("no clipboard")

func TestEnterInAFocusedFieldFiresOnEnterOnly(t *testing.T) {
	enters := 0
	f := NewTextField("x")
	root := Row(Field(&f, Grow(1), Placeholder("p"), OnEnter(func() { enters++ })))
	c := testCtx(t)
	Layout(c, root, 200, 30)

	c.In = core.Input{Enter: true}
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

func TestCodeNodeForwardsInputToTheView(t *testing.T) {
	v := &CodeView{}
	v.SetLines([]Line{{Spans: []Span{{Text: "one"}}}, {Spans: []Span{{Text: "two"}}}})
	root := Row(Code(v, Grow(1)))
	c := testCtx(t)
	Layout(c, root, 400, 100)

	c.In = core.Input{X: 300, Y: 15, Click: true, Held: true, Clicks: 3} // triple click in the text area of the first line
	Update(c, root)
	if v.SelectedText() != "one" || !v.Focused {
		t.Fatalf("selection %q focused %v", v.SelectedText(), v.Focused)
	}
}

func TestIconsHaveTheirNaturalWidths(t *testing.T) {
	c := testCtx(t)
	icons := Row(Checkbox(true), CrossIcon(), TriangleIcon(false), DotsIcon())
	Layout(c, icons, 200, 30)
	want := []int{14, 10, 10, 4}
	for i, k := range icons.kids {
		if k.rect.W != want[i] {
			t.Errorf("icon %d width %d, want %d", i, k.rect.W, want[i])
		}
	}
}

func editableSpec() TableSpec {
	return TableSpec{
		Columns: []string{"Name", "Value"}, Weights: []float64{1, 1},
		Rows:        []TableRow{{Cells: []string{"q", "x"}}, {Cells: []string{"o", "y"}}},
		Placeholder: []string{"Name", "Value"},
		Editable:    true,
		OnToggle:    func(int) {}, OnDelete: func(int) {},
	}
}

// Columns of a 400px editable table: 36px actions, then two 164px cells.
const cellX0, cellX1 = 36 + 80, 36 + 164 + 80

func cellY(row int) int { return (row+1)*core.RowH + core.RowH/2 }

func TestClickingACellStartsEditingIt(t *testing.T) {
	var got [][2]int
	spec := editableSpec()
	spec.OnEditCell = func(row, col int) { got = append(got, [2]int{row, col}) }
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)

	click(c, cellX1, cellY(1))
	Update(c, root)
	click(c, cellX0, cellY(2)) // the placeholder row is editable too
	Update(c, root)

	if len(got) != 2 || got[0] != [2]int{1, 1} || got[1] != [2]int{2, 0} {
		t.Fatalf("edit requests = %v, want [[1 1] [2 0]]", got)
	}
}

func TestReadOnlyTableCellsIgnoreClicks(t *testing.T) {
	called := false
	spec := editableSpec()
	spec.Editable = false
	spec.OnEditCell = func(int, int) { called = true }
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)
	click(c, cellX0-36, cellY(0)) // read-only tables have no action column: shift left
	Update(c, root)
	if called {
		t.Fatal("a read-only table must not start edits")
	}
}

func TestEditedCellShowsAFieldWithEnterAndTab(t *testing.T) {
	var enters int
	var tabs []bool
	f := NewTextField("x")
	f.Focused = true
	spec := editableSpec()
	spec.Editing = &TableEdit{Row: 0, Col: 1, Field: &f,
		OnEnter: func() { enters++ }, OnTab: func(back bool) { tabs = append(tabs, back) }}
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)

	c.In = core.Input{Typed: []rune("yz")}
	Update(c, root)
	if f.Text != "xyz" {
		t.Fatalf("typing reached the field as %q, want xyz", f.Text)
	}
	c.In = core.Input{Enter: true}
	Update(c, root)
	c.In = core.Input{Tab: true}
	Update(c, root)
	c.In = core.Input{Tab: true, Shift: true}
	Update(c, root)
	if enters != 1 || len(tabs) != 2 || tabs[0] || !tabs[1] {
		t.Fatalf("enters %d tabs %v, want 1 and [false true]", enters, tabs)
	}
}

func focusedField(t *testing.T, text string, width int) (*TextField, *core.Ctx, *El, *int) {
	t.Helper()
	f := NewTextField(text)
	f.Focused = true
	changes := 0
	root := Row(Field(&f, Grow(1), OnChange(func() { changes++ })))
	c := testCtx(t)
	Layout(c, root, width, 30)
	return &f, c, root, &changes
}

func TestFieldUndoRedoShortcuts(t *testing.T) {
	f, c, root, changes := focusedField(t, "", 300)
	c.In = core.Input{Typed: []rune("abc")}
	Update(c, root)
	if f.Text != "abc" || *changes != 1 {
		t.Fatalf("typed: %q changes %d", f.Text, *changes)
	}

	c.In = core.Input{Ctrl: true, Undo: true}
	Update(c, root)
	if f.Text != "" || *changes != 2 {
		t.Fatalf("undo: %q changes %d (undo must report a change)", f.Text, *changes)
	}
	c.In = core.Input{Ctrl: true, Redo: true}
	Update(c, root)
	if f.Text != "abc" || *changes != 3 {
		t.Fatalf("redo: %q changes %d", f.Text, *changes)
	}
}

func TestDoubleAndTripleClickSelectWordAndAll(t *testing.T) {
	f, c, root, _ := focusedField(t, "alpha beta", 300)
	inWord := fieldPad + c.Width(c.F.UI, "alpha b") // inside "beta"

	c.In = core.Input{X: inWord, Y: 10, Click: true, Held: true, Clicks: 1}
	Update(c, root)
	c.In = core.Input{X: inWord, Y: 10, Click: true, Held: true, Clicks: 2}
	Update(c, root)
	if f.SelectedText() != "beta" {
		t.Fatalf("double click selected %q, want beta", f.SelectedText())
	}
	c.In = core.Input{X: inWord, Y: 10, Click: true, Held: true, Clicks: 3}
	Update(c, root)
	if f.SelectedText() != "alpha beta" {
		t.Fatalf("triple click selected %q", f.SelectedText())
	}
}

func TestLongTextScrollsToTheCursorAndClicksAccountForIt(t *testing.T) {
	long := strings.Repeat("w", 80)
	f, c, root, _ := focusedField(t, long, 120)

	c.In = core.Input{Typed: []rune("z")}
	Update(c, root)
	if f.ScrollX <= 0 {
		t.Fatalf("the cursor at the end of long text must scroll the view, ScrollX = %d", f.ScrollX)
	}

	c.In = core.Input{X: fieldPad, Y: 10, Click: true, Held: true, Clicks: 1}
	Update(c, root)
	if f.Cursor < 40 {
		t.Fatalf("a click at the left edge of the scrolled view hit position %d; the scroll offset must count", f.Cursor)
	}

	f.Cursor, f.HasSel = 0, false
	c.In = core.Input{Edits: []core.Edit{core.EditHome}}
	Update(c, root)
	if f.ScrollX != 0 {
		t.Fatalf("Home must scroll back to the start, ScrollX = %d", f.ScrollX)
	}
}

func focusFlags(f1, f2 *TextField, v *CodeView) [3]bool {
	return [3]bool{f1.Focused, f2.Focused, v.Focused}
}

func TestTabCyclesFocusThroughFieldsAndCodeViews(t *testing.T) {
	a, b := NewTextField("a"), NewTextField("b")
	v := &CodeView{}
	v.SetLines([]Line{{Spans: []Span{{Text: "x"}}}})
	root := Col(Field(&a, H(30)), Field(&b, H(30)), Code(v, H(30)))
	c := testCtx(t)
	Layout(c, root, 200, 100)

	tab := func(shift bool) {
		c.In = core.Input{Tab: true, Shift: shift}
		Update(c, root)
	}
	steps := []struct {
		shift bool
		want  [3]bool
	}{
		{false, [3]bool{true, false, false}}, // nothing focused: Tab starts at the first
		{false, [3]bool{false, true, false}},
		{false, [3]bool{false, false, true}},
		{false, [3]bool{true, false, false}}, // wraps
		{true, [3]bool{false, false, true}},  // Shift+Tab goes back, wrapping
		{true, [3]bool{false, true, false}},
	}
	for i, st := range steps {
		tab(st.shift)
		if got := focusFlags(&a, &b, v); got != st.want {
			t.Fatalf("step %d: focus %v, want %v", i, got, st.want)
		}
	}
}

func TestTabSelectsTheTextOfTheFieldItFocuses(t *testing.T) {
	a := NewTextField("hello")
	root := Col(Field(&a, H(30)))
	c := testCtx(t)
	Layout(c, root, 200, 30)
	c.In = core.Input{Tab: true}
	Update(c, root)
	if a.SelectedText() != "hello" {
		t.Fatalf("selection %q", a.SelectedText())
	}
}

func TestTabIsLeftToAFieldThatHandlesIt(t *testing.T) {
	a, b := NewTextField("a"), NewTextField("b")
	a.Focused = true
	var tabs int
	root := Col(Field(&a, H(30), OnTab(func(bool) { tabs++ })), Field(&b, H(30)))
	c := testCtx(t)
	Layout(c, root, 200, 60)
	c.In = core.Input{Tab: true}
	Update(c, root)
	if tabs != 1 || !a.Focused || b.Focused {
		t.Fatalf("handler calls %d, focus a=%v b=%v: a field with OnTab keeps focus", tabs, a.Focused, b.Focused)
	}
}

func TestCtrlTabIsNotFocusCycling(t *testing.T) {
	a := NewTextField("a")
	root := Col(Field(&a, H(30)))
	c := testCtx(t)
	Layout(c, root, 200, 30)
	c.In = core.Input{Tab: true, Ctrl: true}
	Update(c, root)
	if a.Focused {
		t.Fatal("Ctrl+Tab switches tabs in the app, it must not move focus")
	}
}

// areaBox is a 400x100 text area in a window of the same size.
func areaBox(t *testing.T, text string) (*TextArea, *core.Ctx, *El, *int) {
	t.Helper()
	area := NewTextArea(text)
	a := &area
	changes := 0
	root := Row(Area(a, Grow(1), OnChange(func() { changes++ })))
	c := testCtx(t)
	Layout(c, root, 400, 100)
	return a, c, root, &changes
}

func focusArea(a *TextArea, c *core.Ctx, root *El) {
	c.In = core.Input{X: 380, Y: 5, Click: true, Held: true, Clicks: 1}
	Update(c, root)
	c.In = core.Input{X: 380, Y: 5}
	Update(c, root)
	_ = a
}

func TestTextAreaTypingEnterAndUndo(t *testing.T) {
	a, c, root, changes := areaBox(t, "")
	focusArea(a, c, root)
	if !a.Focused {
		t.Fatal("a click must focus the area")
	}
	c.In = core.Input{Typed: []rune("hi")}
	Update(c, root)
	c.In = core.Input{Enter: true}
	Update(c, root)
	c.In = core.Input{Typed: []rune("yo")}
	Update(c, root)
	if a.Text != "hi\nyo" || *changes != 3 {
		t.Fatalf("text %q changes %d", a.Text, *changes)
	}
	c.In = core.Input{Ctrl: true, Undo: true}
	Update(c, root)
	if a.Text != "hi\n" {
		t.Fatalf("undo: %q", a.Text)
	}
}

func TestTextAreaClickPlacesTheCursorByLineAndColumn(t *testing.T) {
	a, c, root, _ := areaBox(t, "abc\ndef\nghi")
	lineH := c.LineH(c.F.Mono) + 4

	c.In = core.Input{X: 390, Y: lineH + 3, Click: true, Held: true, Clicks: 1}
	Update(c, root)
	if l, col := a.LineCol(a.Cursor); l != 1 || col != 3 {
		t.Fatalf("far right on line 1: %d,%d want 1,3", l, col)
	}
	c.In = core.Input{X: 2, Y: 2*lineH + 3, Click: true, Held: true, Clicks: 1} // in the gutter
	Update(c, root)
	if l, col := a.LineCol(a.Cursor); l != 2 || col != 0 {
		t.Fatalf("left edge on line 2: %d,%d want 2,0", l, col)
	}
}

func TestTextAreaDragSelectsAcrossLines(t *testing.T) {
	a, c, root, _ := areaBox(t, "abc\ndef\nghi")
	lineH := c.LineH(c.F.Mono) + 4
	c.In = core.Input{X: 390, Y: 3, Click: true, Held: true, Clicks: 1}
	Update(c, root)
	c.In = core.Input{X: 390, Y: lineH + 3, Held: true}
	Update(c, root)
	c.In = core.Input{X: 390, Y: lineH + 3}
	Update(c, root)
	if a.SelectedText() != "\ndef" || a.Dragging {
		t.Fatalf("selection %q dragging %v", a.SelectedText(), a.Dragging)
	}
}

func TestTextAreaTabIndentsAndKeepsFocus(t *testing.T) {
	a, c, _, _ := areaBox(t, "x")
	b := NewTextField("other")
	root := Col(Area(a, H(50)), Field(&b, H(30)))
	Layout(c, root, 400, 80)
	a.Focused = true

	c.In = core.Input{Tab: true}
	Update(c, root)
	if a.Text != "x " || !a.Focused || b.Focused {
		t.Fatalf("Tab: text %q area focused %v field focused %v; Tab belongs to the editor", a.Text, a.Focused, b.Focused)
	}
	a.Text, a.Cursor = "  x", 3
	c.In = core.Input{Tab: true, Shift: true}
	Update(c, root)
	if a.Text != "x" {
		t.Fatalf("Shift+Tab: %q", a.Text)
	}
}

func TestTextAreaEscapeBlurs(t *testing.T) {
	a, c, root, _ := areaBox(t, "x")
	a.Focused = true
	c.In = core.Input{Escape: true}
	Update(c, root)
	if a.Focused {
		t.Fatal("Escape must leave the editor")
	}
}

func TestTextAreaScrollsWithWheelAndFollowsTheCursor(t *testing.T) {
	var text string
	for i := 0; i < 100; i++ {
		text += "line\n"
	}
	a, c, root, _ := areaBox(t, text)
	c.In = core.Input{X: 50, Y: 20, Wheel: -2}
	Update(c, root)
	if a.ScrollY <= 0 {
		t.Fatalf("wheel must scroll, ScrollY = %d", a.ScrollY)
	}

	a.Focused = true
	a.Cursor = 0
	c.In = core.Input{Edits: []core.Edit{core.EditDocEnd}}
	Update(c, root)
	lineH := c.LineH(c.F.Mono) + 4
	if want := 101*lineH - 100; a.ScrollY < want-ScrollbarSlack || a.ScrollY > want+ScrollbarSlack {
		t.Fatalf("ScrollY = %d after Ctrl+End, want about %d (the cursor line must be in view)", a.ScrollY, want)
	}
}

// ScrollbarSlack allows for the horizontal scrollbar eating a few pixels.
const ScrollbarSlack = ScrollbarW

func TestTextAreaPasteKeepsLines(t *testing.T) {
	a, c, root, _ := areaBox(t, "")
	a.Focused = true
	c.Clipboard = &fakeClipboard{text: "{\r\n  \"a\": 1\r\n}"}
	c.In = core.Input{Ctrl: true, Paste: true}
	Update(c, root)
	if a.Text != "{\n  \"a\": 1\n}" || a.LineCount() != 3 {
		t.Fatalf("text %q", a.Text)
	}
}

func TestTextAreaShiftWheelScrollsSideways(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "w"
	}
	a, c, root, _ := areaBox(t, long)
	c.In = core.Input{X: 50, Y: 20, Wheel: -2, Shift: true}
	Update(c, root)
	if a.ScrollX <= 0 || a.ScrollY != 0 {
		t.Fatalf("ScrollX %d ScrollY %d", a.ScrollX, a.ScrollY)
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

func TestFocusScopeTrapsTab(t *testing.T) {
	outside, in1, in2 := NewTextField("o"), NewTextField("1"), NewTextField("2")
	root := Stack(
		Col(Field(&outside, H(30))),
		Col(FocusScope(), FloatAt(0, 40), Field(&in1, H(30)), Field(&in2, H(30))),
	)
	c := testCtx(t)
	Layout(c, root, 200, 150)
	tab := func(shift bool) {
		c.In = core.Input{Tab: true, Shift: shift}
		Update(c, root)
	}
	flags := func() [3]bool { return [3]bool{outside.Focused, in1.Focused, in2.Focused} }

	tab(false)
	if flags() != [3]bool{false, true, false} {
		t.Fatalf("Tab must start inside the scope, focus %v", flags())
	}
	tab(false)
	tab(false) // wraps inside the scope, never to the field outside
	if flags() != [3]bool{false, true, false} {
		t.Fatalf("Tab must wrap inside the scope, focus %v", flags())
	}
	tab(true)
	if flags() != [3]bool{false, false, true} {
		t.Fatalf("Shift+Tab, focus %v", flags())
	}
}

func TestOnRightClickGetsThePointer(t *testing.T) {
	var gx, gy, n int
	root := Row(box(W(100), OnRightClick(func(x, y int) { gx, gy, n = x, y, n+1 })))
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

func TestTableRowRightClickReportsTheRowButNotThePlaceholder(t *testing.T) {
	var rows []int
	spec := editableSpec()
	spec.OnRowMenu = func(row, x, y int) { rows = append(rows, row) }
	root := Col(Table(spec))
	c := testCtx(t)
	Layout(c, root, 400, 400)
	for _, row := range []int{0, 1, 2} { // 2 is the placeholder row
		c.In = core.Input{X: 150, Y: cellY(row), RightClick: true}
		Update(c, root)
	}
	if !reflect.DeepEqual(rows, []int{0, 1}) {
		t.Fatalf("rows = %v", rows)
	}
}

func TestBorderLeftOptionRecordsTheTokenAndInset(t *testing.T) {
	e := box(BorderL(core.BorderSoft), BorderInset(6))
	if e.borderL != core.BorderSoft || e.borderInset != 6 {
		t.Fatalf("%q %d", e.borderL, e.borderInset)
	}
}

func TestFieldTextInsetDefaultsAndCanBeReduced(t *testing.T) {
	f := NewTextField("x")
	plain := Field(&f)
	tight := Field(&f, FieldPad(5))
	if plain.textInset() != fieldPad || tight.textInset() != 5 {
		t.Fatalf("insets %d %d", plain.textInset(), tight.textInset())
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
	root := Row(box(W(100), OnClickAt(func(x, y int) { gx, gy, n = x, y, n+1 })))
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

func TestCheckboxOnChangeRunsWhenTheBoxIsClicked(t *testing.T) {
	var changes int
	root := Row(Checkbox(true, W(20), H(20), OnChange(func() { changes++ })))
	c := testCtx(t)
	Layout(c, root, 100, 20)
	c.In = core.Input{X: 60, Y: 10, Click: true} // beside the box
	Update(c, root)
	c.In = core.Input{X: 10, Y: 10, Click: true}
	Update(c, root)
	if changes != 1 {
		t.Fatalf("OnChange ran %d times, want 1", changes)
	}
}
