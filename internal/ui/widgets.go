package ui

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

const tabGap = 10

// tabItem is one entry of a pane tab strip; Badge > 0 shows a count.
type tabItem struct {
	Label string
	Badge int
}

func (c *Ctx) tabWidth(it tabItem) int {
	w := c.width(c.F.UI, it.Label)
	if it.Badge > 0 {
		w += 4 + c.width(c.F.Small, strconv.Itoa(it.Badge))
	}
	return w
}

// tabRects places the tabs of items left to right inside r.
func (c *Ctx) tabRects(r Rect, items []tabItem) []Rect {
	rects := make([]Rect, len(items))
	x := r.X + c.Th.Pad
	for i, it := range items {
		w := c.tabWidth(it) + 2*tabGap
		rects[i] = Rect{x, r.Y, w, r.H}
		x += w
	}
	return rects
}

// hitTab returns the index of the clicked tab or -1.
func (c *Ctx) hitTab(r Rect, items []tabItem) int {
	for i, tr := range c.tabRects(r, items) {
		if c.In.Hit(tr) {
			return i
		}
	}
	return -1
}

func (c *Ctx) drawTabs(dst *ebiten.Image, r Rect, items []tabItem, active int) {
	for i, tr := range c.tabRects(r, items) {
		clr := c.Th.Muted
		if i == active {
			clr = c.Th.Text
			c.fill(dst, Rect{tr.X + tabGap, tr.Y + tr.H - 2, tr.W - 2*tabGap, 2}, c.Th.Accent)
		}
		x := tr.X + tabGap
		c.textMid(dst, c.F.UI, items[i].Label, x, tr, clr)
		if b := items[i].Badge; b > 0 {
			bx := x + c.width(c.F.UI, items[i].Label) + 4
			c.text(dst, c.F.Small, strconv.Itoa(b), bx, tr.Y+tr.H/2-c.lineH(c.F.Small)+2, c.Th.Muted)
		}
	}
	c.hline(dst, r.X, r.Y+r.H-1, r.W, c.Th.Border)
}

// Table geometry: a checkbox column, name/value(/description) columns
// and a delete column.
const (
	checkW = 36
	trashW = 36
)

// tableCols splits a row into its cells.
func tableCols(row Rect, desc bool) (check, name, value, descr, trash Rect) {
	check, rest := row.SplitLeft(checkW)
	rest, trash = rest.SplitRight(trashW)
	if desc {
		name, rest = rest.SplitLeft(rest.W * 28 / 100)
		value, descr = rest.SplitLeft(rest.W * 55 / 100)
		return
	}
	name, value = rest.SplitLeft(rest.W * 35 / 100)
	return
}

// tableRow is the rect of data row i; the header sits above row 0.
func (c *Ctx) tableRow(r Rect, i int) Rect {
	return Rect{r.X, r.Y + (i+1)*c.Th.RowH, r.W, c.Th.RowH}
}

type tableAction int

const (
	tableNone tableAction = iota
	tableToggle
	tableDelete
)

// hitTable reports which existing row's checkbox or delete button was clicked.
func (c *Ctx) hitTable(r Rect, n int, desc bool) (int, tableAction) {
	for i := range n {
		check, _, _, _, trash := tableCols(c.tableRow(r, i), desc)
		switch {
		case c.In.Hit(check):
			return i, tableToggle
		case c.In.Hit(trash):
			return i, tableDelete
		}
	}
	return -1, tableNone
}

// drawTable draws rows as zebra stripes under an uppercase header with
// a rule. editable adds the checkbox and delete columns and the empty
// placeholder row.
func (c *Ctx) drawTable(dst *ebiten.Image, r Rect, rows []model.KV, desc, editable bool) {
	th := c.Th
	n := len(rows)
	if editable {
		n++
	}
	c.fill(dst, Rect{r.X, r.Y, r.W, (n + 1) * th.RowH}, th.Sunken)

	header := Rect{r.X, r.Y, r.W, th.RowH}
	_, name, value, descr, _ := tableCols(header, desc)
	c.textMid(dst, c.F.Small, "NAME", name.X+8, header, th.Muted)
	c.textMid(dst, c.F.Small, "VALUE", value.X+8, header, th.Muted)
	if desc {
		c.textMid(dst, c.F.Small, "DESCRIPTION", descr.X+8, header, th.Muted)
	}
	c.hline(dst, r.X, r.Y+th.RowH-1, r.W, th.Rule)

	for i := range n {
		row := c.tableRow(r, i)
		if i%2 == 1 {
			c.fill(dst, row, th.Alt)
		}
		check, name, value, descr, trash := tableCols(row, desc)
		if i == len(rows) {
			c.cells(dst, "Name", "Value", "Description", name, value, descr, desc, row, th.Muted)
			continue
		}
		kv := rows[i]
		clr := th.Text
		if !kv.Enabled {
			clr = th.Muted
		}
		if editable {
			c.checkbox(dst, check, kv.Enabled)
			c.cross(dst, float32(trash.X+trash.W/2), float32(trash.Y+trash.H/2), 4, th.Muted)
		}
		c.cells(dst, kv.Name, kv.Value, kv.Description, name, value, descr, desc, row, clr)
	}
}

func (c *Ctx) cells(dst *ebiten.Image, nameS, valueS, descS string, name, value, descr Rect, desc bool, row Rect, clr color.RGBA) {
	c.cell(dst, nameS, name, row, clr)
	c.cell(dst, valueS, value, row, clr)
	if desc {
		c.cell(dst, descS, descr, row, clr)
	}
}

func (c *Ctx) cell(dst *ebiten.Image, s string, col, row Rect, clr color.RGBA) {
	c.textMid(dst, c.F.UI, c.ellipsize(c.F.UI, s, col.W-16), col.X+8, row, clr)
}

// checkbox is a square gold outline, with a filled core when on.
func (c *Ctx) checkbox(dst *ebiten.Image, cell Rect, on bool) {
	const size = 14
	box := Rect{cell.X + (cell.W-size)/2, cell.Y + (cell.H-size)/2, size, size}
	c.outline(dst, box, c.Th.Accent)
	c.outline(dst, box.Inset(1), c.Th.Accent)
	if on {
		c.fill(dst, box.Inset(4), c.Th.Accent)
	}
}

func (c *Ctx) button(dst *ebiten.Image, r Rect, label string, bg, fg color.RGBA) {
	if c.In.Over(r) {
		bg = lighten(bg)
	}
	c.fill(dst, r, bg)
	c.textCenter(dst, c.F.UI, label, r, fg)
}
