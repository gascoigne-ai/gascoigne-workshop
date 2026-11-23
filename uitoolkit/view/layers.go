package view

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// A frame that holds a modal overlay splits the root's children in two: the
// ones before the first modal (the base, which cannot react to the pointer
// while the overlay is open) and the rest. An app that keeps the base in
// an image of its own draws it with DrawBelow once and the rest with
// DrawFrom every frame; DrawBelow then DrawFrom on one image paint what
// Draw paints, element for element.

// ModalLayer is the index of the first visible direct child of root that is
// a modal overlay, or -1 when there is none. It is also -1 when the split
// would change the paint order: a modal deeper in the tree, a root that
// draws after its children (a border, the style border of its control, an
// OverDrawer behavior, a tab edge) or a root that is a modal itself. The
// menus count too: a context menu alone gives the layer under it. The theme
// of ctx may give the root's control a border.
func ModalLayer(ctx *core.Ctx, root Node) int {
	if root.hidden || root.modal || drawsAfterChildren(ctx, root) {
		return -1
	}
	for index, kid := range root.kids {
		if kid.modal && !kid.hidden {
			return index
		}
	}
	return -1
}

// drawsAfterChildren reports whether drawAfter paints anything for e; it
// must list what drawAfter reads.
func drawsAfterChildren(ctx *core.Ctx, e *El) bool {
	_, over := e.b.(OverDrawer)
	return over || e.border != "" || e.edgeTok != "" || ctx.Th.Style(e.control).Border != ""
}

// DrawBelow paints the root's own part and its children before layer, as
// Draw does, with hover feedback off: the part under a modal overlay never
// reacts to the pointer. layer comes from ModalLayer.
func DrawBelow(ctx *core.Ctx, root Node, layer int, dst *ebiten.Image) {
	ctx.HoverOff = true
	ctx.TipText = ""
	root.drawSelf(ctx, dst)
	root.drawKids(ctx, root.kidsDst(ctx, dst), root.kids[:layer])
	ctx.HoverOff = false
}

// DrawFrom paints the root's children from layer on, the overlays and what
// lies over them, into the clip the root gives them. Hover feedback is off
// for all but the overlay's own elements, as in Draw.
func DrawFrom(ctx *core.Ctx, root Node, layer int, dst *ebiten.Image) {
	ctx.HoverOff = true
	ctx.TipText = ""
	root.drawKids(ctx, root.kidsDst(ctx, dst), root.kids[layer:])
	ctx.HoverOff = false
}
