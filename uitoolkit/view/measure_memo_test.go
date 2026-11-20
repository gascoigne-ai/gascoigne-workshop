package view_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// counted is a leaf whose Measure counts its calls.
type counted struct {
	view.Base
	width, measures int
}

func (c *counted) Measure(*view.El, *core.Ctx, int) (int, int) {
	c.measures++
	return c.width, 20
}
func (c *counted) Draw(*view.El, *core.Ctx, *ebiten.Image) {}

func nested(leaf view.Node) *view.El {
	return view.Col(view.Row(view.Col(view.Row(leaf), view.Row(view.Spacer()))), view.Row(view.Spacer()))
}

func TestLayoutMeasuresATextOncePerPass(t *testing.T) {
	ctx := newCtx(t)
	ctx.Stats = &core.DrawStats{}
	root := nested(view.Text("a line of text"))
	view.Layout(ctx, root, 300, 200)
	if ctx.Stats.Widths != 1 {
		t.Errorf("a text was measured %d times in one Layout, want 1", ctx.Stats.Widths)
	}
}

// A behavior is measured once per width: a parent that measures a child
// and then lays it out at the same width asks once.
func TestLayoutMeasuresABehaviorOncePerWidth(t *testing.T) {
	ctx := newCtx(t)
	leaf := &counted{width: 30}
	root := view.Col(view.AlignItems(view.AlignStretch), view.Row(view.AlignItems(view.AlignStretch), view.Custom(leaf)))
	view.Layout(ctx, root, 300, 200)
	if leaf.measures != 1 {
		t.Errorf("leaf measured %d times in one Layout, want 1", leaf.measures)
	}
}

// The memo is valid for one Layout: a text that changed between two passes
// must be measured again.
func TestMemoDoesNotSurviveALayout(t *testing.T) {
	ctx := newCtx(t)
	leaf := &counted{width: 30}
	node := view.Custom(leaf)
	root := view.Row(view.AlignItems(view.AlignStart), node)
	view.Layout(ctx, root, 300, 200)
	leaf.width = 80
	view.Layout(ctx, root, 300, 200)
	if got := node.Rect().W; got != 80 {
		t.Errorf("width after the second Layout = %d, want 80", got)
	}
	if leaf.measures != 2 {
		t.Errorf("measures = %d, want 1 per pass", leaf.measures)
	}
}
