package widget

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// sampleLines is this JSON, pretty printed, as lines (index: text):
// 0 {  1   "a": 1,  2   "items": [  3     {  4       "id": 2,
// 5       "o": {  6         "x": 3  7       }  8     },  9     {
// 10      "id": 4  11    }  12  ],  13  "z": 5  14 }
func sampleLines() []Line {
	texts := []struct {
		text     string
		foldable bool
	}{
		{"{", true}, {`  "a": 1,`, false}, {`  "items": [`, true}, {"    {", true},
		{`      "id": 2,`, false}, {`      "o": {`, true}, {`        "x": 3`, false},
		{"      }", false}, {"    },", false}, {"    {", true}, {`      "id": 4`, false},
		{"    }", false}, {"  ],", false}, {`  "z": 5`, false}, {"}", false},
	}
	lines := make([]Line, len(texts))
	for i, t := range texts {
		lines[i] = Line{Spans: []Span{{Text: t.text}}, Foldable: t.foldable}
	}
	return lines
}

func TestCodeViewVisibleFolds(t *testing.T) {
	all := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
	tests := []struct {
		name string
		fold []int
		want []int
	}{
		{"none", nil, all},
		{"fold items", []int{2}, []int{0, 1, 2, 13, 14}},
		{"fold first element", []int{3}, []int{0, 1, 2, 3, 9, 10, 11, 12, 13, 14}},
		{"fold root", []int{0}, []int{0}},
		{"fold inside folded still hidden", []int{2, 5}, []int{0, 1, 2, 13, 14}},
		{"fold on non-foldable ignored", []int{1}, all},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v CodeView
			v.SetLines(sampleLines())
			for _, i := range tt.fold {
				v.ToggleFold(i)
			}
			if got := v.Visible(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeViewToggleTwiceUnfolds(t *testing.T) {
	var v CodeView
	v.SetLines(sampleLines())
	v.ToggleFold(2)
	v.ToggleFold(2)
	if len(v.Visible()) != 15 {
		t.Fatal("second toggle must unfold")
	}
	v.ToggleFold(-1)
	v.ToggleFold(99) // out of range must not panic
}

func TestCodeViewScrollClamps(t *testing.T) {
	tests := []struct {
		name                   string
		start, dy, content, vh int
		want                   int
	}{
		{"down", 0, 50, 1000, 400, 50},
		{"past end", 500, 500, 1000, 400, 600},
		{"above top", 10, -50, 1000, 400, 0},
		{"content shorter than view", 0, 80, 100, 400, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := CodeView{ScrollY: tt.start}
			v.Scroll(tt.dy, tt.content, tt.vh)
			if v.ScrollY != tt.want {
				t.Fatalf("ScrollY = %d, want %d", v.ScrollY, tt.want)
			}
		})
	}
}

func TestCodeViewSelectionText(t *testing.T) {
	var v CodeView
	v.SetLines(sampleLines())
	if v.SelectedText() != "" {
		t.Fatal("nothing selected yet")
	}
	v.SelectLines(1, 2)
	if got, want := v.SelectedText(), "  \"a\": 1,\n  \"items\": ["; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	v.SelectLines(2, 1) // reversed range selects the same lines
	if got := v.SelectedText(); got != "  \"a\": 1,\n  \"items\": [" {
		t.Fatalf("reversed range: %q", got)
	}
	v.SelectLines(-5, 99) // clamped to the content
	if got := v.SelectedText(); got != v.AllText() {
		t.Fatalf("clamped selection should cover everything")
	}
	v.SetLines(nil)
	v.SelectLines(0, 3)
	if v.HasSel || v.SelectedText() != "" {
		t.Fatal("an empty view cannot have a selection")
	}
}

func TestCodeViewSetLinesClearsSelection(t *testing.T) {
	var v CodeView
	v.SetLines(sampleLines())
	v.SelectLines(0, 3)
	v.SetLines(sampleLines())
	if v.HasSel {
		t.Fatal("new content must clear the selection")
	}
}

type memClip struct{ text string }

func (m *memClip) Read() (string, error) { return m.text, nil }
func (m *memClip) Write(s string) error  { m.text = s; return nil }

// viewport is a 600x400 code view at the origin of a context with a memory clipboard.
func viewport(t *testing.T) (*CodeView, *Ctx, *memClip, Rect) {
	t.Helper()
	c, err := NewCtx(Dark())
	if err != nil {
		t.Fatal(err)
	}
	clip := &memClip{}
	c.Clipboard = clip
	v := &CodeView{}
	v.SetLines(sampleLines())
	return v, c, clip, Rect{W: 600, H: 400}
}

// rowY is the pixel row of the middle of visible row i.
func rowY(c *Ctx, v *CodeView, r Rect, i int) int {
	g := v.geometry(c, r)
	return r.Y + i*g.lineH + g.lineH/2
}

func TestCodeViewClickSelectsALineDragExtendsAndCtrlCCopies(t *testing.T) {
	v, c, clip, r := viewport(t)
	textX := 300 // well right of the gutter

	c.In = Input{X: textX, Y: rowY(c, v, r, 2), Click: true, Held: true}
	v.Update(c, r)
	if !v.Focused || v.SelectedText() != "  \"items\": [" {
		t.Fatalf("click: focused %v selection %q", v.Focused, v.SelectedText())
	}

	c.In = Input{X: textX, Y: rowY(c, v, r, 4), Held: true}
	v.Update(c, r)
	c.In = Input{X: textX, Y: rowY(c, v, r, 4)} // released
	v.Update(c, r)
	if v.dragging {
		t.Fatal("releasing the button must end the drag")
	}
	if got, want := v.SelectedText(), v.textOf(2, 4); got != want {
		t.Fatalf("drag selection = %q, want %q", got, want)
	}

	c.In = Input{Ctrl: true, Copy: true}
	v.Update(c, r)
	if clip.text != v.textOf(2, 4) {
		t.Fatalf("clipboard = %q", clip.text)
	}
}

func TestCodeViewCopyWithoutSelectionCopiesEverything(t *testing.T) {
	v, c, clip, r := viewport(t)
	v.Focused = true
	c.In = Input{X: 5000, Y: 5000, Ctrl: true, Copy: true} // pointer elsewhere: only focus matters
	v.Update(c, r)
	if clip.text != v.AllText() {
		t.Fatalf("clipboard = %q", clip.text)
	}
}

func TestCodeViewSelectAllAndBlurOnOutsideClick(t *testing.T) {
	v, c, _, r := viewport(t)
	v.Focused = true
	c.In = Input{X: 5000, Y: 5000, Ctrl: true, SelectAll: true}
	v.Update(c, r)
	if v.SelectedText() != v.AllText() {
		t.Fatal("Ctrl+A must select every line")
	}

	c.In = Input{X: 5000, Y: 5000, Click: true}
	v.Update(c, r)
	if v.Focused {
		t.Fatal("a click outside must blur the view")
	}
	c.In = Input{X: 5000, Y: 5000, Ctrl: true, Copy: true}
	clipBefore := c.Clipboard.(*memClip).text
	v.Update(c, r)
	if c.Clipboard.(*memClip).text != clipBefore {
		t.Fatal("an unfocused view must ignore Ctrl+C")
	}
}

func TestCodeViewFoldArrowToggles(t *testing.T) {
	v, c, _, r := viewport(t)
	g := v.geometry(c, r)
	arrowX := r.X + Pad + g.numW + 4 + foldColW/2

	c.In = Input{X: arrowX, Y: rowY(c, v, r, 0), Click: true}
	v.Update(c, r)
	if len(v.Visible()) != 1 {
		t.Fatalf("folding the root must hide the rest, visible = %v", v.Visible())
	}
	if v.HasSel {
		t.Fatal("a click on the arrow must not select the line")
	}
}

func TestCodeViewWheelScrollsOnlyWhenOver(t *testing.T) {
	v, c, _, _ := viewport(t)
	small := Rect{W: 600, H: 60} // content is taller than the viewport
	c.In = Input{X: 10, Y: 10, Wheel: -1}
	v.Update(c, small)
	if v.ScrollY <= 0 {
		t.Fatalf("wheel over the view must scroll, ScrollY = %d", v.ScrollY)
	}
	before := v.ScrollY
	c.In = Input{X: 900, Y: 10, Wheel: -1}
	v.Update(c, small)
	if v.ScrollY != before {
		t.Fatal("wheel elsewhere must not scroll")
	}
}

func TestCodeViewEmptyContentIsHarmless(t *testing.T) {
	var v CodeView
	c, err := NewCtx(Dark())
	if err != nil {
		t.Fatal(err)
	}
	c.In = Input{X: 10, Y: 10, Click: true, Held: true, Ctrl: true, Copy: true, SelectAll: true}
	v.Update(c, Rect{W: 100, H: 100}) // must not panic on zero lines
	if v.HasSel {
		t.Fatal("nothing to select")
	}
}

func TestCtxSetThemeSwitchesColorsAndFontSizes(t *testing.T) {
	c, err := NewCtx(Dark())
	if err != nil {
		t.Fatal(err)
	}
	small := Light().WithTextSize(10, 8, 10)
	if err := c.SetTheme(small); err != nil {
		t.Fatal(err)
	}
	if c.Col(Bg) != Light().Color(Bg) {
		t.Fatal("colors did not switch")
	}
	if c.F.UI.Size != 10 {
		t.Fatalf("font size = %v, want 10 (times scale 1)", c.F.UI.Size)
	}
	if err := c.SetScale(2); err != nil || c.F.UI.Size != 20 {
		t.Fatalf("after SetScale(2) the face must be 20px, got %v err %v", c.F.UI.Size, err)
	}
}

func longLines() []Line {
	long := strings.Repeat("0123456789", 30) // 300 chars, far wider than the view
	return []Line{{Spans: []Span{{Text: long}}}, {Spans: []Span{{Text: "short"}}}}
}

func TestCodeViewScrollsHorizontallyWithShiftWheelAndClamps(t *testing.T) {
	v, c, _, _ := viewport(t)
	v.SetLines(longLines())
	r := Rect{W: 300, H: 100}

	c.In = Input{X: 50, Y: 20, Wheel: -1, Shift: true}
	v.Update(c, r)
	if v.ScrollX <= 0 {
		t.Fatalf("Shift+wheel must scroll sideways, ScrollX = %d", v.ScrollX)
	}
	if v.ScrollY != 0 {
		t.Fatal("Shift+wheel must not scroll vertically")
	}
	for range 200 {
		c.In = Input{X: 50, Y: 20, Wheel: -1, Shift: true}
		v.Update(c, r)
	}
	if max := v.contentWidth(c, r) - r.W; v.ScrollX != max {
		t.Fatalf("ScrollX = %d, want the clamp %d", v.ScrollX, max)
	}
	c.In = Input{X: 50, Y: 20, Wheel: 1, Shift: true}
	v.Update(c, r)
	if v.ScrollX >= v.contentWidth(c, r)-r.W {
		t.Fatal("wheel up with Shift must scroll back")
	}
}

func TestCodeViewShortLinesNeverScrollSideways(t *testing.T) {
	v, c, _, _ := viewport(t)
	r := Rect{W: 600, H: 400}
	c.In = Input{X: 50, Y: 20, Wheel: -3, Shift: true}
	v.Update(c, r)
	if v.ScrollX != 0 {
		t.Fatalf("ScrollX = %d, want 0", v.ScrollX)
	}
}

func TestCodeViewPagingKeysWhenFocused(t *testing.T) {
	v, c, _, _ := viewport(t)
	v.SetLines(manyLines(100))
	r := Rect{W: 600, H: 100}
	v.Focused = true
	g := v.geometry(c, r)
	pressed := func(k ebiten.Key) {
		c.In = Input{X: 5000, Y: 5000, Keys: []ebiten.Key{k}}
		v.Update(c, r)
	}

	pressed(ebiten.KeyPageDown)
	if v.ScrollY != r.H-g.lineH {
		t.Fatalf("PageDown moves one page less a line, ScrollY = %d", v.ScrollY)
	}
	pressed(ebiten.KeyEnd)
	if want := 100*g.lineH - r.H; v.ScrollY != want {
		t.Fatalf("End = %d, want %d", v.ScrollY, want)
	}
	pressed(ebiten.KeyPageUp)
	pressed(ebiten.KeyHome)
	if v.ScrollY != 0 {
		t.Fatalf("Home = %d", v.ScrollY)
	}

	v.Focused = false
	pressed(ebiten.KeyPageDown)
	if v.ScrollY != 0 {
		t.Fatal("an unfocused view must ignore paging keys")
	}
}

func manyLines(n int) []Line {
	lines := make([]Line, n)
	for i := range lines {
		lines[i] = Line{Spans: []Span{{Text: "row"}}}
	}
	return lines
}

func TestCodeViewScrollbarThumbDragScrollsTheContent(t *testing.T) {
	v, c, _, _ := viewport(t)
	v.SetLines(manyLines(100))
	r := Rect{W: 400, H: 200}
	g := v.geometry(c, r)
	track := Rect{X: r.W - ScrollbarW, Y: 0, W: ScrollbarW, H: r.H}
	off, length, ok := Thumb(track.H, 100*g.lineH, r.H, v.ScrollY)
	if !ok {
		t.Fatal("100 rows in 200px must need a scrollbar")
	}
	grabY := off + length/2

	c.In = Input{X: track.X + 2, Y: grabY, Click: true, Held: true}
	v.Update(c, r)
	if v.ScrollY != 0 || v.Selected() {
		t.Fatalf("grabbing the thumb must not scroll or select, ScrollY %d", v.ScrollY)
	}
	c.In = Input{X: track.X + 2, Y: grabY + 50, Held: true}
	v.Update(c, r)
	if v.ScrollY <= 0 {
		t.Fatalf("dragging the thumb down must scroll down, ScrollY = %d", v.ScrollY)
	}
	pos := v.ScrollY
	c.In = Input{X: track.X + 2, Y: grabY + 50} // released
	v.Update(c, r)
	c.In = Input{X: track.X + 2, Y: grabY + 90, Held: true}
	v.Update(c, r)
	if v.ScrollY != pos {
		t.Fatal("after release, moving the pointer must not scroll")
	}
}
