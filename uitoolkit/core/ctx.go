package core

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// Fonts are the faces the toolkit draws with: UI and Small text, Mono for code, and bold forms of UI and Small.
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
func newFonts(th Theme, scale float64, chosen FontChoice) (Fonts, error) {
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

// FontChoice is the fonts picked for the interface and for code.
type FontChoice struct{ UI, Code FontData }

// Focuser is the state of a component that takes the keyboard: it holds
// the focus flag the window and the runtime read and write. It lives here
// so Ctx can name the focus owner by it; view.Focuser is the same type.
type Focuser interface {
	HasFocus() bool
	SetFocus(on bool)
}

// Ctx is what widgets draw and hit-test with: theme, fonts and this
// frame's input. Widgets work in logical pixels; Scale is the number of
// device pixels per logical pixel, applied only when drawing.
type Ctx struct {
	Th Theme
	F  Fonts
	// Chosen are the fonts picked by the user; the zero value is the
	// embedded ones. They survive theme and scale changes.
	Chosen FontChoice
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
	FocusOwner Focuser

	// Fired counts the handlers (OnClick, OnSelect, OnEnter, ...) that ran
	// since the app began counting. A handler changes app state, so the app
	// compares it before and after view.Update to know whether the tree it
	// built still shows that state. A widget that calls a handler adds one.
	Fired int

	// LayoutGen counts the passes of view.Layout and Laying is true inside
	// one: an element's measured size is remembered for one pass only, and
	// outside a pass (a size asked for by an Update handler) it is never
	// remembered, because the content may have changed since.
	LayoutGen int
	Laying    bool

	// Stats counts the draw calls of the frame when a benchmark or test sets
	// it; nil in the app, where counting costs one nil check.
	Stats *DrawStats

	// masks caches the corner masks of rounded fills; maskMaker builds one
	// and is nil in the app (a test sets it to need no GPU).
	masks     *cornerMasks
	maskMaker func(radius int) *ebiten.Image
	ringMaker func(radius, width int) *ebiten.Image

	// TipText is the tooltip text of the element under the pointer, set by
	// view.Draw for widget.TooltipLayer to show.
	TipText string

	// Menu is the open popup menu; widget owns its type (*widget.MenuState) and
	// core only carries it, so that any element can open a menu without a
	// host. nil when none is open. It is any on purpose (decision 27): core
	// cannot import widget, and a typed field would need a widget type here.
	Menu any

	// HostOS is the operating system the shortcut text and modifier follow:
	// runtime.GOOS from NewCtx; a test sets it to pin the text across machines.
	HostOS string

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
	fonts, err := newFonts(th, 1, FontChoice{})
	if err != nil {
		return nil, err
	}
	return &Ctx{Th: th, F: fonts, Scale: 1, Clipboard: SystemClipboard(), HostOS: runtime.GOOS}, nil
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
	c.resetMasks()
	return nil
}

// SetTheme switches theme and rebuilds the fonts, whose sizes the theme sets.
func (c *Ctx) SetTheme(th Theme) error {
	fonts, err := newFonts(th, c.Scale, c.Chosen)
	if err != nil {
		return err
	}
	c.Th, c.F = th, fonts
	c.resetMasks()
	return nil
}

// SetFonts switches the interface and code fonts; empty data is the
// embedded font. Fonts that cannot be read change nothing.
func (c *Ctx) SetFonts(ui, code FontData) error {
	chosen := FontChoice{UI: ui, Code: code}
	fonts, err := newFonts(c.Th, c.Scale, chosen)
	if err != nil {
		return err
	}
	c.Chosen, c.F = chosen, fonts
	return nil
}

// Hovered reports whether the pointer is over r and hover feedback is on.
func (c *Ctx) Hovered(rect Rect) bool { return !c.HoverOff && c.In.Over(rect) }

// Color resolves a color token in the current theme.
func (c *Ctx) Color(tok Token) color.RGBA { return c.Th.Color(tok) }

func (c *Ctx) dev(value float32) float32 { return value * float32(c.Scale) }

// devRect is rect's corners in device pixels: left, top, right, bottom.
func (c *Ctx) devRect(rect Rect) (x0, y0, x1, y1 float32) {
	return c.dev(float32(rect.X)), c.dev(float32(rect.Y)), c.dev(float32(rect.X + rect.W)), c.dev(float32(rect.Y + rect.H))
}

// snap is a logical coordinate as a device pixel position. Rounding the
// edges (not the sizes) lets neighbouring rectangles share an edge at any
// scale, with no gap or overlap.
func (c *Ctx) snap(pixels int) float32 { return float32(math.Round(float64(pixels) * c.Scale)) }

// LineWidth is the thickness in device pixels of the one-pixel lines and
// outlines: the scale rounded, never less than one, so a line is a whole
// number of device pixels and stays sharp at 125%, 150% and 175%.
func (c *Ctx) LineWidth() float32 { return float32(max(1, math.Round(c.Scale))) }

// Fill paints r with clr, its edges on device pixels.
func (c *Ctx) Fill(dst *ebiten.Image, rect Rect, clr color.RGBA) {
	x0, y0 := c.snap(rect.X), c.snap(rect.Y)
	vector.FillRect(dst, x0, y0, c.snap(rect.X+rect.W)-x0, c.snap(rect.Y+rect.H)-y0, clr, false)
}

// Outline strokes a one-pixel border just inside r.
func (c *Ctx) Outline(dst *ebiten.Image, rect Rect, clr color.RGBA) {
	w := c.LineWidth()
	x0, y0 := c.snap(rect.X), c.snap(rect.Y)
	vector.StrokeRect(dst, x0+w/2, y0+w/2, c.snap(rect.X+rect.W)-x0-w, c.snap(rect.Y+rect.H)-y0-w, w, clr, false)
}

// HLine draws a one-pixel horizontal line.
func (c *Ctx) HLine(dst *ebiten.Image, posX, posY, width int, clr color.RGBA) {
	x0 := c.snap(posX)
	vector.FillRect(dst, x0, c.snap(posY), c.snap(posX+width)-x0, c.LineWidth(), clr, false)
}

// VLine draws a one-pixel vertical line.
func (c *Ctx) VLine(dst *ebiten.Image, posX, posY, height int, clr color.RGBA) {
	y0 := c.snap(posY)
	vector.FillRect(dst, c.snap(posX), y0, c.LineWidth(), c.snap(posY+height)-y0, clr, false)
}

// Clip returns dst limited to r, so drawing outside r is discarded.
func (c *Ctx) Clip(dst *ebiten.Image, rect Rect) *ebiten.Image {
	b := dst.Bounds()
	x0 := max(int(c.dev(float32(rect.X))), b.Min.X)
	y0 := max(int(c.dev(float32(rect.Y))), b.Min.Y)
	x1 := min(int(c.dev(float32(rect.X+rect.W))+0.5), b.Max.X)
	y1 := min(int(c.dev(float32(rect.Y+rect.H))+0.5), b.Max.Y)
	return dst.SubImage(image.Rect(x0, y0, max(x1, x0), max(y1, y0))).(*ebiten.Image)
}

// Width is the logical width of s in face.
func (c *Ctx) Width(face *text.GoTextFace, content string) int {
	if c.Stats != nil {
		c.Stats.Widths++
	}
	w, _ := text.Measure(content, face, 0)
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
func (c *Ctx) Text(dst *ebiten.Image, face *text.GoTextFace, content string, posX, posY int, clr color.RGBA) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(posX)*c.Scale, float64(posY)*c.Scale)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, content, face, op)
}

