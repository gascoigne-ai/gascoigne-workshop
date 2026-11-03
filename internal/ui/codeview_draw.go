package ui

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

const foldColW = 18

// codeGeometry is the pixel layout of a CodeView in r.
type codeGeometry struct {
	lineH, charW, numW, textX int
}

func (c *Ctx) codeGeometry(r Rect, v *CodeView) codeGeometry {
	g := codeGeometry{lineH: c.lineH(c.F.Mono) + 4, charW: c.width(c.F.Mono, "0")}
	g.numW = g.charW * len(strconv.Itoa(max(len(v.Lines), 1)))
	g.textX = r.X + c.Th.Pad + g.numW + foldColW + 8
	return g
}

// updateCodeView handles wheel scrolling and fold-arrow clicks.
func (c *Ctx) updateCodeView(r Rect, v *CodeView) {
	if !c.In.Over(r) {
		return
	}
	g := c.codeGeometry(r, v)
	vis := v.Visible()
	if c.In.Wheel != 0 {
		v.Scroll(int(-c.In.Wheel*float64(g.lineH)*3), len(vis)*g.lineH, r.H)
	}
	foldCol := Rect{r.X + c.Th.Pad + g.numW + 4, r.Y, foldColW, r.H}
	if !c.In.Hit(foldCol) {
		return
	}
	if row := (c.In.Y - r.Y + v.ScrollY) / g.lineH; row >= 0 && row < len(vis) {
		v.ToggleFold(vis[row])
	}
}

func (c *Ctx) drawCodeView(dst *ebiten.Image, r Rect, v *CodeView) {
	th := c.Th
	g := c.codeGeometry(r, v)
	vis := v.Visible()
	v.Scroll(0, len(vis)*g.lineH, r.H)
	dst = c.clip(dst, r)

	for row := v.ScrollY / g.lineH; row < len(vis); row++ {
		y := r.Y + row*g.lineH - v.ScrollY
		if y >= r.Y+r.H {
			break
		}
		idx := vis[row]
		line := v.Lines[idx]
		cell := Rect{r.X, y, r.W, g.lineH}

		num := strconv.Itoa(idx + 1)
		c.textMid(dst, c.F.Mono, num, r.X+th.Pad+g.numW-len(num)*g.charW, cell, th.Muted)
		if line.Foldable {
			cx := float32(r.X + th.Pad + g.numW + 4 + foldColW/2)
			c.triangle(dst, cx, float32(y+g.lineH/2), 4, !v.Folded[idx], th.Muted)
		}

		x := g.textX
		for _, sp := range line.Spans {
			c.textMid(dst, c.F.Mono, sp.Text, x, cell, th.Syntax[sp.Kind])
			x += len([]rune(sp.Text)) * g.charW
		}
		if v.Folded[idx] {
			c.textMid(dst, c.F.Mono, " ... }", x, cell, th.Muted)
		}
	}
}
