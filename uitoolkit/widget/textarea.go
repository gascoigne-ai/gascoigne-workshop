package widget

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// TextArea is a multi-line editor: a multi-line TextField plus the
// view state (vertical scroll, optional syntax styling). Selection,
// undo, word movement and the clipboard come from the TextField.
type TextArea struct {
	TextField
	ScrollY int

	// Styler colors a line; nil draws plain text.
	Styler func(line string) []Span

	// Wrap breaks long lines before a word at the width of the editor
	// instead of scrolling sideways; Up and Down then move by visual row.
	// Area's Wrap option sets it every frame.
	Wrap bool

	drag           barDrag
	caretX, caretY int // where the caret is on screen, as of the last Update
	caretKnown     bool
	revealed       int  // cursor position last scrolled into view
	laidWrap       bool // the Wrap the scroll was last laid out with
}

// NewTextArea returns an editor holding text.
func NewTextArea(text string) TextArea {
	a := TextArea{TextField: NewTextField(text)}
	a.Multiline = true
	a.revealed = -1
	return a
}

// areaGeometry is the pixel layout of an area in r.
type areaGeometry struct {
	lineH, charW, textX int
	adv                 float64 // the exact advance of one character
	lines               []string
	starts              []int // rune offset of each line
	longest             int   // runes in the longest line
	wrapped             bool
	layout              wrapLayout // the visual rows of the lines
}

func (a *TextArea) geometry(c *core.Ctx, r core.Rect) areaGeometry {
	g := areaGeometry{lineH: c.LineH(c.F.Mono) + 4, charW: c.Width(c.F.Mono, "0"), adv: c.Advance(c.F.Mono)}
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
	g.wrapped = a.Wrap
	g.layout = a.layout(g, r)
	return g
}

// layout lays the lines out in visual rows, wrapped at the editor's width
// when Wrap is on. Every rune is one advance wide, as drawing places text
// by column.
func (a *TextArea) layout(g areaGeometry, r core.Rect) wrapLayout { return areaLayout(a.Wrap, g, r) }

func areaLayout(wrap bool, g areaGeometry, r core.Rect) wrapLayout {
	lines := make([]wrapLine, len(g.lines))
	for i, text := range g.lines {
		lines[i] = wrapLine{line: i, text: []rune(text)}
	}
	return layoutWrap(lines, areaWrapWidth(wrap, g, r), func(rune) float64 { return g.adv })
}

// areaWrapWidth is the room in pixels for the text of a row, 0 for an
// editor that does not wrap. The vertical bar is assumed present, so the
// rows do not change when wrapping makes it appear; at least one pixel,
// since 0 would mean no wrap.
func areaWrapWidth(wrap bool, g areaGeometry, r core.Rect) float64 {
	if !wrap {
		return 0
	}
	return float64(max(r.W-scrollbarW-(g.textX-r.X)-core.Gutter, 1))
}

// contentW is the width of the widest line plus the gutter; wrapped text
// fits the width, so it has none to scroll.
func (g areaGeometry) contentW(r core.Rect) int {
	if g.wrapped {
		return 0
	}
	return g.textX - r.X + g.colX(g.longest) + core.Gutter
}

// colX is the pixel offset of the start of column col from the start of the
// text, by the exact advance.
func (g areaGeometry) colX(col int) int { return int(float64(col)*g.adv + 0.5) }

// colAt is the column whose cell contains offset x from the start of the
// text (the nearest boundary), never negative.
func (g areaGeometry) colAt(x int) int { return max(int((float64(x)+g.adv/2)/g.adv), 0) }

func (g areaGeometry) contentH() int { return len(g.layout.rows) * g.lineH }

// caretRow is the visual row of the cursor and its column offset from the
// row's start.
func (a *TextArea) caretRow(g areaGeometry) (row, offset int) {
	line, col := a.LineCol(a.Cursor)
	row = g.layout.rowOf(textPos{line, col})
	return row, col - g.layout.rows[row].from
}

// posAtPixel maps a pointer position to a rune position.
func (a *TextArea) posAtPixel(r core.Rect, g areaGeometry, x, y int) int {
	row := min(max((y-r.Y+a.ScrollY)/g.lineH, 0), len(g.layout.rows)-1)
	pos := g.layout.posAt(row, float64(x-g.textX+a.ScrollX))
	return a.PosAt(pos.line, pos.col)
}