// TextMid draws s starting at x, vertically centered in r.
func (c *Ctx) TextMid(dst *ebiten.Image, face *text.GoTextFace, content string, posX int, rect Rect, clr color.RGBA) {
	c.Text(dst, face, content, posX, rect.Y+(rect.H-c.LineH(face))/2, clr)
}

// TextCenter draws s centered in r.
func (c *Ctx) TextCenter(dst *ebiten.Image, face *text.GoTextFace, content string, rect Rect, clr color.RGBA) {
	c.TextMid(dst, face, content, rect.X+(rect.W-c.Width(face, content))/2, rect, clr)
}

// Ellipsize shortens s with "..." until it fits maxW pixels. The longest
// prefix whose trimmed form plus the dots fits is found by binary search on
// the rune count, as CursorAt does, not by measuring every prefix.
func (c *Ctx) Ellipsize(face *text.GoTextFace, content string, maxW int) string {
	if c.Width(face, content) <= maxW {
		return content
	}
	runes := []rune(content)
	candidate := func(count int) string { return strings.TrimRight(string(runes[:count]), " ") + "..." }
	fits := func(count int) bool { return c.Width(face, candidate(count)) <= maxW }
	low, high := 0, len(runes)-1 // the longest count that fits is in low-1..high
	for low < high {
		middle := (low + high + 1) / 2
		if fits(middle) {
			low = middle
		} else {
			high = middle - 1
		}
	}
	if !fits(low) {
		return ""
	}
	return candidate(low)
}

