package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	menuW   = 260
	chipGap = 6
)

// menuButton is the three-dot button at the right end of the header.
func menuButton(header Rect, pad int) Rect {
	return Rect{header.X + header.W - pad - header.H, header.Y, header.H, header.H}
}

// menuGeom is where the parts of the settings menu sit.
type menuGeom struct {
	Panel, Title, ThemeLabel, SizeLabel Rect
	Theme, Size                         []Rect // one chip per choice
}

// settingsMenu lays the menu out below the header, right-aligned.
func settingsMenu(header Rect, rowH, pad int) menuGeom {
	w := min(menuW, header.W)
	panel := Rect{header.X + header.W - pad - w, header.Y + header.H, w, 0}
	panel.X = max(panel.X, header.X)
	x, inner := panel.X+pad, w-2*pad
	y := panel.Y + pad

	chips := func(n int) []Rect {
		cw := (inner - (n-1)*chipGap) / n
		out := make([]Rect, n)
		for i := range out {
			out[i] = Rect{x + i*(cw+chipGap), y, cw, rowH}
		}
		y += rowH + pad
		return out
	}
	label := func() Rect {
		r := Rect{x, y, inner, rowH - 6}
		y += rowH - 6
		return r
	}

	g := menuGeom{Title: Rect{x, y, inner, rowH}}
	y += rowH + pad/2
	g.ThemeLabel = label()
	g.Theme = chips(len(themeChoices))
	g.SizeLabel = label()
	g.Size = chips(len(textSizeChoices))
	panel.H = y - panel.Y
	g.Panel = panel
	return g
}

// updateMenu handles the menu button and, while open, every click. It
// reports whether the input was consumed, so the panes below ignore it.
func (a *App) updateMenu(header Rect) bool {
	in := a.ctx.In
	th := a.ctx.Th
	if in.Hit(menuButton(header, th.Pad)) {
		a.menu = !a.menu
		return true
	}
	if !a.menu {
		return false
	}
	if in.Escape {
		a.menu = false
		return true
	}
	if !in.Click {
		return false
	}
	g := settingsMenu(header, th.RowH, th.Pad)
	for i, r := range g.Theme {
		if in.Hit(r) {
			a.changeSettings(func(s *settings.Settings) { s.Theme = themeChoices[i].id })
		}
	}
	for i, r := range g.Size {
		if in.Hit(r) {
			a.changeSettings(func(s *settings.Settings) { s.TextSize = textSizeChoices[i].id })
		}
	}
	if !g.Panel.Contains(in.X, in.Y) {
		a.menu = false
	}
	return true
}

// changeSettings edits the settings, applies them and saves them.
func (a *App) changeSettings(edit func(*settings.Settings)) {
	edit(&a.settings)
	if err := a.ctx.SetTheme(ThemeFor(a.settings.Theme, a.settings.TextSize)); err != nil {
		a.status = err.Error()
		return
	}
	if a.settingsPath == "" {
		return
	}
	if err := a.settings.Save(a.settingsPath); err != nil {
		a.status = "Could not save settings: " + err.Error()
	}
}

func (a *App) drawMenu(dst *ebiten.Image, header Rect) {
	if !a.menu {
		return
	}
	c := &a.ctx
	th := c.Th
	g := settingsMenu(header, th.RowH, th.Pad)

	c.fill(dst, Rect{g.Panel.X + 4, g.Panel.Y + 4, g.Panel.W, g.Panel.H}, color.RGBA{A: 0x66})
	c.fill(dst, g.Panel, th.Bg)
	c.outline(dst, g.Panel, th.Border)

	c.caption(dst, g.Title, "Settings")
	c.hline(dst, g.Title.X, g.Title.Y+g.Title.H-1, g.Title.W, th.Rule)
	c.caption(dst, g.ThemeLabel, "Theme")
	c.caption(dst, g.SizeLabel, "Text size")
	for i, r := range g.Theme {
		c.choice(dst, r, themeChoices[i].label, themeChoices[i].id == a.settings.Theme)
	}
	for i, r := range g.Size {
		c.choice(dst, r, textSizeChoices[i].label, textSizeChoices[i].id == a.settings.TextSize)
	}
}

// choice is one chip of a segmented choice: gold when selected.
func (c *Ctx) choice(dst *ebiten.Image, r Rect, label string, selected bool) {
	if selected {
		c.fill(dst, r, c.Th.Accent)
		c.textCenter(dst, c.F.UI, label, r, c.Th.AccentText)
		return
	}
	c.button(dst, r, label, c.Th.Chip, c.Th.ChipText)
}
