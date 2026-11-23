package auth

// Typing reports whether a field of the current tab has the keyboard, also
// one of an auth type the pane does not show now. The focus of a hidden
// tab's field does not count. It makes no state: a tab that never showed the
// pane has no field to focus.
func (c *Component) Typing() bool {
	current := c.deps.Tabs.Current()
	if current == nil {
		return false
	}
	state, ok := c.states[current]
	if !ok {
		return false
	}
	for _, field := range state.fields {
		if field.Focused {
			return true
		}
	}
	return false
}