// CursorAt maps a pixel offset into s to the nearest rune boundary. The
// width of a prefix grows with its length, so the boundary is found by
// binary search: measuring every prefix would be quadratic in the length
// of the text.
func (c *Ctx) CursorAt(content string, posX int) int {
	runes := []rune(content)
	widthTo := func(count int) int { return c.Width(c.F.UI, string(runes[:count])) }
	low, high := 0, len(runes) // the first boundary whose width reaches x is in low..high
	for low < high {
		middle := (low + high) / 2
		if widthTo(middle) < posX {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low > 0 && abs(posX-widthTo(low-1)) <= abs(posX-widthTo(low)) {
		return low - 1
	}
	return low
}

func abs(count int) int {
	if count < 0 {
		return -count
	}
	return count
}

// Line draws an antialiased segment between logical points.
func (c *Ctx) Line(dst *ebiten.Image, x0, y0, x1, y1, width float32, clr color.RGBA) {
	vector.StrokeLine(dst, c.dev(x0), c.dev(y0), c.dev(x1), c.dev(y1), c.dev(width), clr, true)
}

var gradientSrc = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()

// Radius is the corner radius of the named control in this theme, after
// the user's corner setting: what every component draws its corners with.
func (c *Ctx) Radius(control string) float64 { return c.Th.Style(control).Radius }

// FillStyled paints r the way a control style says: a flat or vertical
// gradient fill from top to bottom, with rounded corners when the style
// has a radius. Square flat fills stay on the crisp rectangle path.
func (c *Ctx) FillStyled(dst *ebiten.Image, rect Rect, top, bottom color.RGBA, radius float64) {
	if radius*c.Scale >= 1 {
		c.fillRounded(dst, rect, top, bottom, radius, CornersAll)
		return
	}
	if top == bottom {
		c.Fill(dst, rect, top)
		return
	}
	x0, y0, x1, y1 := c.devRect(rect)
	c.fillGradient(dst, rectPath(x0, y0, x1, y1), y0, y1, top, bottom)
}

// fillGradient fills the square-cornered path, which spans y0 to y1 (device
// pixels), from top to bottom.
func (c *Ctx) fillGradient(dst *ebiten.Image, path *vector.Path, y0, y1 float32, top, bottom color.RGBA) {
	verts, idx := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range verts {
		verts[i].ColorR, verts[i].ColorG, verts[i].ColorB, verts[i].ColorA = gradientAt(top, bottom, y0, y1, verts[i].DstY)
		verts[i].SrcX, verts[i].SrcY = 0.5, 0.5
	}
	if len(verts) == 0 {
		return
	}
	dst.DrawTriangles(verts, idx, gradientSrc, &ebiten.DrawTrianglesOptions{})
}

// fillRounded paints rect with the corners in the set rounded: the edges
// snap to device pixels as Fill's do, and the corners are quarters of a
// cached anti-aliased disc, so the whole fill is one DrawTriangles with no
// offscreen. The gradient rides on the vertex colors.
func (c *Ctx) fillRounded(dst *ebiten.Image, rect Rect, top, bottom color.RGBA, radius float64, corners Corners) {
	x0, y0 := c.snap(rect.X), c.snap(rect.Y)
	x1, y1 := c.snap(rect.X+rect.W), c.snap(rect.Y+rect.H)
	r := min(float32(max(1, math.Round(float64(c.dev(float32(radius)))))), float32(math.Floor(float64(min(x1-x0, y1-y0))/2)))
	if r < 1 || corners == 0 {
		c.fillGradient(dst, rectPath(x0, y0, x1, y1), y0, y1, top, bottom)
		return
	}
	mask := c.cornerMask(int(r))
	quads := sliceQuads(x0, y0, x1, y1, r, corners)
	verts := make([]ebiten.Vertex, 0, 4*len(quads))
	idx := make([]uint16, 0, 6*len(quads))
	for _, q := range quads {
		base := uint16(len(verts))
		for _, corner := range [4][4]float32{
			{q.dst.x0, q.dst.y0, q.src.x0, q.src.y0}, {q.dst.x1, q.dst.y0, q.src.x1, q.src.y0},
			{q.dst.x1, q.dst.y1, q.src.x1, q.src.y1}, {q.dst.x0, q.dst.y1, q.src.x0, q.src.y1},
		} {
			red, green, blue, alpha := gradientAt(top, bottom, y0, y1, corner[1])
			verts = append(verts, ebiten.Vertex{DstX: corner[0], DstY: corner[1], SrcX: corner[2], SrcY: corner[3], ColorR: red, ColorG: green, ColorB: blue, ColorA: alpha})
		}
		idx = append(idx, base, base+1, base+2, base, base+2, base+3)
	}
	if c.Stats != nil {
		c.Stats.MaskFills++
	}
	dst.DrawTriangles(verts, idx, mask, &ebiten.DrawTrianglesOptions{})
}

// rectPath is the square-cornered rectangle as a path.
func rectPath(x0, y0, x1, y1 float32) *vector.Path {
	var path vector.Path
	path.MoveTo(x0, y0)
	path.LineTo(x1, y0)
	path.LineTo(x1, y1)
	path.LineTo(x0, y1)
	path.Close()
	return &path
}

// DrawStats counts what one frame asks the GPU for, to pin and measure the
// batching of rounded shapes. Set Ctx.Stats to collect; the caller zeroes it
// between frames.
type DrawStats struct {
	MaskFills int // rounded fills drawn from the corner mask, one DrawTriangles each
	RingFills int // rounded outlines drawn from the ring mask, one DrawTriangles each
	Widths    int // Ctx.Width measurements
}

// Corners is a set of rectangle corners.
type Corners uint8

// The corners of a rectangle.
const (
	CornerTL Corners = 1 << iota
	CornerTR
	CornerBR
	CornerBL

	CornersTop    = CornerTL | CornerTR
	CornersBottom = CornerBL | CornerBR
	CornersAll    = CornersTop | CornersBottom
)

// FillStyledCorners is FillStyled with only the given corners rounded, for
// a bar or a column that sits in the corner of a rounded panel: its corner
// follows the panel's, with anti-aliased (alpha) edges, instead of poking
// out past it.
func (c *Ctx) FillStyledCorners(dst *ebiten.Image, rect Rect, top, bottom color.RGBA, radius float64, corners Corners) {
	if radius*c.Scale < 1 || corners == 0 {
		c.FillStyled(dst, rect, top, bottom, 0)
		return
	}
	c.fillRounded(dst, rect, top, bottom, radius, corners)
}

// cornersPath is r with the corners in the set rounded by rad.
func cornersPath(x0, y0, x1, y1, rad float32, corners Corners) *vector.Path {
	rad = min(rad, (x1-x0)/2, (y1-y0)/2)
	r := func(cornersValue Corners) float32 {
		if corners&cornersValue != 0 {
			return rad
		}
		return 0
	}
	tl, tr, br, bl := r(CornerTL), r(CornerTR), r(CornerBR), r(CornerBL)
	var p vector.Path
	p.MoveTo(x0+tl, y0)
	p.LineTo(x1-tr, y0)
	if tr > 0 {
		p.Arc(x1-tr, y0+tr, tr, -math.Pi/2, 0, vector.Clockwise)
	}
	p.LineTo(x1, y1-br)
	if br > 0 {
		p.Arc(x1-br, y1-br, br, 0, math.Pi/2, vector.Clockwise)
	}
	p.LineTo(x0+bl, y1)
	if bl > 0 {
		p.Arc(x0+bl, y1-bl, bl, math.Pi/2, math.Pi, vector.Clockwise)
	}
	p.LineTo(x0, y0+tl)
	if tl > 0 {
		p.Arc(x0+tl, y0+tl, tl, math.Pi, 3*math.Pi/2, vector.Clockwise)
	}
	p.Close()
	return &p
}

// OutlineStyled strokes a border just inside r, LineWidth thick, with
// rounded corners when radius is set. The edges snap to device pixels as
// Outline's do, and the rounded outline is one DrawTriangles from the cached
// ring mask. The width is LineWidth (the unrounded scale a stroke used to
// take would blur on a fractional scale and not match Outline's lines).
func (c *Ctx) OutlineStyled(dst *ebiten.Image, rect Rect, clr color.RGBA, radius float64) {
	if radius*c.Scale < 1 {
		c.Outline(dst, rect, clr)
		return
	}
	x0, y0 := c.snap(rect.X), c.snap(rect.Y)
	x1, y1 := c.snap(rect.X+rect.W), c.snap(rect.Y+rect.H)
	r := min(float32(max(1, math.Round(float64(c.dev(float32(radius)))))), float32(math.Floor(float64(min(x1-x0, y1-y0))/2)))
	if r < 1 {
		c.Outline(dst, rect, clr)
		return
	}
	width := min(c.LineWidth(), r)
	mask := c.ringMask(int(r), int(width))
	quads := ringQuads(x0, y0, x1, y1, r, width)
	red, green, blue, alpha := float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255
	verts := make([]ebiten.Vertex, 0, 4*len(quads))
	idx := make([]uint16, 0, 6*len(quads))
	for _, q := range quads {
		base := uint16(len(verts))
		for _, corner := range [4][4]float32{
			{q.dst.x0, q.dst.y0, q.src.x0, q.src.y0}, {q.dst.x1, q.dst.y0, q.src.x1, q.src.y0},
			{q.dst.x1, q.dst.y1, q.src.x1, q.src.y1}, {q.dst.x0, q.dst.y1, q.src.x0, q.src.y1},
		} {
			verts = append(verts, ebiten.Vertex{DstX: corner[0], DstY: corner[1], SrcX: corner[2], SrcY: corner[3], ColorR: red, ColorG: green, ColorB: blue, ColorA: alpha})
		}
		idx = append(idx, base, base+1, base+2, base, base+2, base+3)
	}
	if c.Stats != nil {
		c.Stats.RingFills++
	}
	dst.DrawTriangles(verts, idx, mask, &ebiten.DrawTrianglesOptions{})
}

// cornerMask is the cached white mask for corners of radius device pixels.
func (c *Ctx) cornerMask(radius int) *ebiten.Image { return c.maskCache().get(radius) }

// ringMask is the cached white ring mask for corners of radius device pixels
// and a line width in device pixels.
func (c *Ctx) ringMask(radius, width int) *ebiten.Image { return c.maskCache().getRing(radius, width) }

func (c *Ctx) maskCache() *cornerMasks {
	if c.masks == nil {
		c.masks = &cornerMasks{make: buildCornerMask, makeRing: buildRingMask}
	}
	if c.maskMaker != nil {
		c.masks.make = c.maskMaker
	}
	if c.ringMaker != nil {
		c.masks.makeRing = c.ringMaker
	}
	return c.masks
}

func (c *Ctx) resetMasks() {
	if c.masks != nil {
		c.masks.reset()
	}
}
