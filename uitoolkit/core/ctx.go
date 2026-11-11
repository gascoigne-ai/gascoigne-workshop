package core

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// Fonts are the three faces the toolkit draws with.
type Fonts struct{ UI, Small, Mono, Bold, SmallBold *text.GoTextFace }

// FontData is the content of a regular and a bold font file. The zero
// value means the embedded Go font.
type FontData struct{ Regular, Bold []byte }

// withDefault is d, or the embedded fonts when d is empty.
func (d FontData) withDefault(regular, bold []byte) FontData {
	if len(d.Regular) == 0 {
		return FontData{Regular: regular, Bold: bold}
	}
	if len(d.Bold) == 0 {
		d.Bold = d.Regular
	}
	return d
}

// newFonts builds the faces from the chosen fonts (the embedded Go fonts
// where none was chosen), sized for a display with the given device scale
// factor.
func newFonts(th Theme, scale float64, chosen Choice) (Fonts, error) {
	ui := chosen.UI.withDefault(goregular.TTF, gobold.TTF)
	code := chosen.Code.withDefault(gomono.TTF, gobold.TTF)
	regular, err := text.NewGoTextFaceSource(bytes.NewReader(ui.Regular))
	if err != nil {
		return Fonts{}, fmt.Errorf("load regular font: %w", err)
	}
	bold, err := text.NewGoTextFaceSource(bytes.NewReader(ui.Bold))
	if err != nil {
		return Fonts{}, fmt.Errorf("load bold font: %w", err)
	}
	mono, err := text.NewGoTextFaceSource(bytes.NewReader(code.Regular))
	if err != nil {
		return Fonts{}, fmt.Errorf("load mono font: %w", err)
	}
	return Fonts{
		UI:        &text.GoTextFace{Source: regular, Size: th.FontSize * scale},
		Small:     &text.GoTextFace{Source: regular, Size: th.SmallSize * scale},
		Mono:      &text.GoTextFace{Source: mono, Size: th.MonoSize * scale},
		Bold:      &text.GoTextFace{Source: bold, Size: th.FontSize * scale},
		SmallBold: &text.GoTextFace{Source: bold, Size: th.SmallSize * scale},
	}, nil
}

// Choice is the fonts picked for the interface and for code.
type Choice struct{ UI, Code FontData }

// Ctx is what widgets draw and hit-test with: theme, fonts and this
// frame's input. Widgets work in logical pixels; Scale is the number of
// device pixels per logical pixel, applied only when drawing.
type Ctx struct {
	Th Theme
	F  Fonts
	// Chosen are the fonts picked by the user; the zero value is the
	// embedded ones. They survive theme and scale changes.
	Chosen Choice
	In     Input
	// PointerX and PointerY are the pointer position in logical pixels as
	// sampled, set by view.Update before any container hides it from its
	// children; a drag that leaves its container keeps following the
	// pointer with them.
	PointerX, PointerY int
	Tick               int

	// Now is the clock the caret blinks by; nil is time.Now. CaretSince is
	// when the caret last became visible (the last input).
	Now        func() time.Time
	CaretSince time.Time
	Scale      float64

	// Clipboard is used by text widgets for copy, cut and paste.
	Clipboard Clipboard

	// FocusOwner is the one widget state (a *TextField, *TextArea or
	// *CodeView) that has the keyboard, nil for none. view.Update keeps it:
	// the app asks for focus by setting a state's Focused, and the owner
	// is whoever asked last; any other focused state, and one that is no
	// longer in the tree, is blurred.
	FocusOwner any

	// Tooltip state, kept by view.Draw: the text of the element under the
	// pointer, whether it has been hovered long enough to show, and
	// whether it is still waiting to (the app keeps drawing meanwhile).
	TipText              string
	TipShown, TipWaiting bool
	TipFor               string // the text the delay is counted for
	TipSince             int    // tick at which the pointer came to rest on it

	// Menu is the open popup menu; view owns its type (*widget.MenuState) and
	// core only carries it, so that any element can open a menu without a
	// host. nil when none is open.
	Menu any

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
	CursorDefault    Cursor = iota
	CursorResizeH           // resize left/right, over a vertical divider
	CursorResizeV           // resize up/down, over a horizontal divider
	CursorResizeNWSE        // resize along a diagonal, over a top-left or bottom-right corner
	CursorResizeNESW        // resize along the other diagonal
	CursorMove              // something is held and being moved: a row, a tab
)

