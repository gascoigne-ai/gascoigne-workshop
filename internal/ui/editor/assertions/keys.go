package assertions

// Typing reports whether the tests area or the timeout field of the current
// tab has the keyboard, so the window holds back shortcut keys. A tab that
// was never shown has no fields and counts as not typing.
func (c *Component) Typing() bool {
	current := c.deps.Tabs.Current()
	if current == nil {
		return false
	}
	state, ok := c.states[current]
	if !ok {
		return false
	}
	return state.tests.Focused || state.timeout.Focused
}
