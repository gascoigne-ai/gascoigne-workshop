package ui

import (
	"bytes"
	"fmt"
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

// NewFonts builds the faces from the embedded Go fonts.
func NewFonts(th Theme) (Fonts, error) {
	regular, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return Fonts{}, fmt.Errorf("load regular font: %w", err)
	}
	mono, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		return Fonts{}, fmt.Errorf("load mono font: %w", err)
	}
	return Fonts{
		UI:    &text.GoTextFace{Source: regular, Size: th.FontSize},
		Small: &text.GoTextFace{Source: regular, Size: th.SmallSize},
		Mono:  &text.GoTextFace{Source: mono, Size: th.MonoSize},
	}, nil
}

// Ctx is what widgets draw and hit-test with: theme, fonts and this
// frame's input.
type Ctx struct {
	Th   Theme
	F    Fonts
	In   Input
	Tick int
}

func (c *Ctx) fill(dst *ebiten.Image, r Rect, clr color.RGBA) {
	vector.FillRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), clr, false)
}

func (c *Ctx) outline(dst *ebiten.Image, r Rect, clr color.RGBA) {
	vector.StrokeRect(dst, float32(r.X)+0.5, float32(r.Y)+0.5, float32(r.W)-1, float32(r.H)-1, 1, clr, false)
}

func (c *Ctx) hline(dst *ebiten.Image, x, y, w int, clr color.RGBA) {
	c.fill(dst, Rect{x, y, w, 1}, clr)
}

// clip returns dst limited to r, so drawing outside r is discarded.
func (c *Ctx) clip(dst *ebiten.Image, r Rect) *ebiten.Image {
	b := dst.Bounds()
	r2 := Rect{max(r.X, b.Min.X), max(r.Y, b.Min.Y), 0, 0}
	r2.W = max(min(r.X+r.W, b.Max.X)-r2.X, 0)
	r2.H = max(min(r.Y+r.H, b.Max.Y)-r2.Y, 0)
	return dst.SubImage(imageRect(r2)).(*ebiten.Image)
}

func (c *Ctx) width(face *text.GoTextFace, s string) int {
	w, _ := text.Measure(s, face, 0)
	return int(w + 0.5)
}

func (c *Ctx) lineH(face *text.GoTextFace) int {
	m := face.Metrics()
	return int(m.HAscent + m.HDescent + 0.5)
}

// text draws s with its top-left at x, y.
func (c *Ctx) text(dst *ebiten.Image, face *text.GoTextFace, s string, x, y int, clr color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
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
	vector.StrokeLine(dst, cx-size, cy-size, cx+size, cy+size, 1.5, clr, true)
	vector.StrokeLine(dst, cx-size, cy+size, cx+size, cy-size, 1.5, clr, true)
}