// keepTopLine keeps the line at the top of the editor in place when Wrap
// has changed since the last layout, as the number of rows above it does.
func (a *TextArea) keepTopLine(g areaGeometry, r core.Rect) {
	if a.Wrap == a.laidWrap {
		return
	}
	top := areaLayout(a.laidWrap, g, r).rowStart(a.ScrollY / g.lineH)
	a.laidWrap = a.Wrap
	if row := g.layout.rowOf(top); row >= 0 {
		a.ScrollY = row * g.lineH
	}
}

// Update handles the pointer and, while focused, the keyboard inside r.
// It reports whether the text changed.
func (a *TextArea) Update(c *core.Ctx, r core.Rect) (changed bool) {
	g := a.geometry(c, r)
	a.keepTopLine(g, r)
	b := layoutBars(r, g.contentW(r), g.contentH())
	a.ViewLines = max(b.viewH/g.lineH, 1)
	a.drag.update(c, b, g.contentW(core.Rect{W: b.viewW}), g.contentH(), &a.ScrollX, &a.ScrollY)

	if !a.drag.active() {
		a.updatePointer(c, r, g, b)
	}
	if a.Focused {
		changed = a.updateKeys(c, r, g, b)
	}
	if changed { // the rows follow the text
		g = a.geometry(c, r)
	}
	a.ScrollY = clampScroll(a.ScrollY, g.contentH(), b.viewH)
	row, offset := a.caretRow(g)
	a.caretX, a.caretY, a.caretKnown = g.textX-a.ScrollX+g.colX(offset), r.Y+(row+1)*g.lineH-a.ScrollY, true
	a.ScrollX = clampScroll(a.ScrollX, g.contentW(core.Rect{W: b.viewW}), b.viewW)
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
			a.ScrollX = clampScroll(a.ScrollX+int(-in.Wheel*float64(g.charW)*6), g.contentW(core.Rect{W: b.viewW}), b.viewW)
		} else {
			a.ScrollY = clampScroll(a.ScrollY+int(-in.Wheel*float64(g.lineH)*3), g.contentH(), b.viewH)
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
	changed = a.applyEdits(c, r, typed, edits) || changed
	if changed || a.Cursor != a.revealed {
		a.reveal(r, a.geometry(c, r), b) // the rows of the text as it is now
		a.revealed = a.Cursor
	}
	return changed
}

// rowMoves are the keys that a wrapped editor moves by visual row, as an
// editor does; Home and End stay on the line.
var rowMoves = map[core.Edit]int{core.EditUp: -1, core.EditDown: 1}

// applyEdits is Apply with Shift extending, except for rowMoves in a
// wrapped editor. Each one lays the text out as it is at that point, since
// an edit before it in the same frame may have changed the rows.
func (a *TextArea) applyEdits(c *core.Ctx, r core.Rect, typed []rune, edits []core.Edit) bool {
	extend := c.In.Shift
	if !a.Wrap {
		return a.Apply(typed, edits, extend)
	}
	changed := a.Apply(typed, nil, extend)
	for _, edit := range edits {
		if dy, ok := rowMoves[edit]; ok {
			a.moveRows(dy, extend, a.geometry(c, r))
			continue
		}
		changed = a.Apply(nil, []core.Edit{edit}, extend) || changed
	}
	return changed
}

// moveRows moves the cursor dy visual rows, keeping the goal column within
// the row. Moving past the first or last row goes to the start or end of
// the text, as moving by lines does.
func (a *TextArea) moveRows(dy int, extend bool, g areaGeometry) {
	row, offset := a.caretRow(g)
	if a.goal > 0 {
		offset = a.goal - 1
	}
	target := row + dy
	switch {
	case target < 0:
		a.moveTo(0, extend)
	case target >= len(g.layout.rows):
		a.moveTo(len([]rune(a.Text)), extend)
	default:
		pos := g.layout.posAt(target, float64(g.colX(offset)))
		a.moveTo(a.PosAt(pos.line, pos.col), extend)
	}
	a.goal = offset + 1 // after moveTo, which clears it
}

