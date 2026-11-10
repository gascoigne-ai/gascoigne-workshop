package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestGrabFollowsTheButtonAndEndsOnRelease(t *testing.T) {
	var g grab
	if g.held(core.Input{Held: true}) {
		t.Fatal("no drag before begin")
	}
	g.begin(7)
	if !g.held(core.Input{Held: true}) || g.off != 7 {
		t.Fatalf("drag lost while held: %+v", g)
	}
	if g.held(core.Input{}) || g.on {
		t.Fatal("releasing the button ends the drag, and that frame is not a move")
	}
}

func TestPressThumbGrabsTheThumbOrPages(t *testing.T) {
	// content 1000 in a view of 100 on a 100px track: the thumb is 10px.
	grabbed, off, pos := pressThumb(100, 1000, 100, 0, 5)
	if !grabbed || off != 5 || pos != 0 {
		t.Fatalf("on the thumb: %v %d %d", grabbed, off, pos)
	}
	grabbed, _, pos = pressThumb(100, 1000, 100, 0, 60)
	if grabbed || pos != 100 {
		t.Fatalf("below the thumb pages down: %v %d", grabbed, pos)
	}
	grabbed, _, pos = pressThumb(100, 1000, 100, 300, 2)
	if grabbed || pos != 200 {
		t.Fatalf("above the thumb pages up: %v %d", grabbed, pos)
	}
	if grabbed, _, pos = pressThumb(100, 50, 100, 0, 5); grabbed || pos != 0 {
		t.Fatal("content that fits has nothing to grab")
	}
}
