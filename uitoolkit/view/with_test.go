package view

import "testing"

func TestWithPutsCallerOptionsAfterDefaults(t *testing.T) {
	defaults := []Item{W(10), H(5)}
	items := With(defaults, []Option{W(20)})
	if len(items) != 3 {
		t.Fatalf("got %d items, want the 2 defaults and 1 option", len(items))
	}
	element := Row(items...)
	if element.w != 20 || element.h != 5 {
		t.Errorf("size = %dx%d, want the caller's W(20) over the default and the default H(5)", element.w, element.h)
	}
}

func TestWithNothingKeepsDefaults(t *testing.T) {
	if items := With([]Item{W(10)}, nil); len(items) != 1 {
		t.Errorf("got %d items, want the single default", len(items))
	}
}