// reveal scrolls so the cursor's row and column are inside the view.
func (a *TextArea) reveal(r core.Rect, g areaGeometry, b bars) {
	row, col := a.caretRow(g)
	top, bottom := row*g.lineH, (row+1)*g.lineH
	if top < a.ScrollY {
		a.ScrollY = top
	} else if bottom > a.ScrollY+b.viewH {
		a.ScrollY = bottom - b.viewH
	}
	left, right := g.colX(col), g.colX(col+1)
	textW := b.viewW - (g.textX - r.X) // the text area, from the editor's own left edge
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

	var spans []Span // the styled spans of the line the rows are on
	for rowIdx := a.ScrollY / g.lineH; rowIdx < len(g.layout.rows); rowIdx++ {
		y := r.Y + rowIdx*g.lineH - a.ScrollY
		if y >= r.Y+b.viewH {
			break
		}
		row := g.layout.rows[rowIdx]
		cell := core.Rect{X: r.X, Y: y, W: b.viewW, H: g.lineH}
		if a.Focused && !hasSel && row.line == curLine {
			c.Fill(area, cell, c.Color(core.Hover))
		}
		if hasSel {
			a.drawSelection(c, area, g, row, y, lo, hi)
		}
		if row.from == 0 || spans == nil { // a line's rows share its styling
			spans = a.lineSpans(g.lines[row.line])
		}
		a.drawRow(c, text, g, row, cell, spans)
		if row.from > 0 { // a continuation row has no number
			continue
		}
		numClr := c.Color(core.Muted)
		if row.line == curLine {
			numClr = c.Color(core.Text)
		}
		num := strconv.Itoa(row.line + 1)
		c.TextMid(area, c.F.Mono, num, g.textX-core.Gutter-len(num)*g.charW, cell, numClr)
	}
	if a.Focused && c.CaretVisible() {
		row, offset := a.caretRow(g)
		cx, cy := g.textX-a.ScrollX+g.colX(offset), r.Y+row*g.lineH-a.ScrollY
		cell := core.Rect{X: cx, Y: cy, W: g.colX(offset+1) - g.colX(offset), H: g.lineH}
		drawBlockCaret(c, text, c.F.Mono, caretGlyph([]rune(a.Text), a.Cursor), cx, cell)
	}
	drawBars(c, dst, b, a.drag, g.contentW(core.Rect{W: b.viewW}), g.contentH(), a.ScrollX, a.ScrollY)
	if a.Focused {
		c.Outline(dst, r, c.Color(core.AccentLine))
	}
}

// lineSpans are the spans a line is drawn with: the Styler's, else plain.
func (a *TextArea) lineSpans(line string) []Span {
	if a.Styler == nil {
		return []Span{{Text: line, Style: core.Text}}
	}
	return a.Styler(line)
}

// drawRow draws the part of its line's spans that a row shows.
func (a *TextArea) drawRow(c *core.Ctx, dst *ebiten.Image, g areaGeometry, row visualRow, cell core.Rect, spans []Span) {
	if row.from > 0 || row.to < len([]rune(g.lines[row.line])) {
		spans = cutSpans(spans, row.from, row.to)
	}
	col := 0 // spans are placed by column, so rounding does not add up along the line
	for _, sp := range spans {
		c.TextMid(dst, c.F.Mono, sp.Text, g.textX-a.ScrollX+g.colX(col), cell, c.Color(sp.Style))
		col += len([]rune(sp.Text))
	}
}

// drawSelection highlights the part of a row inside the selection [lo, hi);
// the last row of a line also fills one character for its line break.
func (a *TextArea) drawSelection(c *core.Ctx, dst *ebiten.Image, g areaGeometry, row visualRow, y, lo, hi int) {
	start := g.starts[row.line] + row.from
	end := g.starts[row.line] + row.to
	if row.to == len([]rune(g.lines[row.line])) {
		end++ // the newline counts
	}
	if hi <= start || lo >= end {
		return
	}
	from, to := max(lo, start)-start, min(hi, end)-start
	x0 := g.textX - a.ScrollX + g.colX(from)
	x1 := g.textX - a.ScrollX + g.colX(to)
	c.Fill(dst, core.Rect{X: x0, Y: y, W: max(x1-x0, 0), H: g.lineH}, c.Color(core.Selection))
}

// CaretAt is where the caret is on screen as of the last Update: the left
// edge of its cell and the bottom of its line, which is where a popup for the
// text opens. ok is false before the editor was first updated.
func (a *TextArea) CaretAt() (x, y int, ok bool) { return a.caretX, a.caretY, a.caretKnown }

func (a *TextArea) HasFocus() bool   { return a.Focused }
func (a *TextArea) SetFocus(on bool) { a.Focused = on }
