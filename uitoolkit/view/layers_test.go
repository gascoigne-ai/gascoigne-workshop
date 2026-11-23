package view

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// overPainter is a Behavior that paints over its children.
type overPainter struct {
	Base
	painted *int
}

func (painter overPainter) DrawOver(*El, *core.Ctx, *ebiten.Image) { *painter.painted++ }

func overlay() *El { return Col(W(60), H(40), FloatAt(10, 10), Modal(), OnDismiss(func() {}), box()) }

func TestModalLayer(t *testing.T) {
	hiddenModal := overlay()
	hiddenModal.hidden = true
	painted := 0
	tests := []struct {
		name string
		root *El
		want int
	}{
		{"no modal", Stack(box(), box()), -1},
		{"a modal at index 2", Stack(box(), box(), overlay(), box()), 2},
		{"a lone modal such as a context menu", Stack(box(), overlay()), 1},
		{"two modals give the first", Stack(box(), overlay(), box(), overlay()), 1},
		{"a hidden modal is skipped", Stack(box(), hiddenModal, box(), overlay()), 3},
		{"a modal nested in a child", Stack(box(), Col(overlay())), -1},
		{"a root with a border", Stack(Border(core.Border), box(), overlay()), -1},
		{"a root with an edge", Stack(EdgeT(core.Border), box(), overlay()), -1},
		{"a root that paints over its children", newEl(overPainter{painted: &painted}, []Item{box(), overlay()}), -1},
		{"a modal root", Col(Modal(), box(), overlay()), -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ModalLayer(testCtx(t), tc.root); got != tc.want {
				t.Fatalf("ModalLayer = %d, want %d", got, tc.want)
			}
		})
	}
}

func layeredRoot() *El {
	return Stack(
		Col(Bg(core.Panel), box(Bg(core.Border)), box(Grow(1), Tooltip("base"))),
		box(Bg(core.Shadow)),
		Col(W(60), H(40), FloatAt(10, 10), Modal(), OnDismiss(func() {}), Bg(core.Panel), box(Grow(1), Tooltip("top"))),
		box(W(5), H(5), FloatAt(0, 0)),
	)
}

func countedDraw(ctx *core.Ctx, draw func()) int {
	ctx.Stats = &core.DrawStats{}
	draw()
	return ctx.Stats.Elements
}

func TestDrawBelowAndDrawFromDrawWhatDrawDraws(t *testing.T) {
	ctx := testCtx(t)
	root := layeredRoot()
	Layout(ctx, root, 200, 100)
	dst := ebiten.NewImage(200, 100)
	const layer = 2

	whole := countedDraw(ctx, func() { Draw(ctx, root, dst) })
	below := countedDraw(ctx, func() { DrawBelow(ctx, root, layer, dst) })
	from := countedDraw(ctx, func() { DrawFrom(ctx, root, layer, dst) })

	if whole == 0 || below+from != whole {
		t.Fatalf("elements: below %d + from %d != whole %d", below, from, whole)
	}
	if want := 1 + 4; below != want { // the root, the base column with its two boxes, the scrim
		t.Errorf("DrawBelow drew %d elements, want %d", below, want)
	}
	if want := 3; from != want { // the panel, its child, the float
		t.Errorf("DrawFrom drew %d elements, want %d", from, want)
	}
	if ctx.HoverOff {
		t.Error("HoverOff must not stay set after drawing")
	}
}

func TestTheTipComesOnlyFromThePartThatDrawsTheHoveredElement(t *testing.T) {
	ctx := testCtx(t)
	root := layeredRoot()
	Layout(ctx, root, 200, 100)
	dst := ebiten.NewImage(200, 100)
	ctx.In = core.Input{X: 30, Y: 30} // over the panel's child
	DrawBelow(ctx, root, 2, dst)
	if ctx.TipText != "" {
		t.Errorf("the base part set the tip %q under a modal", ctx.TipText)
	}
	DrawFrom(ctx, root, 2, dst)
	if ctx.TipText != "top" {
		t.Errorf("TipText = %q, want the panel's tip", ctx.TipText)
	}
}
