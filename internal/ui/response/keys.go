package response

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// Typing reports whether the find field of the current tab has the keyboard.
// A hidden tab's field may keep a stale Focused, so only the current one
// counts, and asking makes no state.
func (c *Component) Typing() bool {
	current := c.deps.Tabs.Current()
	if current == nil {
		return false
	}
	state, ok := c.states[current]
	return ok && state.find.Focused
}

// Keys reports whether the frame's keys are the find field's, which it
// handles itself: Undo and Redo, and a frame of keys that type text (a
// binding with no Ctrl or Alt, no function key, not Delete, which waits until
// no text field has the keyboard, engine.Binding.TypesText). It claims nothing
// while the field is not the current tab's focused one. Escape is never
// claimed: it closes the find bar. Ctrl+F and Ctrl+Enter are the
// application's. The application offers the keys when no overlay is open.
func (c *Component) Keys(in core.Input) bool {
	if !c.Typing() {
		return false
	}
	return in.Undo || in.Redo || typesOnly(in)
}

// typesOnly reports whether every key of the frame is one a text field is
// typed with, under the modifiers held. It is the third copy, next to
// urlbar's and body's: the toolkit would own it as engine.TypesOnly, but a
// change of uitoolkit needs the user's approval, which the three components
// wait for. Revisit when it is given.
func typesOnly(in core.Input) bool {
	if in.Escape || len(in.Keys) == 0 {
		return false
	}
	for _, key := range in.Keys {
		binding := engine.Binding{Key: key, Ctrl: in.Ctrl, Shift: in.Shift, Alt: in.Alt, Bound: true}
		if !binding.TypesText() {
			return false
		}
	}
	return true
}