// NewCtx returns a context at scale 1.
func NewCtx(th Theme) (*Ctx, error) {
	fonts, err := newFonts(th, 1, Choice{})
	if err != nil {
		return nil, err
	}
	return &Ctx{Th: th, F: fonts, Scale: 1, Clipboard: SystemClipboard()}, nil
}

// CaretHalfPeriod is how long the caret is shown, and then hidden.
const CaretHalfPeriod = 500 * time.Millisecond

func (c *Ctx) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// RestartCaret makes the caret visible now and starts its blink over, as an
// edit or a click does.
func (c *Ctx) RestartCaret() { c.CaretSince = c.now() }

// CaretVisible reports whether the blinking caret is in its shown phase.
// The phase follows the clock, so it does not depend on how often the app
// updates or draws.
func (c *Ctx) CaretVisible() bool {
	if c.CaretSince.IsZero() {
		return true
	}
	return (c.now().Sub(c.CaretSince)/CaretHalfPeriod)%2 == 0
}

// SetScale rebuilds the fonts when the device scale factor changes, for
// example when the window moves to another monitor.
func (c *Ctx) SetScale(scale float64) error {
	if scale <= 0 || scale == c.Scale {
		return nil
	}
	fonts, err := newFonts(c.Th, scale, c.Chosen)
	if err != nil {
		return err
	}
	c.F, c.Scale = fonts, scale
	return nil
}

// SetTheme switches theme and rebuilds the fonts, whose sizes the theme sets.
func (c *Ctx) SetTheme(th Theme) error {
	fonts, err := newFonts(th, c.Scale, c.Chosen)
	if err != nil {
		return err
	}
	c.Th, c.F = th, fonts
	return nil
}

// SetFonts switches the interface and code fonts; empty data is the
// embedded font. Fonts that cannot be read change nothing.
func (c *Ctx) SetFonts(ui, code FontData) error {
	chosen := Choice{UI: ui, Code: code}
	fonts, err := newFonts(c.Th, c.Scale, chosen)
	if err != nil {
		return err
	}
	c.Chosen, c.F = chosen, fonts
	return nil
}

// Hovered reports whether the pointer is over r and hover feedback is on.
func (c *Ctx) Hovered(r Rect) bool { return !c.HoverOff && c.In.Over(r) }

// Col resolves a color token in the current theme.
func (c *Ctx) Color(tok Token) color.RGBA { return c.Th.Color(tok) }

func (c *Ctx) dev(n float32) float32 { return n * float32(c.Scale) }

// snap is a logical coordinate as a device pixel position. Rounding the
// edges (not the sizes) lets neighbouring rectangles share an edge at any
// scale, with no gap or overlap.
func (c *Ctx) snap(n int) float32 { return float32(math.Round(float64(n) * c.Scale)) }

// LineWidth is the thickness in device pixels of the one-pixel lines and
// outlines: the scale rounded, never less than one, so a line is a whole
// number of device pixels and stays sharp at 125%, 150% and 175%.
func (c *Ctx) LineWidth() float32 { return float32(max(1, math.Round(c.Scale))) }

// Fill paints r with clr, its edges on device pixels.
func (c *Ctx) Fill(dst *ebiten.Image, r Rect, clr color.RGBA) {
	x0, y0 := c.snap(r.X), c.snap(r.Y)
	vector.FillRect(dst, x0, y0, c.snap(r.X+r.W)-x0, c.snap(r.Y+r.H)-y0, clr, false)
}

// Outline strokes a one-pixel border just inside r.
func (c *Ctx) Outline(dst *ebiten.Image, r Rect, clr color.RGBA) {
	w := c.LineWidth()
	x0, y0 := c.snap(r.X), c.snap(r.Y)
	vector.StrokeRect(dst, x0+w/2, y0+w/2, c.snap(r.X+r.W)-x0-w, c.snap(r.Y+r.H)-y0-w, w, clr, false)
}

// HLine draws a one-pixel horizontal line.
func (c *Ctx) HLine(dst *ebiten.Image, x, y, w int, clr color.RGBA) {
	x0 := c.snap(x)
	vector.FillRect(dst, x0, c.snap(y), c.snap(x+w)-x0, c.LineWidth(), clr, false)
}

