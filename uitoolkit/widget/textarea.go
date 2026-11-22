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

	// Styler colors one line at a time (its text is a single line, never
	// the whole content); nil draws plain text.
	Styler Styler

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
	monoGrid
	lines   []string
	starts  []int // rune offset of each line
	longest int   // display columns of the longest line
	wrapped bool
	layout  wrapLayout // the visual rows of the lines
}

func (a *TextArea) geometry(ctx *core.Ctx, rect core.Rect) areaGeometry {
	g := areaGeometry{monoGrid: newMonoGrid(ctx)}
	g.lines = strings.Split(a.Text, "\n")
	off := 0
	for _, l := range g.lines {
		runes := []rune(l)
		n := len(runes)
		g.starts = append(g.starts, off)
		g.longest = max(g.longest, core.TextColumn(runes, n))
		off += n + 1
	}
	digits := len(strconv.Itoa(len(g.lines)))
	g.textX = rect.X + core.Gutter + digits*g.charW + core.Gutter
	g.wrapped = a.Wrap
	g.layout = a.layout(g, rect)
	return g
}

// layout lays the lines out in visual rows, wrapped at the editor's width
// when Wrap is on. Every rune is one advance wide and a tab reaches its
// stop, as drawing places text by column.
func (a *TextArea) layout(geometry areaGeometry, rect core.Rect) wrapLayout {
	return areaLayout(a.Wrap, geometry, rect)
}

func areaLayout(wrap bool, geometry areaGeometry, rect core.Rect) wrapLayout {
	lines := make([]wrapLine, len(geometry.lines))
	for i, text := range geometry.lines {
		lines[i] = wrapLine{line: i, text: []rune(text)}
	}
	return layoutWrap(lines, geometry.wrapWidth(wrap, rect), monoColumns(geometry.adv))
}

// contentW is the width of the widest line plus the gutter; wrapped text
// fits the width, so it has none to scroll.
func (g areaGeometry) contentW(rect core.Rect) int {
	if g.wrapped {
		return 0
	}
	return g.textX - rect.X + g.colX(g.longest) + core.Gutter
}

// colAt is monoGrid.colAt, never negative.
func (g areaGeometry) colAt(posX int) int { return max(g.monoGrid.colAt(posX), 0) }

func (g areaGeometry) contentH() int { return len(g.layout.rows) * g.lineH }

// caretRow is the visual row of the cursor and its display column from the
// row's start.
func (a *TextArea) caretRow(geometry areaGeometry) (row, offset int) {
	line, col := a.LineCol(a.Cursor)
	return geometry.layout.rowCol(textPos{line, col})
}

// posAtPixel maps a pointer position to a rune position.
func (a *TextArea) posAtPixel(rect core.Rect, geometry areaGeometry, posX, posY int) int {
	row := min(max((posY-rect.Y+a.ScrollY)/geometry.lineH, 0), len(geometry.layout.rows)-1)
	pos := geometry.layout.posAt(row, float64(posX-geometry.textX+a.ScrollX))
	return a.PosAt(pos.line, pos.col)
}

// keepTopLine keeps the line at the top of the editor in place when Wrap
// has changed since the last layout, as the number of rows above it does.
func (a *TextArea) keepTopLine(geometry areaGeometry, rect core.Rect) {
	if a.Wrap == a.laidWrap {
		return
	}
	before := areaLayout(a.laidWrap, geometry, rect)
	a.laidWrap = a.Wrap
	a.ScrollY = geometry.keepTop(a.ScrollY, before, geometry.layout)
}

// Update handles the pointer and, while focused, the keyboard inside r.
// It reports whether the text changed.
func (a *TextArea) Update(ctx *core.Ctx, rect core.Rect) (changed bool) {
	g := a.geometry(ctx, rect)
	a.keepTopLine(g, rect)
	b := layoutBars(rect, g.contentW(rect), g.contentH())
	a.ViewLines = max(b.viewH/g.lineH, 1)
	a.drag.update(ctx, b, g.contentW(core.Rect{W: b.viewW}), g.contentH(), &a.ScrollX, &a.ScrollY)

	if !a.drag.active() {
		a.updatePointer(ctx, rect, g, b)
	}
	if a.Focused {
		changed = a.updateKeys(ctx, rect, g, b)
	}
	if changed { // the rows follow the text
		g = a.geometry(ctx, rect)
	}
	a.ScrollY = clampScroll(a.ScrollY, g.contentH(), b.viewH)
	row, offset := a.caretRow(g)
	a.caretX, a.caretY, a.caretKnown = g.textX-a.ScrollX+g.colX(offset), rect.Y+(row+1)*g.lineH-a.ScrollY, true
	a.ScrollX = clampScroll(a.ScrollX, g.contentW(core.Rect{W: b.viewW}), b.viewW)
	return changed
}

