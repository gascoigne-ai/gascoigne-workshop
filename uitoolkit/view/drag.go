package view

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// grab is a drag in progress that started on a press: the split divider,
// a scrollbar thumb, a column edge. off is where inside the grabbed thing
// the pointer was when it began, so it does not jump to the pointer.
type grab struct {
	on  bool
	off int
}

// begin starts a drag that grabbed the thing at off.
func (g *grab) begin(off int) { g.on, g.off = true, off }

// held reports whether the drag goes on this frame. Releasing the button
// ends it, and the frame of the release is not a move.
func (g *grab) held(in core.Input) bool {
	if g.on && !in.Held {
		g.on = false
	}
	return g.on
}