// VLine draws a one-pixel vertical line.
func (c *Ctx) VLine(dst *ebiten.Image, x, y, h int, clr color.RGBA) {
	y0 := c.snap(y)
	vector.FillRect(dst, c.snap(x), y0, c.LineWidth(), c.snap(y+h)-y0, clr, false)
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

// Advance is the exact logical width of one character of a monospace face,
// not rounded: columns placed at whole multiples of the rounded Width drift
// away from the text, which is drawn with the exact advance.
func (c *Ctx) Advance(face *text.GoTextFace) float64 {
	w, _ := text.Measure("0", face, 0)
	return w / c.Scale
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

var gradientSrc = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()

// roundedPath is r (device pixels) with corner radius rad, clamped to half
// the shorter side so a large radius makes a pill.
func roundedPath(x0, y0, x1, y1, rad float32) *vector.Path {
	rad = min(rad, (x1-x0)/2, (y1-y0)/2)
	var p vector.Path
	p.MoveTo(x0+rad, y0)
	p.LineTo(x1-rad, y0)
	p.Arc(x1-rad, y0+rad, rad, -math.Pi/2, 0, vector.Clockwise)
	p.LineTo(x1, y1-rad)
	p.Arc(x1-rad, y1-rad, rad, 0, math.Pi/2, vector.Clockwise)
	p.LineTo(x0+rad, y1)
	p.Arc(x0+rad, y1-rad, rad, math.Pi/2, math.Pi, vector.Clockwise)
	p.LineTo(x0, y0+rad)
	p.Arc(x0+rad, y0+rad, rad, math.Pi, 3*math.Pi/2, vector.Clockwise)
	p.Close()
	return &p
}

// Radius is the corner radius of the named control in this theme, after
// the user's corner setting: what every component draws its corners with.
func (c *Ctx) Radius(control string) float64 { return c.Th.Style(control).Radius }

// GradientFor is the top and bottom colors a control's fill takes from
// base: the style's explicit ends, else base shaded by the style's
// strength, else base twice.
func GradientFor(base color.RGBA, st ControlStyle) (top, bottom color.RGBA) {
	switch {
	case st.Top != nil && st.Bottom != nil:
		return *st.Top, *st.Bottom
	case st.Shade > 0:
		return gradientEnds(base, st.Shade)
	}
	return base, base
}

// FillStyled paints r the way a control style says: a flat or vertical
// gradient fill from top to bottom, with rounded corners when the style
// has a radius. Square flat fills stay on the crisp rectangle path.
func (c *Ctx) FillStyled(dst *ebiten.Image, r Rect, top, bottom color.RGBA, radius float64) {
	round := radius*c.Scale >= 1
	if !round && top == bottom {
		c.Fill(dst, r, top)
		return
	}
	x0, y0 := c.dev(float32(r.X)), c.dev(float32(r.Y))
	x1, y1 := c.dev(float32(r.X+r.W)), c.dev(float32(r.Y+r.H))
	var path *vector.Path
	if round {
		path = roundedPath(x0, y0, x1, y1, c.dev(float32(radius)))
	} else {
		path = &vector.Path{}
		path.MoveTo(x0, y0)
		path.LineTo(x1, y0)
		path.LineTo(x1, y1)
		path.LineTo(x0, y1)
		path.Close()
	}
	verts, idx := path.AppendVerticesAndIndicesForFilling(nil, nil)
	span := max(y1-y0, 1)
	for i := range verts {
		t := min(max((verts[i].DstY-y0)/span, 0), 1)
		mix := func(a, b uint8) float32 { return (float32(a) + (float32(b)-float32(a))*t) / 255 }
		verts[i].ColorR, verts[i].ColorG, verts[i].ColorB, verts[i].ColorA = mix(top.R, bottom.R), mix(top.G, bottom.G), mix(top.B, bottom.B), mix(top.A, bottom.A)
		verts[i].SrcX, verts[i].SrcY = 0.5, 0.5
	}
	dst.DrawTriangles(verts, idx, gradientSrc, &ebiten.DrawTrianglesOptions{AntiAlias: round})
}

// OutlineStyled strokes a one-pixel border just inside r, with rounded
// corners when radius is set.
func (c *Ctx) OutlineStyled(dst *ebiten.Image, r Rect, clr color.RGBA, radius float64) {
	if radius*c.Scale < 1 {
		c.Outline(dst, r, clr)
		return
	}
	w := c.dev(1)
	x0, y0 := c.dev(float32(r.X))+w/2, c.dev(float32(r.Y))+w/2
	x1, y1 := c.dev(float32(r.X+r.W))-w/2, c.dev(float32(r.Y+r.H))-w/2
	path := roundedPath(x0, y0, x1, y1, c.dev(float32(radius))-w/2)
	var so vector.StrokeOptions
	so.Width = w
	var dp vector.DrawPathOptions
	dp.AntiAlias = true
	dp.ColorScale.ScaleWithColor(clr)
	vector.StrokePath(dst, path, &so, &dp)
}
