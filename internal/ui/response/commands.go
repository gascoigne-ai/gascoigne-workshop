package response

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
)

// Commands are the shortcut actions the response pane offers.
func (c *Component) Commands() []keymap.Command {
	return []keymap.Command{
		{ID: "find", Name: "Find in response", Default: keymap.Chord{Key: int(ebiten.KeyF), Ctrl: true}, Run: c.OpenFind},
	}
}
