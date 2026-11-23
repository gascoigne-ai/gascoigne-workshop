package auth

// Tick drops the state of every closed tab, its unsaved draft lost with it,
// and reports whether the view changed, which it never does: a closed tab is
// not drawn.
func (c *Component) Tick() bool {
	for tab := range c.states {
		if !c.deps.Tabs.IsOpen(tab) {
			delete(c.states, tab)
		}
	}
	return false
}
