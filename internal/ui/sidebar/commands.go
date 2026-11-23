package sidebar

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
)

// Commands are the shortcut actions the sidebar offers, in display order.
func (c *Component) Commands() []keymap.Command {
	return []keymap.Command{
		{ID: "open-workspace", Name: "Open collection folder", Default: keymap.Chord{Key: int(ebiten.KeyO), Ctrl: true}, Run: func() { c.deps.Workspace.OpenFolder() }},
		{ID: "close-collection", Name: "Close collection", Default: keymap.Chord{Key: int(ebiten.KeyW), Ctrl: true, Shift: true}, Run: c.closeFocused},
	}
}

// closeFocused closes the collection the chip menu acts on, if one is open.
func (c *Component) closeFocused() {
	if col := c.deps.Collections.Focused(); col != nil {
		c.deps.Collection.Close(col)
	}
}
