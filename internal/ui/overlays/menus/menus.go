// Package menus is the context menu of the window: the one menu that is
// open, the keys it takes and the layer it is drawn in.
//
// There is no Component and no Deps. The open menu is state of the toolkit's
// context (core.Ctx.Menu), shared with every widget that opens a menu from
// its own click handler, so a component here would hold a second copy of a
// fact the context already owns; the functions take the context instead.
// Revisit when the menu needs state the context cannot hold.
package menus

import (
	"github.com/hajimehoshi/ebiten/v2"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Open shows a context menu of items at the pointer position x, y. Items
// without an icon get the one their label maps to.
func Open(ctx *w.Ctx, posX, posY int, items []MenuItem) {
	OpenMenu(ctx, posX, posY, withIcons(items))
}

// Close hides the menu.
func Close(ctx *w.Ctx) { CloseMenu(ctx) }

// Shown reports whether a menu is open.
func Shown(ctx *w.Ctx) bool { return MenuOpen(ctx) }

// Keys handles the keys of an open menu: Up, Down and Enter.
func Keys(ctx *w.Ctx, in w.Input) { PopupMenuKeys(ctx, in) }

// Swallow removes the keys a menu used from this frame's input, so the text
// widget that still has the focus under it does not act on them too.
func Swallow(ctx *w.Ctx) {
	ctx.In.Consume(ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyEscape, ebiten.KeyTab)
}

// View is the menu's layer of the window, nothing while none is open.
func View(ctx *w.Ctx) Node { return PopupMenuLayer(ctx) }
