package mcp

// Tick drops the state of every closed tab, its form and its answer with
// it. It changes nothing on screen, so it reports false.
func (c *Component) Tick() bool {
	for tab := range c.states {
		if !c.deps.Tabs.IsOpen(tab) {
			delete(c.states, tab)
		}
	}
	return false
}
