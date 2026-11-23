package response

// Tick drops the state of every closed tab, with its find text and its
// picture, which would otherwise stay for as long as the program runs. It
// reports nothing changed: a closed tab shows nothing.
func (c *Component) Tick() bool {
	for tab := range c.states {
		if !c.deps.Tabs.IsOpen(tab) {
			delete(c.states, tab)
		}
	}
	return false
}
