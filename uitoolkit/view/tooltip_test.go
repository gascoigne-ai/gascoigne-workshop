package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestNoTooltipForElementsUnderAModalOverlay(t *testing.T) {
	under := box(W(100), Tooltip("under"))
	overlay := box(W(10), H(10), FloatAt(150, 0), Modal(), OnDismiss(func() {}))
	root := Stack(Row(under), overlay)
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 30}
	drawn(t, c, root, 200, 50)
	if c.TipText != "" {
		t.Fatalf("tooltip %q under a modal", c.TipText)
	}
}
