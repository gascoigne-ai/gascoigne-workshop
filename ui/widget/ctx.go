package widget

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

// Fonts are the three faces the toolkit draws with.
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
	Th Theme
	F  Fonts
	In Input
	// RawX and RawY are the pointer position as sampled, set by vml.Update
	// before any container hides it from its children; a drag that leaves
	// its container keeps following the pointer with them.
	RawX, RawY int
	Tick       int
	Scale      float64

	// Clipboard is used by text widgets for copy, cut and paste.
	Clipboard Clipboard

	// Tooltip state, kept by vml.Draw: the text of the element under the
	// pointer, whether it has been hovered long enough to show, and
	// whether it is still waiting to (the app keeps drawing meanwhile).
	TipText              string
	TipShown, TipWaiting bool
	TipFor               string // the text the delay is counted for
	TipSince             int    // tick at which the pointer came to rest on it

	// HoverOff turns hover feedback off for everything drawn while it is
	// set: elements under a modal overlay must not light up.
	HoverOff bool

	// Cursor is the mouse cursor shape widgets ask for this frame; the
	// application applies it. It is reset to CursorDefault every frame.
	Cursor Cursor
}

// Cursor is a mouse cursor shape.
type Cursor int

const (
	CursorDefault Cursor = iota
	CursorResizeH        // resize left/right, over a vertical divider
	CursorResizeV        // resize up/down, over a horizontal divider
)

// NewCtx returns a context at scale 1.
func NewCtx(th Theme) (*Ctx, error) {
	fonts, err := NewFonts(th, 1)
	if err != nil {
		return nil, err
	}
	return &Ctx{Th: th, F: fonts, Scale: 1, Clipboard: SystemClipboard()}, nil
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

// Hovered reports whether the pointer is over r and hover feedback is on.
func (c *Ctx) Hovered(r Rect) bool { return !c.HoverOff && c.In.Over(r) }

// Col resolves a color token in the current theme.
func (c *Ctx) Col(tok Token) color.RGBA { return c.Th.Color(tok) }

func (c *Ctx) dev(n float32) float32 { return n * float32(c.Scale) }

// Fill paints r.
func (c *Ctx) Fill(dst *ebiten.Image, r Rect, clr color.RGBA) {
	vector.FillRect(dst, c.dev(float32(r.X)), c.dev(float32(r.Y)), c.dev(float32(r.W)), c.dev(float32(r.H)), clr, false)
}

// Outline strokes a one-pixel border just inside r.
func (c *Ctx) Outline(dst *ebiten.Image, r Rect, clr color.RGBA) {
	w := c.dev(1)
	vector.StrokeRect(dst, c.dev(float32(r.X))+w/2, c.dev(float32(r.Y))+w/2, c.dev(float32(r.W))-w, c.dev(float32(r.H))-w, w, clr, false)
}

// HLine draws a one-pixel horizontal line.
func (c *Ctx) HLine(dst *ebiten.Image, x, y, w int, clr color.RGBA) {
	c.Fill(dst, Rect{x, y, w, 1}, clr)
}

// VLine draws a one-pixel vertical line.
func (c *Ctx) VLine(dst *ebiten.Image, x, y, h int, clr color.RGBA) {
	c.Fill(dst, Rect{x, y, 1, h}, clr)
}

// Clip returns dst limited to r, so drawing outside r is discarded.
func (c *Ctx) Clip(dst *ebiten.Image, r Rect) *ebiten.Image {
	b := dst.Bounds()
	x0 := max(int(c.dev(float32(r.X))), b.Min.X)
	y0 := max(int(c.dev(float32(r.Y))), b.Min.Y)
	x1 := min(int(c.dev(float32(r.X+r.W))+0.5), b.Max.X)
	y1 := min(int(c.dev(float32(r.Y+r.H))+0.5), b.Max.Y)
	return dst.SubImage(image.Rect(x0, y0, max(x1, x0), max(y1, y0))).(*ebiten.Image)
}

// Width is the logical width of s in face.
func (c *Ctx) Width(face *text.GoTextFace, s string) int {
	w, _ := text.Measure(s, face, 0)
	return int(w/c.Scale + 0.5)
}

// LineH is the logical line height of face.
func (c *Ctx) LineH(face *text.GoTextFace) int {
	m := face.Metrics()
	return int((m.HAscent+m.HDescent)/c.Scale + 0.5)
}

// Ascent is the logical distance from the top of a line of face to its baseline.
func (c *Ctx) Ascent(face *text.GoTextFace) int { return int(face.Metrics().HAscent/c.Scale + 0.5) }

// Text draws s with its top-left at x, y.
func (c *Ctx) Text(dst *ebiten.Image, face *text.GoTextFace, s string, x, y int, clr color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x)*c.Scale, float64(y)*c.Scale)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// TextMid draws s starting at x, vertically centered in r.
func (c *Ctx) TextMid(dst *ebiten.Image, face *text.GoTextFace, s string, x int, r Rect, clr color.RGBA) {
	c.Text(dst, face, s, x, r.Y+(r.H-c.LineH(face))/2, clr)
}

// TextCenter draws s centered in r.
func (c *Ctx) TextCenter(dst *ebiten.Image, face *text.GoTextFace, s string, r Rect, clr color.RGBA) {
	c.TextMid(dst, face, s, r.X+(r.W-c.Width(face, s))/2, r, clr)
}

// Ellipsize shortens s with "..." until it fits maxW pixels.
func (c *Ctx) Ellipsize(face *text.GoTextFace, s string, maxW int) string {
	if c.Width(face, s) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if cand := strings.TrimRight(string(runes), " ") + "..."; c.Width(face, cand) <= maxW {
			return cand
		}
	}
	return ""
}

// CursorAt maps a pixel offset into s to the nearest rune boundary.
func (c *Ctx) CursorAt(s string, x int) int {
	runes := []rune(s)
	best, bestDist := 0, abs(x)
	for i := 1; i <= len(runes); i++ {
		if d := abs(x - c.Width(c.F.UI, string(runes[:i]))); d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Triangle draws a filled arrowhead centered on cx, cy.
func (c *Ctx) Triangle(dst *ebiten.Image, cx, cy, size float32, down bool, clr color.RGBA) {
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

// Cross draws an x centered on cx, cy.
func (c *Ctx) Cross(dst *ebiten.Image, cx, cy, size float32, clr color.RGBA) {
	c.Line(dst, cx-size, cy-size, cx+size, cy+size, 1.5, clr)
	c.Line(dst, cx-size, cy+size, cx+size, cy-size, 1.5, clr)
}

// Line draws an antialiased segment between logical points.
func (c *Ctx) Line(dst *ebiten.Image, x0, y0, x1, y1, width float32, clr color.RGBA) {
	vector.StrokeLine(dst, c.dev(x0), c.dev(y0), c.dev(x1), c.dev(y1), c.dev(width), clr, true)
}

// Lighten nudges a color toward white, for hover states.
func Lighten(c color.RGBA) color.RGBA {
	const step = 16
	up := func(v uint8) uint8 { return uint8(min(int(v)+step, 255)) }
	return color.RGBA{R: up(c.R), G: up(c.G), B: up(c.B), A: c.A}
}
