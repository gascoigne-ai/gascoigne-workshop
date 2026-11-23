package assertions

// Tick drops the state of every closed tab, its unsaved caret and undo
// history with it, and reports that nothing on screen changed.
func (c *Component) Tick() bool {
	for tab := range c.states {
		if !c.deps.Tabs.IsOpen(tab) {
			delete(c.states, tab)
		}
	}
	return false
}
