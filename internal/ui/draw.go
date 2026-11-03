package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// Fonts are the three faces the UI draws with.
type Fonts struct{ UI, Small, Mono *text.GoTextFace }

// NewFonts builds the faces from the embedded Go fonts, sized for a
// display with the given device scale factor.
func NewFonts(th Theme, scale float64) (Fonts, error) {
	regular, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return Fonts{}, fmt.Errorf("load regular font: %w", err)
	}
	mono, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		return Fonts{}, fmt.Errorf("load mono font: %w", err)
	}
	return Fonts{
		UI:    &text.GoTextFace{Source: regular, Size: th.FontSize * scale},
		Small: &text.GoTextFace{Source: regular, Size: th.SmallSize * scale},
		Mono:  &text.GoTextFace{Source: mono, Size: th.MonoSize * scale},
	}, nil
}

// Ctx is what widgets draw and hit-test with: theme, fonts and this
// frame's input. Widgets work in logical pixels; Scale is the number of
// device pixels per logical pixel, applied only when drawing.
type Ctx struct {
	Th    Theme
	F     Fonts
	In    Input
	Tick  int
	Scale float64
}

// SetScale rebuilds the fonts when the device scale factor changes, for
// example when the window moves to another monitor.
func (c *Ctx) SetScale(scale float64) error {
	if scale <= 0 || scale == c.Scale {
		return nil
	}
	fonts, err := NewFonts(c.Th, scale)
	if err != nil {
		return err
	}
	c.F, c.Scale = fonts, scale
	return nil
}

// SetTheme switches theme and rebuilds the fonts, whose sizes the theme sets.
func (c *Ctx) SetTheme(th Theme) error {
	fonts, err := NewFonts(th, c.Scale)
	if err != nil {
		return err
	}
	c.Th, c.F = th, fonts
	return nil
}

// caption draws s as a small uppercase muted label, vertically centered in r.
func (c *Ctx) caption(dst *ebiten.Image, r Rect, s string) {
	c.textMid(dst, c.F.Small, strings.ToUpper(s), r.X, r, c.Th.Muted)
}

// dev converts logical pixels to device pixels.
func (c *Ctx) dev(n float32) float32 { return n * float32(c.Scale) }

func (c *Ctx) fill(dst *ebiten.Image, r Rect, clr color.RGBA) {
	vector.FillRect(dst, c.dev(float32(r.X)), c.dev(float32(r.Y)), c.dev(float32(r.W)), c.dev(float32(r.H)), clr, false)
}

func (c *Ctx) outline(dst *ebiten.Image, r Rect, clr color.RGBA) {
	w := c.dev(1)
	vector.StrokeRect(dst, c.dev(float32(r.X))+w/2, c.dev(float32(r.Y))+w/2, c.dev(float32(r.W))-w, c.dev(float32(r.H))-w, w, clr, false)
}

func (c *Ctx) hline(dst *ebiten.Image, x, y, w int, clr color.RGBA) {
	c.fill(dst, Rect{x, y, w, 1}, clr)
}

// clip returns dst limited to r, so drawing outside r is discarded.
func (c *Ctx) clip(dst *ebiten.Image, r Rect) *ebiten.Image {
	b := dst.Bounds()
	x0 := max(int(c.dev(float32(r.X))), b.Min.X)
	y0 := max(int(c.dev(float32(r.Y))), b.Min.Y)
	x1 := min(int(c.dev(float32(r.X+r.W))+0.5), b.Max.X)
	y1 := min(int(c.dev(float32(r.Y+r.H))+0.5), b.Max.Y)
	return dst.SubImage(image.Rect(x0, y0, max(x1, x0), max(y1, y0))).(*ebiten.Image)
}

func (c *Ctx) width(face *text.GoTextFace, s string) int {
	w, _ := text.Measure(s, face, 0)
	return int(w/c.Scale + 0.5)
}

func (c *Ctx) lineH(face *text.GoTextFace) int {
	m := face.Metrics()
	return int((m.HAscent+m.HDescent)/c.Scale + 0.5)
}

// text draws s with its top-left at x, y.
func (c *Ctx) text(dst *ebiten.Image, face *text.GoTextFace, s string, x, y int, clr color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x)*c.Scale, float64(y)*c.Scale)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// textMid draws s starting at x, vertically centered in r.
func (c *Ctx) textMid(dst *ebiten.Image, face *text.GoTextFace, s string, x int, r Rect, clr color.RGBA) {
	c.text(dst, face, s, x, r.Y+(r.H-c.lineH(face))/2, clr)
}

// textCenter draws s centered in r.
func (c *Ctx) textCenter(dst *ebiten.Image, face *text.GoTextFace, s string, r Rect, clr color.RGBA) {
	c.textMid(dst, face, s, r.X+(r.W-c.width(face, s))/2, r, clr)
}

// ellipsize shortens s with "..." until it fits maxW pixels.
func (c *Ctx) ellipsize(face *text.GoTextFace, s string, maxW int) string {
	if c.width(face, s) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if cand := strings.TrimRight(string(runes), " ") + "..."; c.width(face, cand) <= maxW {
			return cand
		}
	}
	return ""
}

// triangle draws a filled arrowhead centered on cx, cy.
func (c *Ctx) triangle(dst *ebiten.Image, cx, cy, size float32, down bool, clr color.RGBA) {
	cx, cy, size = c.dev(cx), c.dev(cy), c.dev(size)
	var p vector.Path
	if down {
		p.MoveTo(cx-size, cy-size/2)
		p.LineTo(cx+size, cy-size/2)
		p.LineTo(cx, cy+size/2)
	} else {
		p.MoveTo(cx-size/2, cy-size)
		p.LineTo(cx+size/2, cy)
		p.LineTo(cx-size/2, cy+size)
	}
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &p, &vector.FillOptions{}, op)
}

// cross draws an x centered on cx, cy.
func (c *Ctx) cross(dst *ebiten.Image, cx, cy, size float32, clr color.RGBA) {
	c.line(dst, cx-size, cy-size, cx+size, cy+size, 1.5, clr)
	c.line(dst, cx-size, cy+size, cx+size, cy-size, 1.5, clr)
}

// line draws an antialiased segment between logical points.
func (c *Ctx) line(dst *ebiten.Image, x0, y0, x1, y1, width float32, clr color.RGBA) {
	vector.StrokeLine(dst, c.dev(x0), c.dev(y0), c.dev(x1), c.dev(y1), c.dev(width), clr, true)
}
