package kvtable

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// Keys claims Undo and Redo for the table that is showing, and reports
// whether it did. Inside a cell or the bulk text, or while any other text
// field has the keyboard (textFocused), Ctrl+Z belongs to that text. The
// application offers the keys only when no overlay is open.
func (c *Component) Keys(in core.Input, textFocused bool) bool {
	if !in.Undo && !in.Redo || textFocused || c.Typing() {
		return false
	}
	table, ok := c.deps.Tabs.Visible()
	if !ok {
		return false
	}
	if in.Undo {
		c.undo(table)
	} else {
		c.redo(table)
	}
	return true
}
