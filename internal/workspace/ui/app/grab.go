package ui

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// treeDragging reports whether a tree row is held and over another row.
func (a *App) treeDragging() bool {
	return a.dragNode != nil && a.dropTarget != nil && a.dropTarget != a.dragNode
}

// grabbing reports whether the pointer holds something that moves: a
// table row, a tab or a tree row.
func (a *App) grabbing() bool {
	if !a.ctx.In.Held && !a.ctx.In.MiddleHeld {
		return false
	}
	return a.dragTab >= 0 || a.dragRow != nil || a.treeDragging()
}

// showGrab turns the cursor into the move shape while something is held,
// unless a divider or column handle already asked for a resize shape.
func (a *App) showGrab() {
	if a.ctx.Cursor == core.CursorDefault && a.grabbing() {
		a.ctx.Cursor = core.CursorMove
	}
}
