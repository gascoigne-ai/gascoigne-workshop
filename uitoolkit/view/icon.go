package view

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Icon is a vector icon by name, drawn in the element's color (Color, or
// muted by default) and in the icon's own hover color. It has the icon's
// natural size, centered; W and H scale it to fit. An unknown name panics
// and lists the names: it is a mistake in the program.
func Icon(name string, opts ...Option) *El {
	if !core.HasIcon(name) {
		panic(fmt.Sprintf("view: no icon %q; icons: %s", name, strings.Join(core.IconNames(), " ")))
	}
	e := newEl(iconB{}, with(nil, opts))
	e.iconName = name
	return e
}

// iconB draws a parsed SVG icon.
type iconB struct{ plain }

func (iconB) measure(e *El, _ *core.Ctx, _ int) (int, int) {
	w, h, _ := core.IconSize(e.iconName)
	return int(math.Ceil(w)), int(math.Ceil(h))
}

// iconToken is the color token an icon is drawn in: its color, and under
// the pointer the element's HoverColor, else the icon's own hover token.
func iconToken(e *El, hover core.Token, hovered bool) core.Token {
	if !hovered {
		return e.color(nil, core.Muted)
	}
	switch {
	case e.hoverColor != "":
		return e.hoverColor
	case hover != "":
		return hover
	}
	return e.color(nil, core.Muted)
}

func (iconB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	w, h, ok := core.IconSize(e.iconName)
	if !ok {
		return
	}
	r := e.rect
	k := 1.0
	switch {
	case e.w > 0 && e.h > 0:
		k = math.Min(float64(e.w)/w, float64(e.h)/h)
	case e.w > 0:
		k = float64(e.w) / w
	case e.h > 0:
		k = float64(e.h) / h
	}
	ox := float64(r.X) + (float64(r.W)-w*k)/2
	oy := float64(r.Y) + (float64(r.H)-h*k)/2
	cur := iconToken(e, core.IconHover(e.iconName), c.Hovered(r))
	c.DrawIcon(dst, e.iconName, ox, oy, k, cur)
}
