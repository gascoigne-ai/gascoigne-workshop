package widget

import (
	"errors"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

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

func TestComposedTextReachesAMultilineEditor(t *testing.T) {
	area := NewTextArea("")
	area.Apply([]rune("ça va\n"), nil, false)
	area.Apply([]rune("日本"), nil, false)
	if area.Text != "ça va日本" { // the line break comes from Enter, not from typed text
		t.Fatalf("text %q", area.Text)
	}
}

func TestPressThumbGrabsTheThumbOrPages(t *testing.T) {
	// content 1000 in a view of 100 on a 100px track: the thumb is 10px.
	grabbed, off, pos := pressThumb(100, 1000, 100, 0, 5)
	if !grabbed || off != 5 || pos != 0 {
		t.Fatalf("on the thumb: %v %d %d", grabbed, off, pos)
	}
	grabbed, _, pos = pressThumb(100, 1000, 100, 0, 60)
	if grabbed || pos != 100 {
		t.Fatalf("below the thumb pages down: %v %d", grabbed, pos)
	}
	grabbed, _, pos = pressThumb(100, 1000, 100, 300, 2)
	if grabbed || pos != 200 {
		t.Fatalf("above the thumb pages up: %v %d", grabbed, pos)
	}
	if grabbed, _, pos = pressThumb(100, 50, 100, 0, 5); grabbed || pos != 0 {
		t.Fatal("content that fits has nothing to grab")
	}
}

// ScrollbarSlack allows for the horizontal scrollbar eating a few pixels.
const ScrollbarSlack = scrollbarW

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
	c.Clipboard = &fakeClipboard{text: "lost", err: errTest}
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

func TestFieldTextInsetDefaultsAndCanBeReduced(t *testing.T) {
	f := NewTextField("x")
	plain := Field(&f)
	tight := Field(&f, FieldPad(5))
	plainInset := plain.Behavior().(*fieldB).textInset()
	tightInset := tight.Behavior().(*fieldB).textInset()
	if plainInset != fieldPad || tightInset != 5 {
		t.Fatalf("insets %d %d", plainInset, tightInset)
	}
}

var errTest = errors.New("no clipboard")
