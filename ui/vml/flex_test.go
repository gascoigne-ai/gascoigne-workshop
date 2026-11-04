package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

func TestAlignItemsPlacesChildrenAcrossARow(t *testing.T) {
	tests := []struct {
		name  string
		align Align
		wantY int // for a 20px child in a 100px row
	}{
		{"default centers a fixed-height child", AlignDefault, 40},
		{"start", AlignStart, 0},
		{"center", AlignCenter, 40},
		{"end", AlignEnd, 80},
		{"stretch keeps a fixed height, at the start", AlignStretch, 0},
	}
	for _, tt := range tests {
		child := box(W(30), H(20))
		root := Row(H(100), AlignItems(tt.align), child)
		Layout(testCtx(t), root, 200, 100)
		if child.rect.Y != tt.wantY || child.rect.H != 20 {
			t.Errorf("%s: y %d h %d, want y %d h 20", tt.name, child.rect.Y, child.rect.H, tt.wantY)
		}
	}
}

func TestStretchFillsTheCrossAxisWhenTheChildHasNoFixedSize(t *testing.T) {
	child := box(W(30))
	root := Row(H(100), AlignItems(AlignStretch), child)
	Layout(testCtx(t), root, 200, 100)
	if child.rect.H != 100 || child.rect.Y != 0 {
		t.Fatalf("rect = %+v", child.rect)
	}
}

func TestAlignSelfOverridesTheParent(t *testing.T) {
	a, b := box(W(30), H(20)), box(W(30), H(20), AlignSelf(AlignEnd))
	root := Row(H(100), AlignItems(AlignStart), a, b)
	Layout(testCtx(t), root, 200, 100)
	if a.rect.Y != 0 || b.rect.Y != 80 {
		t.Fatalf("a.y %d b.y %d, want 0 and 80", a.rect.Y, b.rect.Y)
	}
}

func TestAlignItemsInAColumnWorksOnTheHorizontalAxis(t *testing.T) {
	for _, tt := range []struct {
		align Align
		wantX int
	}{{AlignDefault, 0}, {AlignStart, 0}, {AlignCenter, 85}, {AlignEnd, 170}} {
		child := box(W(30), H(20))
		root := Col(AlignItems(tt.align), child)
		Layout(testCtx(t), root, 200, 100)
		if child.rect.X != tt.wantX || child.rect.W != 30 {
			t.Errorf("align %d: x %d w %d, want x %d w 30", tt.align, child.rect.X, child.rect.W, tt.wantX)
		}
	}
}

func TestTextTakesItsOwnHeightWhenNotStretched(t *testing.T) {
	c := testCtx(t)
	lineH := c.LineH(c.F.UI)
	for _, tt := range []struct {
		align Align
		wantY int
	}{{AlignStart, 0}, {AlignCenter, (100 - lineH) / 2}, {AlignEnd, 100 - lineH}} {
		text := Text("hello", AlignSelf(tt.align))
		root := Row(H(100), text)
		Layout(c, root, 200, 100)
		if text.rect.H != lineH || text.rect.Y != tt.wantY {
			t.Errorf("align %d: y %d h %d, want y %d h %d", tt.align, text.rect.Y, text.rect.H, tt.wantY, lineH)
		}
	}
}

func TestBaselineAlignmentLinesUpTextOfDifferentSizes(t *testing.T) {
	c := testCtx(t)
	big, small := Text("Big"), Text("small", Small())
	root := Row(H(60), AlignItems(AlignBaseline), big, small)
	Layout(c, root, 300, 60)
	bigBase := big.rect.Y + c.Ascent(c.F.UI)
	smallBase := small.rect.Y + c.Ascent(c.F.Small)
	if bigBase != smallBase {
		t.Fatalf("baselines %d and %d must match (y %d and %d)", bigBase, smallBase, big.rect.Y, small.rect.Y)
	}
	if c.Ascent(c.F.UI) <= c.Ascent(c.F.Small) {
		t.Fatal("test setup: the UI face must have the larger ascent")
	}
	// the line (the stack of both texts on one baseline) is centered in the row
	asc := c.Ascent(c.F.UI)
	below := max(c.LineH(c.F.UI)-asc, c.LineH(c.F.Small)-c.Ascent(c.F.Small))
	if want := (60 - (asc + below)) / 2; big.rect.Y != want {
		t.Fatalf("the line is centered in the row: y %d, want %d", big.rect.Y, want)
	}
}

