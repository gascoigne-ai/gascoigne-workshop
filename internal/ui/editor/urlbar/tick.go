package urlbar

// Tick drops the field of every closed tab, its text lost with it, and
// reports whether it changed what the bar shows (never: a closed tab is not
// drawn).
func (c *Component) Tick() bool {
	for tab := range c.fields {
		if !c.deps.Tabs.IsOpen(tab) {
			delete(c.fields, tab)
		}
	}
	return false
}
