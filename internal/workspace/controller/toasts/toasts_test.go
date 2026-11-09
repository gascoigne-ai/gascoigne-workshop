package toasts

import (
	"strconv"
	"testing"
	"time"
)

var start = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

func TestAnInfoToastFadesAnErrorStays(t *testing.T) {
	var stack Stack
	stack.Show("saved", false, start)
	stack.Show("failed", true, start)
	if stack.Prune(start.Add(InfoFor - time.Millisecond)) {
		t.Fatal("nothing has expired yet")
	}
	if !stack.Prune(start.Add(InfoFor)) || len(stack) != 1 || !stack[0].IsErr {
		t.Fatalf("after the time only the error is left: %+v", stack)
	}
	if stack.Prune(start.Add(time.Hour)) {
		t.Fatal("an error toast stays until dismissed")
	}
}

func TestTheSameErrorIsShownOnce(t *testing.T) {
	var stack Stack
	if !stack.Show("boom", true, start) {
		t.Fatal("the first one is shown")
	}
	if stack.Show("boom", true, start) || len(stack) != 1 {
		t.Fatalf("a repeated failure is one message: %+v", stack)
	}
	if !stack.Show("boom", false, start) {
		t.Fatal("an info toast with the same text is a different message")
	}
}

func TestOlderToastsAreDroppedBeyondTheMaximum(t *testing.T) {
	var stack Stack
	for i := range Max + 3 {
		stack.Show(strconv.Itoa(i), true, start)
	}
	if len(stack) != Max || stack[0].Text != "3" || stack[Max-1].Text != strconv.Itoa(Max+2) {
		t.Fatalf("stack = %+v", stack)
	}
}

func TestDismissRemovesOneToast(t *testing.T) {
	var stack Stack
	stack.Show("a", true, start)
	stack.Show("b", true, start)
	stack.Dismiss(0)
	if len(stack) != 1 || stack[0].Text != "b" {
		t.Fatalf("stack = %+v", stack)
	}
}
