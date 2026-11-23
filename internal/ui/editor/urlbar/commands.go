package urlbar

// Send sends the request of the current tab, as the Send button does; on a
// socket tab it opens or closes the connection. The window's shortcut table
// keeps the id and the default key of the "send" action.
func (c *Component) Send() {
	if current := c.deps.Tabs.Current(); current != nil {
		c.deps.Sends.Send(current)
	}
}

// FocusURL puts the keyboard in the URL field of the current tab with its
// text selected. The context takes the keyboard from whatever had it.
func (c *Component) FocusURL() {
	current := c.deps.Tabs.Current()
	if current == nil {
		return
	}
	field := c.Field(current)
	field.Focused = true
	field.SelectAll()
}