func TestJustifyContentDistributesFreeSpace(t *testing.T) {
	tests := []struct {
		name  string
		j     Justify
		wantX [3]int // three 40px children in 300px: 180px free
	}{
		{"start", JustifyStart, [3]int{0, 40, 80}},
		{"center", JustifyCenter, [3]int{90, 130, 170}},
		{"end", JustifyEnd, [3]int{180, 220, 260}},
		{"between", JustifyBetween, [3]int{0, 130, 260}},
		{"around", JustifyAround, [3]int{30, 130, 230}},
	}
	for _, tt := range tests {
		a, b, c := box(W(40)), box(W(40)), box(W(40))
		root := Row(JustifyContent(tt.j), a, b, c)
		Layout(testCtx(t), root, 300, 50)
		got := [3]int{a.rect.X, b.rect.X, c.rect.X}
		if got != tt.wantX {
			t.Errorf("%s: %v, want %v", tt.name, got, tt.wantX)
		}
	}
}

func TestJustifyDoesNothingWhenAChildGrows(t *testing.T) {
	a, b := box(W(40)), box(Grow(1))
	root := Row(JustifyContent(JustifyEnd), a, b)
	Layout(testCtx(t), root, 300, 50)
	if a.rect.X != 0 || b.rect.W != 260 {
		t.Fatalf("a %+v b %+v", a.rect, b.rect)
	}
}

func TestMinAndMaxSizes(t *testing.T) {
	c := testCtx(t)
	short := Text("hi", MinW(120))
	grower := box(Grow(1), MaxW(80))
	tall := box(H(500), MaxH(100))
	root := Col(Row(short, grower), tall)
	Layout(c, root, 400, 600)
	if short.rect.W != 120 {
		t.Errorf("MinW: width %d", short.rect.W)
	}
	if grower.rect.W != 80 {
		t.Errorf("MaxW on a growing child: width %d", grower.rect.W)
	}
	if tall.rect.H != 100 {
		t.Errorf("MaxH: height %d", tall.rect.H)
	}
	if w, _ := Text("a long piece of text", MaxW(30)).size(c, 400); w != 30 {
		t.Errorf("MaxW on content: %d", w)
	}
}

func TestMarginsShiftAndEnlargeTheOuterBox(t *testing.T) {
	first, second := box(W(50), H(20), Margin(5)), box(W(50), H(20))
	root := Row(H(100), AlignItems(AlignStart), first, second)
	Layout(testCtx(t), root, 300, 100)
	if first.rect.X != 5 || first.rect.Y != 5 {
		t.Errorf("first at %d,%d, want 5,5", first.rect.X, first.rect.Y)
	}
	if second.rect.X != 5+50+5 {
		t.Errorf("the next sibling starts after the margin: x %d", second.rect.X)
	}
}

func TestMarginsShrinkAStretchedChild(t *testing.T) {
	child := box(W(50), MarginXY(0, 10))
	root := Row(H(100), child)
	Layout(testCtx(t), root, 300, 100)
	if child.rect.Y != 10 || child.rect.H != 80 {
		t.Fatalf("rect = %+v", child.rect)
	}
}

func TestMarginsCountInAHuggingParentsSize(t *testing.T) {
	child := box(W(50), H(20), Margin(5))
	w, h := Row(child).size(testCtx(t), 400)
	if w != 60 || h != 30 {
		t.Fatalf("hug size %dx%d, want 60x30", w, h)
	}
}

func TestAscentIsPositiveAndShrinksWithTheFont(t *testing.T) {
	c := testCtx(t)
	if c.Ascent(c.F.UI) <= 0 || c.Ascent(c.F.Small) <= 0 || c.Ascent(c.F.UI) > c.LineH(c.F.UI) {
		t.Fatalf("ascents %d %d, line height %d", c.Ascent(c.F.UI), c.Ascent(c.F.Small), c.LineH(c.F.UI))
	}
}

var _ = widget.Rect{}
