package overlays

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/menus"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Keys handles the keys of an open menu (Up, Down, Enter) or dialog (Enter,
// for one without a text field, which has its own), of the destination
// picker or of the file browser, when no question is asked over them. The
// first that is open claims the frame's keys, in that order.
func (c *Component) Keys(ctx *w.Ctx, in w.Input) {
	enter := in.Enter || in.Pressed(ebiten.KeyEnter)
	switch {
	case menus.Shown(ctx):
		menus.Keys(ctx, in)
	case c.dialog != nil && c.dialog.Input == nil && enter:
		c.Default()
	case c.dialog == nil && c.picker != nil:
		c.pickerKeys(ctx, in)
	case c.Browser.Shown():
		c.Browser.Keys(ctx, in)
	}
}

// pickerKeys hands the picker the frame's keys and takes them from the
// input, so the field under it does not act on them too.
func (c *Component) pickerKeys(ctx *w.Ctx, in w.Input) {
	in.Enter = in.Enter || in.Pressed(ebiten.KeyEnter)
	ctx.In.Consume(ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyArrowDown)
	c.picker.Keys(in, c.pickerHost)
}
