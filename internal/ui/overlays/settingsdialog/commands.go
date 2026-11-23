package settingsdialog

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
)

// Commands are the shortcut actions the dialog offers, in display order.
func (c *Component) Commands() []keymap.Command {
	return []keymap.Command{
		{ID: "settings", Name: "Settings", Default: keymap.Chord{Key: int(ebiten.KeyComma), Ctrl: true}, Run: c.Show},
	}
}
