package kvtable

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// BeginFrame ends a row drag when the button was released. It runs before
// the view is built, so the build already sees no drag.
func (c *Component) BeginFrame(in core.Input) {
	if !in.Held && !in.MiddleHeld {
		c.dragged = nil
	}
}

// Tick runs the per-frame work after the view was built and reports whether
// it changed what the view shows. The order matters: (1) an edit of a closed
// tab is dropped, its text lost; (2) an edit whose field lost focus (a click
// elsewhere) is committed; (3) bulk text whose table is no longer visible is
// applied to its rows, a closed tab's too; (4) the selection and drag of a
// closed tab are dropped.
func (c *Component) Tick() bool {
	changed := false
	c.liveEdit()
	if c.edit != nil && !c.edit.field.Focused {
		c.Commit()
		changed = true
	}
	if c.bulk != nil {
		if table, ok := c.deps.Tabs.Visible(); !ok || table != c.bulk.Table {
			c.closeBulk()
			changed = true
		}
	}
	if c.selected != nil && !c.deps.Tabs.IsOpen(c.selected.Tab) {
		c.selected = nil
	}
	if c.dragged != nil && !c.deps.Tabs.IsOpen(c.dragged.Tab) {
		c.dragged = nil
	}
	return changed
}