func (a *TextArea) updatePointer(ctx *core.Ctx, rect core.Rect, geometry areaGeometry, sizes bars) {
	in := ctx.In
	over := in.Over(rect)
	if in.Click {
		a.Focused = over
		inBar := sizes.vTrack.Contains(in.X, in.Y) || sizes.hTrack.Contains(in.X, in.Y)
		if over && !inBar {
			a.PressN(a.posAtPixel(rect, geometry, in.X, in.Y), in.Shift, max(in.Clicks, 1))
			a.revealed = a.Cursor
		}
	}
	if a.Dragging {
		if in.Held {
			a.Drag(a.posAtPixel(rect, geometry, in.X, in.Y))
		} else {
			a.Dragging = false
		}
	}
	if over && in.Wheel != 0 {
		if in.Shift {
			a.ScrollX = clampScroll(a.ScrollX+geometry.wheelDX(in.Wheel), geometry.contentW(core.Rect{W: sizes.viewW}), sizes.viewW)
		} else {
			a.ScrollY = clampScroll(a.ScrollY+geometry.wheelDY(in.Wheel), geometry.contentH(), sizes.viewH)
		}
		ctx.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
}

// updateKeys applies the keyboard to the text and scrolls the cursor into view.
func (a *TextArea) updateKeys(ctx *core.Ctx, rect core.Rect, geometry areaGeometry, sizes bars) (changed bool) {
	in := ctx.In
	if in.Escape {
		a.Focused = false
		return false
	}
	changed = a.RunShortcuts(ctx)
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
	changed = a.applyEdits(ctx, rect, typed, edits) || changed
	if changed || a.Cursor != a.revealed {
		a.reveal(rect, a.geometry(ctx, rect), sizes) // the rows of the text as it is now
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
func (a *TextArea) applyEdits(ctx *core.Ctx, rect core.Rect, typed []rune, edits []core.Edit) bool {
	extend := ctx.In.Shift
	if !a.Wrap {
		return a.Apply(typed, edits, extend)
	}
	changed := a.Apply(typed, nil, extend)
	for _, edit := range edits {
		if dy, ok := rowMoves[edit]; ok {
			a.moveRows(dy, extend, a.geometry(ctx, rect))
			continue
		}
		changed = a.Apply(nil, []core.Edit{edit}, extend) || changed
	}
	return changed
}

// moveRows moves the cursor dy visual rows, keeping the goal column within
// the row. Moving past the first or last row goes to the start or end of
// the text, as moving by lines does.
func (a *TextArea) moveRows(dy int, extend bool, geometry areaGeometry) {
	row, offset := a.caretRow(geometry)
	if a.goal > 0 {
		offset = a.goal - 1
	}
	target := row + dy
	switch {
	case target < 0:
		a.moveTo(0, extend)
	case target >= len(geometry.layout.rows):
		a.moveTo(len([]rune(a.Text)), extend)
	default:
		pos := geometry.layout.posAt(target, float64(geometry.colX(offset)))
		a.moveTo(a.PosAt(pos.line, pos.col), extend)
	}
	a.goal = offset + 1 // after moveTo, which clears it
}

// reveal scrolls so the cursor's row and column are inside the view.
func (a *TextArea) reveal(rect core.Rect, geometry areaGeometry, sizes bars) {
	row, col := a.caretRow(geometry)
	top, bottom := row*geometry.lineH, (row+1)*geometry.lineH
	if top < a.ScrollY {
		a.ScrollY = top
	} else if bottom > a.ScrollY+sizes.viewH {
		a.ScrollY = bottom - sizes.viewH
	}
	left, right := geometry.colX(col), geometry.colX(col+1)
	textW := sizes.viewW - (geometry.textX - rect.X) // the text area, from the editor's own left edge
	if left < a.ScrollX {
		a.ScrollX = left
	} else if right > a.ScrollX+textW {
		a.ScrollX = right - textW
	}
}

// Draw paints the editor inside r; the caller fills the background.
// Line numbers stay put while the text scrolls sideways.
func (a *TextArea) Draw(ctx *core.Ctx, dst *ebiten.Image, rect core.Rect) {
	g := a.geometry(ctx, rect)
	b := layoutBars(rect, g.contentW(rect), g.contentH())
	view := core.Rect{X: rect.X, Y: rect.Y, W: b.viewW, H: b.viewH}
	body := core.Rect{X: g.textX - 4, Y: rect.Y, W: b.viewW - (g.textX - 4 - rect.X), H: b.viewH}
	area, text := ctx.Clip(dst, view), ctx.Clip(dst, body)
	curLine, _ := a.LineCol(a.Cursor)
	lo, hi, hasSel := a.Selection()

	var spans []Span // the styled spans of the line the rows are on
	for rowIdx := a.ScrollY / g.lineH; rowIdx < len(g.layout.rows); rowIdx++ {
		y := rect.Y + rowIdx*g.lineH - a.ScrollY
		if y >= rect.Y+b.viewH {
			break
		}
		row := g.layout.rows[rowIdx]
		cell := core.Rect{X: rect.X, Y: y, W: b.viewW, H: g.lineH}
		if a.Focused && !hasSel && row.line == curLine {
			ctx.Fill(area, cell, ctx.Color(core.Hover))
		}
		if hasSel {
			a.drawSelection(ctx, area, g, row, y, lo, hi)
		}
		if row.from == 0 || spans == nil { // a line's rows share its styling
			spans = a.lineSpans(g.lines[row.line])
		}
		a.drawRow(ctx, text, g, row, cell, spans)
		if row.from > 0 { // a continuation row has no number
			continue
		}
		numClr := ctx.Color(core.Muted)
		if row.line == curLine {
			numClr = ctx.Color(core.Text)
		}
		num := strconv.Itoa(row.line + 1)
		ctx.TextMid(area, ctx.F.Mono, num, g.textX-core.Gutter-len(num)*g.charW, cell, numClr)
	}
	if a.Focused && ctx.CaretVisible() {
		row, offset := a.caretRow(g)
		cx, cy := g.textX-a.ScrollX+g.colX(offset), rect.Y+row*g.lineH-a.ScrollY
		cell := core.Rect{X: cx, Y: cy, W: g.colX(offset+1) - g.colX(offset), H: g.lineH}
		drawBlockCaret(ctx, text, ctx.F.Mono, caretGlyph([]rune(a.Text), a.Cursor), cx, cell)
	}
	drawBars(ctx, dst, b, a.drag, g.contentW(core.Rect{W: b.viewW}), g.contentH(), a.ScrollX, a.ScrollY)
	if a.Focused {
		ctx.Outline(dst, rect, ctx.Color(core.AccentLine))
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
func (a *TextArea) drawRow(ctx *core.Ctx, dst *ebiten.Image, geometry areaGeometry, row visualRow, cell core.Rect, spans []Span) {
	if row.from > 0 || row.to < len([]rune(geometry.lines[row.line])) {
		spans = cutSpans(spans, row.from, row.to)
	}
	geometry.drawSpans(ctx, dst, a.ScrollX, cell, row, spans)
}

// drawSelection highlights the part of a row inside the selection [lo, hi);
// the last row of a line also fills one character for its line break.
func (a *TextArea) drawSelection(ctx *core.Ctx, dst *ebiten.Image, geometry areaGeometry, row visualRow, posY, lo, hi int) {
	start := geometry.starts[row.line] + row.from
	end := geometry.starts[row.line] + row.to
	if row.to == len([]rune(geometry.lines[row.line])) {
		end++ // the newline counts
	}
	if hi <= start || lo >= end {
		return
	}
	from := geometry.layout.colIn(row, row.from+max(lo, start)-start)
	to := geometry.layout.colIn(row, row.from+min(hi, end)-start)
	ctx.Fill(dst, geometry.selectionRect(a.ScrollX, from, to, posY, geometry.lineH), ctx.Color(core.Selection))
}

// CaretAt is where the caret is on screen as of the last Update: the left
// edge of its cell and the bottom of its line, which is where a popup for the
// text opens. ok is false before the editor was first updated.
func (a *TextArea) CaretAt() (x, y int, ok bool) { return a.caretX, a.caretY, a.caretKnown }

// HasFocus reports whether the editor has the keyboard focus.
func (a *TextArea) HasFocus() bool { return a.Focused }

// SetFocus gives or takes the keyboard focus.
func (a *TextArea) SetFocus(on bool) { a.Focused = on }
