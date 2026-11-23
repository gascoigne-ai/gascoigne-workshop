package body

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// Typing reports whether the text area or the path field of the current tab
// has the keyboard. The focus of a hidden tab's widget does not count.
func (c *Component) Typing() bool {
	current := c.deps.Tabs.Current()
	if current == nil {
		return false
	}
	state := c.state(current)
	return state.area.Focused || state.file.Focused
}

// Keys reports whether the frame's keys are the body's, which it claims
// while Typing: Undo and Redo, which the table's Undo must not take, and
// keys that only type text, which no shortcut without Ctrl or Alt may take.
// Escape and Ctrl shortcuts stay with the window.
func (c *Component) Keys(in core.Input) bool {
	if !c.Typing() {
		return false
	}
	return in.Undo || in.Redo || typesOnly(in)
}

// typesOnly reports whether every key of the frame is one a text field is
// typed with, under the modifiers held. It is a copy of the one in the URL
// bar, and ui/response has a third (siblings may not import each other); the
// toolkit would own it as engine.TypesOnly, which waits for the user's
// approval of a change to uitoolkit.
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
