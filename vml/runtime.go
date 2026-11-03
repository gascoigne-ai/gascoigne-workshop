package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Layout places the tree in a w by h window.
func Layout(c *widget.Ctx, root Node, w, h int) {
	root.place(c, widget.Rect{W: w, H: h})
}

// Update delivers this frame's input (c.In) to the tree, top-most
// element first. It must follow Layout on the same tree.
func Update(c *widget.Ctx, root Node) { root.update(c) }

// Draw paints the tree. It must follow Layout on the same tree.
func Draw(c *widget.Ctx, root Node, dst *ebiten.Image) { root.draw(c, dst) }

// update reports whether a modal element swallowed the input.
func (e *El) update(c *widget.Ctx) bool {
	if e.hidden {
		return false
	}
	for i := len(e.kids) - 1; i >= 0; i-- {
		if e.kids[i].update(c) {
			return true
		}
	}
	e.b.update(e, c)
	if e.onClick != nil && c.In.Hit(e.rect) {
		e.onClick()
	}
	if e.modal && c.In.Click {
		if !e.rect.Contains(c.In.X, c.In.Y) && e.onDismiss != nil {
			e.onDismiss()
		}
		return true
	}
	return false
}

func (e *El) draw(c *widget.Ctx, dst *ebiten.Image) {
	if e.hidden {
		return
	}
	r := e.rect
	if e.shadow {
		c.Fill(dst, widget.Rect{X: r.X + 4, Y: r.Y + 4, W: r.W, H: r.H}, c.Col(widget.Shadow))
	}
	over := c.In.Over(r)
	if e.bg != "" {
		clr := c.Col(e.bg)
		if e.lighten && over {
			clr = widget.Lighten(clr)
		}
		c.Fill(dst, r, clr)
	}
	if e.hoverBg != "" && over {
		c.Fill(dst, r, c.Col(e.hoverBg))
	}
	if e.border != "" {
		c.Outline(dst, r, c.Col(e.border))
	}
	if e.borderTop != "" {
		c.HLine(dst, r.X, r.Y, r.W, c.Col(e.borderTop))
	}
	if e.borderBottom != "" {
		c.HLine(dst, r.X, r.Y+r.H-1, r.W, c.Col(e.borderBottom))
	}

	e.b.draw(e, c, dst)
	kidsDst := dst
	if e.clip {
		kidsDst = c.Clip(dst, r)
	}
	for _, k := range e.kids {
		k.draw(c, kidsDst)
	}

	if e.underline != "" {
		c.Fill(dst, widget.Rect{X: r.X + e.underInset, Y: r.Y + r.H - 2, W: r.W - 2*e.underInset, H: 2}, c.Col(e.underline))
	}
}
