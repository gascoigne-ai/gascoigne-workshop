package tabstrip

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Commands are the shortcut actions the strip offers, in display order.
func (c *Component) Commands() []keymap.Command {
	return []keymap.Command{
		{ID: "new-request", Name: "New request", Default: keymap.Chord{Key: int(ebiten.KeyT), Ctrl: true}, Run: c.deps.Requests.NewRequest},
		{ID: "close-tab", Name: "Close tab", Default: keymap.Chord{Key: int(ebiten.KeyW), Ctrl: true}, Run: c.closeActive},
		{ID: "next-tab", Name: "Next tab", Default: keymap.Chord{Key: int(ebiten.KeyTab), Ctrl: true}, Run: func() { c.deps.Tabs.Update(tabs.Step{By: 1}) }},
		{ID: "prev-tab", Name: "Previous tab", Default: keymap.Chord{Key: int(ebiten.KeyTab), Ctrl: true, Shift: true}, Run: func() { c.deps.Tabs.Update(tabs.Step{By: -1}) }},
	}
}

// closeActive closes the shown tab; closing the last one leaves a blank request.
func (c *Component) closeActive() { c.deps.Tabs.Update(tabs.Close{Index: c.deps.Tabs.Active()}) }
