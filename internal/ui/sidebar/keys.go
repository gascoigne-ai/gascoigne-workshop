package sidebar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/hajimehoshi/ebiten/v2"
)

// Keys handles Up, Down, Enter, F2 and Delete while the tree has the
// keyboard and nothing is being typed; width is the sidebar's width, so a
// click beyond it takes the keyboard away. It reports whether it acted on a
// key. The key codes are read here so the logic package needs no Ebitengine.
func (c *Component) Keys(in core.Input, textFocused bool, width int) bool {
	if textFocused || in.Click && in.X > width {
		c.tree.Blur()
	}
	if !c.tree.Focused() {
		return false
	}
	rows := c.VisibleRows()
	if len(rows) == 0 {
		return false
	}
	acted := false
	for _, key := range in.Keys {
		switch key {
		case ebiten.KeyArrowDown:
			c.tree.Step(rows, 1)
		case ebiten.KeyArrowUp:
			c.tree.Step(rows, -1)
		case ebiten.KeyEnter:
			c.activateSelected(rows)
		case ebiten.KeyF2:
			if node := c.tree.Selected(); !node.None() {
				c.deps.Nodes.Rename(node)
			}
		case ebiten.KeyDelete:
			c.deps.Nodes.Delete(c.tree.Selected())
		default:
			continue
		}
		acted = true
	}
	return acted
}
