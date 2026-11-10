package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestAskingForFocusTakesItFromTheOwnerWithoutClearingByHand(t *testing.T) {
	a, b := NewTextField("a"), NewTextField("b")
	root := Col(Field(&a, H(30)), Field(&b, H(30)))
	c := testCtx(t)
	Layout(c, root, 200, 100)

	a.Focused = true
	Update(c, root)
	if c.FocusOwner != any(&a) || !a.Focused {
		t.Fatalf("owner %v, a focused %v", c.FocusOwner, a.Focused)
	}
	b.Focused = true // the app asks for b and does not clear a
	Update(c, root)
	if c.FocusOwner != any(&b) || a.Focused || !b.Focused {
		t.Fatalf("owner %v, a %v, b %v", c.FocusOwner, a.Focused, b.Focused)
	}
}

func TestAWidgetLeavingTheTreeLosesTheFocus(t *testing.T) {
	a, b := NewTextField("a"), NewTextField("b")
	c := testCtx(t)
	both := Col(Field(&a, H(30)), Field(&b, H(30)))
	Layout(c, both, 200, 100)
	b.Focused = true
	Update(c, both)

	only := Col(Field(&a, H(30))) // b's row is gone, as a closed find bar is
	Layout(c, only, 200, 100)
	c.In = core.Input{}
	Update(c, only)
	if b.Focused || c.FocusOwner != nil {
		t.Fatalf("b focused %v, owner %v", b.Focused, c.FocusOwner)
	}
}

func TestBlurringTheOwnerClearsTheOwner(t *testing.T) {
	a := NewTextField("a")
	root := Col(Field(&a, H(30)))
	c := testCtx(t)
	Layout(c, root, 200, 30)
	a.Focused = true
	Update(c, root)
	a.Focused = false
	Update(c, root)
	if c.FocusOwner != nil {
		t.Fatalf("owner %v", c.FocusOwner)
	}
}
