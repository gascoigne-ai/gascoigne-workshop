package ws

// Typing reports whether the composer of the current tab has the keyboard.
// It makes no state: a tab that never showed its pane has no composer to
// type in. The pane claims no key itself (Enter reaches the field through
// the field's own handler), so there is no Keys.
func (c *Component) Typing() bool {
	current := c.deps.Tabs.Current()
	if current == nil {
		return false
	}
	made, ok := c.states[current]
	if !ok {
		return false
	}
	return made.composer.Focused
}
