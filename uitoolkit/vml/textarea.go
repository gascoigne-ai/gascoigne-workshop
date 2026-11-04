package vml

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// TextArea is a multi-line editor: a multi-line TextField plus the
// view state (vertical scroll, optional syntax styling). Selection,
// undo, word movement and the clipboard come from the TextField.
type TextArea struct {
	TextField
	ScrollY int

	// Styler colors a line; nil draws plain text.
	Styler func(line string) []Span

	drag     barDrag
	revealed int // cursor position last scrolled into view
}

// NewTextArea returns an editor holding text.
func NewTextArea(text string) *TextArea {
	a := &TextArea{TextField: NewMultiline(text)}
	a.revealed = -1
	return a
}

// areaGeometry is the pixel layout of an area in r.
type areaGeometry struct {
	lineH, charW, textX int
	lines               []string
	starts              []int // rune offset of each line
	longest             int   // runes in the longest line
}

func (a *TextArea) geometry(c *core.Ctx, r core.Rect) areaGeometry {
	g := areaGeometry{lineH: c.LineH(c.F.Mono) + 4, charW: c.Width(c.F.Mono, "0")}
	g.lines = strings.Split(a.Text, "\n")
	off := 0
	for _, l := range g.lines {
		n := len([]rune(l))
		g.starts = append(g.starts, off)
		g.longest = max(g.longest, n)
		off += n + 1
	}
	digits := len(strconv.Itoa(len(g.lines)))
	g.textX = r.X + core.Gutter + digits*g.charW + core.Gutter
	return g
}

func (g areaGeometry) contentW(r core.Rect) int {
	return g.textX - r.X + g.longest*g.charW + core.Gutter
}
func (g areaGeometry) contentH() int { return len(g.lines) * g.lineH }

// posAtPixel maps a pointer position to a rune position.
func (a *TextArea) posAtPixel(r core.Rect, g areaGeometry, x, y int) int {
	line := min(max((y-r.Y+a.ScrollY)/g.lineH, 0), len(g.lines)-1)
	col := max((x-g.textX+a.ScrollX+g.charW/2)/g.charW, 0)
	return a.PosAt(line, col)
}

// Update handles the pointer and, while focused, the keyboard inside r.
// It reports whether the text changed.
func (a *TextArea) Update(c *core.Ctx, r core.Rect) (changed bool) {
	g := a.geometry(c, r)
	b := layoutBars(r, g.contentW(r), g.contentH())
	a.ViewLines = max(b.viewH/g.lineH, 1)
	a.drag.update(c, b, g.contentW(core.Rect{W: b.viewW}), g.contentH(), &a.ScrollX, &a.ScrollY)

	if !a.drag.active() {
		a.updatePointer(c, r, g, b)
	}
	if a.Focused {
		changed = a.updateKeys(c, r, g, b)
	}
	a.ScrollY = ClampScroll(a.ScrollY, g.contentH(), b.viewH)
	a.ScrollX = ClampScroll(a.ScrollX, g.contentW(core.Rect{W: b.viewW}), b.viewW)
	return changed
}

