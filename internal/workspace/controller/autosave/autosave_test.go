package autosave

import (
	"testing"
	"time"
)

var start = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

func TestDueOnlyAfterTheKeyStoppedChangingForTheWait(t *testing.T) {
	debounce := New(time.Second, "a")
	if debounce.Due("a", start) {
		t.Fatal("nothing changed since the last save")
	}
	if debounce.Due("b", start) {
		t.Fatal("a change starts the wait, it is not due yet")
	}
	if debounce.Due("b", start.Add(999*time.Millisecond)) {
		t.Fatal("the wait has not passed")
	}
	if !debounce.Due("b", start.Add(time.Second)) {
		t.Fatal("unchanged for the wait: due")
	}
}

func TestAChangeRestartsTheWait(t *testing.T) {
	debounce := New(time.Second, "a")
	debounce.Due("b", start)
	debounce.Due("c", start.Add(900*time.Millisecond))
	if debounce.Due("c", start.Add(1500*time.Millisecond)) {
		t.Fatal("the wait counts from the last change")
	}
	if !debounce.Due("c", start.Add(1900*time.Millisecond)) {
		t.Fatal("the wait has passed since the last change")
	}
}

func TestSavedEndsTheNeedAndAChangeBackToItIsNotDue(t *testing.T) {
	debounce := New(time.Second, "a")
	debounce.Due("b", start)
	debounce.Saved("b")
	if debounce.Due("b", start.Add(time.Hour)) || debounce.Pending("b") {
		t.Fatal("what was saved is neither due nor pending")
	}
	if !debounce.Pending("c") {
		t.Fatal("a different key is pending")
	}
	debounce.Due("c", start.Add(2*time.Second))
	if debounce.Due("b", start.Add(time.Hour)) {
		t.Fatal("going back to the saved content needs no save")
	}
}
