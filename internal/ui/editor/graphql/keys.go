package graphql

// Typing reports whether the variables area of the current tab has the
// keyboard. The focus of a hidden tab's area does not count. The query area
// is the body component's, which reports it.
func (c *Component) Typing() bool {
	current := c.deps.Tabs.Current()
	if current == nil {
		return false
	}
	state, ok := c.states[current]
	return ok && state.vars.Focused
}