func (a *TextArea) updatePointer(c *core.Ctx, r core.Rect, g areaGeometry, b bars) {
	in := c.In
	over := in.Over(r)
	if in.Click {
		a.Focused = over
		inBar := b.vTrack.Contains(in.X, in.Y) || b.hTrack.Contains(in.X, in.Y)
		if over && !inBar {
			a.PressN(a.posAtPixel(r, g, in.X, in.Y), in.Shift, max(in.Clicks, 1))
			a.revealed = a.Cursor
		}
	}
	if a.Dragging {
		if in.Held {
			a.Drag(a.posAtPixel(r, g, in.X, in.Y))
		} else {
			a.Dragging = false
		}
	}
	if over && in.Wheel != 0 {
		if in.Shift {
			a.ScrollX = ClampScroll(a.ScrollX+int(-in.Wheel*float64(g.charW)*6), g.contentW(core.Rect{W: b.viewW}), b.viewW)
		} else {
			a.ScrollY = ClampScroll(a.ScrollY+int(-in.Wheel*float64(g.lineH)*3), g.contentH(), b.viewH)
		}
		c.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
}

// updateKeys applies the keyboard to the text and scrolls the cursor into view.
func (a *TextArea) updateKeys(c *core.Ctx, r core.Rect, g areaGeometry, b bars) (changed bool) {
	in := c.In
	if in.Escape {
		a.Focused = false
		return false
	}
	changed = a.RunShortcuts(c)
	typed := in.Typed
	if in.Ctrl {
		typed = nil // Ctrl+letter is a shortcut, not text
	}
	edits := append([]core.Edit(nil), in.Edits...)
	if in.Enter {
		edits = append(edits, core.EditNewline)
	}
	if in.Tab {
		edits = append(edits, map[bool]core.Edit{false: core.EditIndent, true: core.EditDedent}[in.Shift])
	}
	changed = a.Apply(typed, edits, in.Shift) || changed
	if changed || a.Cursor != a.revealed {
		a.reveal(g, b)
		a.revealed = a.Cursor
	}
	return changed
}

// reveal scrolls so the cursor line and column are inside the view.
func (a *TextArea) reveal(g areaGeometry, b bars) {
	line, col := a.LineCol(a.Cursor)
	top, bottom := line*g.lineH, (line+1)*g.lineH
	if top < a.ScrollY {
		a.ScrollY = top
	} else if bottom > a.ScrollY+b.viewH {
		a.ScrollY = bottom - b.viewH
	}
	left, right := col*g.charW, (col+1)*g.charW
	textW := b.viewW - (g.textX - 0)
	if left < a.ScrollX {
		a.ScrollX = left
	} else if right > a.ScrollX+textW {
		a.ScrollX = right - textW
	}
}

// Draw paints the editor inside r; the caller fills the background.
// Line numbers stay put while the text scrolls sideways.
func (a *TextArea) Draw(c *core.Ctx, dst *ebiten.Image, r core.Rect) {
	g := a.geometry(c, r)
	b := layoutBars(r, g.contentW(r), g.contentH())
	view := core.Rect{X: r.X, Y: r.Y, W: b.viewW, H: b.viewH}
	body := core.Rect{X: g.textX - 4, Y: r.Y, W: b.viewW - (g.textX - 4 - r.X), H: b.viewH}
	area, text := c.Clip(dst, view), c.Clip(dst, body)
	curLine, _ := a.LineCol(a.Cursor)
	lo, hi, hasSel := a.Selection()

	for i := a.ScrollY / g.lineH; i < len(g.lines); i++ {
		y := r.Y + i*g.lineH - a.ScrollY
		if y >= r.Y+b.viewH {
			break
		}
		cell := core.Rect{X: r.X, Y: y, W: b.viewW, H: g.lineH}
		if a.Focused && !hasSel && i == curLine {
			c.Fill(area, cell, c.Color(core.Hover))
		}
		line := g.lines[i]
		if hasSel {
			a.drawSelection(c, area, g, i, y, lo, hi)
		}
		x := g.textX - a.ScrollX
		spans := []Span{{Text: line, Style: core.Text}}
		if a.Styler != nil {
			spans = a.Styler(line)
		}
		for _, sp := range spans {
			c.TextMid(text, c.F.Mono, sp.Text, x, cell, c.Color(sp.Style))
			x += c.Width(c.F.Mono, sp.Text)
		}
		numClr := c.Color(core.Muted)
		if i == curLine {
			numClr = c.Color(core.Text)
		}
		num := strconv.Itoa(i + 1)
		c.TextMid(area, c.F.Mono, num, g.textX-core.Gutter-len(num)*g.charW, cell, numClr)
	}
	if a.Focused && (c.Tick/30)%2 == 0 {
		line, col := a.LineCol(a.Cursor)
		cx, cy := g.textX-a.ScrollX+col*g.charW, r.Y+line*g.lineH-a.ScrollY
		c.VLine(text, cx, cy+2, g.lineH-4, c.Color(core.Text))
	}
	drawBars(c, dst, b, a.drag, g.contentW(core.Rect{W: b.viewW}), g.contentH(), a.ScrollX, a.ScrollY)
	if a.Focused {
		c.Outline(dst, r, c.Color(core.AccentLine))
	}
}

// drawSelection highlights the part of line i inside the selection [lo, hi).
func (a *TextArea) drawSelection(c *core.Ctx, dst *ebiten.Image, g areaGeometry, i, y, lo, hi int) {
	start, end := g.starts[i], g.starts[i]+len([]rune(g.lines[i]))
	if hi <= start || lo > end {
		return
	}
	from, to := max(lo, start)-start, min(hi, end+1)-start // +1: the newline counts
	x0 := g.textX - a.ScrollX + from*g.charW
	x1 := g.textX - a.ScrollX + to*g.charW
	c.Fill(dst, core.Rect{X: x0, Y: y, W: max(x1-x0, 0), H: g.lineH}, c.Color(core.Selection))
}
