package widget

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
