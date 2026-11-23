package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// treeDragging reports whether a tree row is held and over another row.
func (a *Component) treeDragging() bool {
	return a.side.Tree().Drag().Phase() == logic.Dragging
}

// grabbing reports whether the pointer holds something that moves: a
// table row, a tab or a tree row.
func (a *Component) grabbing() bool {
	if !a.ctx.In.Held && !a.ctx.In.MiddleHeld {
		return false
	}
	return a.tabs.Drag() >= 0 || a.editor.Tables.Dragging() || a.treeDragging()
}

// showGrab turns the cursor into the move shape while something is held,
// unless a divider or column handle already asked for a resize shape.
func (a *Component) showGrab() {
	if a.ctx.Cursor == core.CursorDefault && a.grabbing() {
		a.ctx.Cursor = core.CursorMove
	}
}
