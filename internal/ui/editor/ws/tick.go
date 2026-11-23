package ws

// Tick drops the state of every closed tab, its log and its draft with it.
// It changes nothing on screen, so it returns false.
func (c *Component) Tick() bool {
	for tab := range c.states {
		if !c.deps.Tabs.IsOpen(tab) {
			delete(c.states, tab)
		}
	}
	return false
}
