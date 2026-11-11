package widget

import (
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
