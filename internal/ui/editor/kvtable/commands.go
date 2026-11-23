package kvtable

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
)

// Commands are the shortcut actions the tables offer, in display order.
func (c *Component) Commands() []keymap.Command {
	return []keymap.Command{
		{ID: "duplicate-row", Name: "Duplicate row", Default: keymap.Chord{Key: int(ebiten.KeyD), Ctrl: true}, Run: c.DuplicateRow},
		{ID: "move-row-up", Name: "Move row up", Default: keymap.Chord{Key: int(ebiten.KeyArrowUp), Alt: true}, Run: func() { c.MoveRow(-1) }},
		{ID: "move-row-down", Name: "Move row down", Default: keymap.Chord{Key: int(ebiten.KeyArrowDown), Alt: true}, Run: func() { c.MoveRow(1) }},
	}
}
