package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// HOf, WOf and SeparatorOf are resolved at every Layout from the context's
// Sizes, so a row built once follows a change of text size.
func TestSizeRolesFollowTheContextsSizes(t *testing.T) {
	boxed := box(HOf(core.RowOf), WOf(core.IconBtnOf), SeparatorOf(func(sizes core.Sizes) int { return sizes.Line }))
	root := Col(boxed)
	ctx := testCtx(t)

	for _, sizes := range []core.Sizes{core.DefaultSizes(), core.SizesFor(20, 16)} {
		ctx.Sizes = sizes
		Layout(ctx, root, 300, 300)
		w, h := boxed.FixedSize()
		if h != sizes.Row || w != sizes.IconBtnW {
			t.Errorf("line %d: fixed size = %dx%d, want %dx%d", sizes.Line, w, h, sizes.IconBtnW, sizes.Row)
		}
		if boxed.borderInset != sizes.Line {
			t.Errorf("line %d: separator inset = %d, want %d", sizes.Line, boxed.borderInset, sizes.Line)
		}
	}
}

// A fixed size given after a role replaces it, as a later H replaces an H:
// a caller that overrides a ready-made row keeps its size at every text size.
func TestFixedSizeAfterARoleReplacesIt(t *testing.T) {
	boxed := box(HOf(core.RowOf), WOf(core.IconBtnOf), H(50), W(60))
	ctx := testCtx(t)
	ctx.Sizes = core.SizesFor(20, 16)
	Layout(ctx, Col(boxed), 300, 300)
	if w, h := boxed.FixedSize(); w != 60 || h != 50 {
		t.Errorf("fixed size = %dx%d, want 60x50", w, h)
	}
}
