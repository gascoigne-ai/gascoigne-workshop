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

// pressThumb is a press at `at` in a scrollbar track for content shown
// through a view scrolled to pos: it either grabs the thumb (grabbed, with
// the offset inside it) or pages toward the press (the new pos).
func pressThumb(track, content, view, pos, at int) (grabbed bool, off, newPos int) {
	thumb, length, ok := Thumb(track, content, view, pos)
	switch {
	case !ok:
		return false, 0, pos
	case at >= thumb && at < thumb+length:
		return true, at - thumb, pos
	case at < thumb:
		return false, 0, clampScroll(pos-view, content, view)
	}
	return false, 0, clampScroll(pos+view, content, view)
}
