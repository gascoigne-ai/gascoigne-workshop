package widget

import (
	"strconv"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

var toastStart = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

func TestAnInfoToastFadesAnErrorStays(t *testing.T) {
	var stack ToastStack
	stack.Show("saved", false, toastStart)
	stack.Show("failed", true, toastStart)
	if stack.Prune(toastStart.Add(ToastInfoFor - time.Millisecond)) {
		t.Fatal("nothing has expired yet")
	}
	if !stack.Prune(toastStart.Add(ToastInfoFor)) || len(stack) != 1 || !stack[0].IsErr {
		t.Fatalf("after the time only the error is left: %+v", stack)
	}
	if stack.Prune(toastStart.Add(time.Hour)) {
		t.Fatal("an error toast stays until dismissed")
	}
}

func TestTheSameErrorIsShownOnce(t *testing.T) {
	var stack ToastStack
	if !stack.Show("boom", true, toastStart) {
		t.Fatal("the first one is shown")
	}
	if stack.Show("boom", true, toastStart) || len(stack) != 1 {
		t.Fatalf("a repeated failure is one message: %+v", stack)
	}
	if !stack.Show("boom", false, toastStart) {
		t.Fatal("an info toast with the same text is a different message")
	}
}

func TestOlderToastsAreDroppedBeyondTheMaximum(t *testing.T) {
	var stack ToastStack
	for i := range ToastMax + 3 {
		stack.Show(strconv.Itoa(i), true, toastStart)
	}
	if len(stack) != ToastMax || stack[0].Text != "3" || stack[ToastMax-1].Text != strconv.Itoa(ToastMax+2) {
		t.Fatalf("stack = %+v", stack)
	}
}

func TestDismissRemovesOneToast(t *testing.T) {
	var stack ToastStack
	stack.Show("a", true, toastStart)
	stack.Show("b", true, toastStart)
	stack.Dismiss(0)
	if len(stack) != 1 || stack[0].Text != "b" {
		t.Fatalf("stack = %+v", stack)
	}
}

// The toasts stack up from the bottom-right corner of the window, above the
// bottom inset; a click dismisses the toast under it.
func TestToastsStackAboveTheInsetAndAClickDismissesOne(t *testing.T) {
	c := testCtx(t)
	var stack ToastStack
	stack.Show("saved", false, toastStart)
	stack.Show("boom", true, toastStart)
	view := Toasts(&stack, 800, 600, 28)
	root := Stack(view)
	Layout(c, root, 800, 600)

	first, second := view.Children()[0].Rect(), view.Children()[1].Rect()
	if view.Rect().X+view.Rect().W != 800-core.Gutter {
		t.Errorf("toasts end at x=%d, want the gutter %d from the right edge", view.Rect().X+view.Rect().W, core.Gutter)
	}
	if second.Y+second.H > 600-28 || first.Y >= second.Y {
		t.Errorf("rows at %d and %d, want oldest on top and the stack above the inset", first.Y, second.Y)
	}
	click(c, first.X+5, first.Y+5)
	Update(c, root)
	if len(stack) != 1 || stack[0].Text != "boom" {
		t.Fatalf("clicking the first toast left %+v", stack)
	}
}

func TestNoToastsNoView(t *testing.T) {
	var stack ToastStack
	if Toasts(&stack, 800, 600, 28) != nil {
		t.Fatal("an empty stack draws nothing")
	}
}
