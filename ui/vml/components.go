package vml

import (
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// Button is an outlined chip with a centered label; it brightens on hover.
func Button(label string, opts ...Option) *El {
	return Row(with([]Item{
		Bg(widget.Sunken), Control("button"), Lighten(),
		Text(label, Center(), Grow(1), Color(widget.ChipText)),
	}, opts)...)
}

// LinkButton is a quiet text button: accent-colored text on no
// background, lit softly under the pointer. It sizes to its label.
func LinkButton(label string, opts ...Option) *El {
	return Row(with([]Item{
		PadXY(8, 0), HoverBg(widget.Hover),
		Text(label, Center(), Color(widget.AccentLine)),
	}, opts)...)
}

// PrimaryButton is the accent-colored call-to-action variant of Button.
func PrimaryButton(label string, opts ...Option) *El {
	return Row(with([]Item{
		Bg(widget.Accent), Control("primary"), Lighten(),
		Text(label, Center(), Grow(1), Color(widget.AccentText)),
	}, opts)...)
}

// TabItem is one tab; Badge > 0 shows a count after the label.
type TabItem struct {
	Label string
	Badge int
}

// Tabs is a strip of labels with a gold bar under the active one.
func Tabs(items []TabItem, active int, onSelect func(i int), opts ...Option) *El {
	return Row(with(append([]Item{PadL(2)}, tabItems(items, active, onSelect)...), opts)...)
}

// ScrollTabs is Tabs in a strip that scrolls sideways when the labels do
// not fit, keeping the active tab in view. It grows to the free width.
func ScrollTabs(state *ScrollState, items []TabItem, active int, onSelect func(i int), opts ...Option) *El {
	return HScroll(state, with(append([]Item{Grow(1), PadL(2)}, tabItems(items, active, onSelect)...), opts)...)
}

func tabItems(items []TabItem, active int, onSelect func(i int)) []Item {
	out := make([]Item, len(items))
	for i, it := range items {
		i, it := i, it
		tab := []Item{
			Gap(4), PadXY(10, 0), AlignItems(AlignBaseline),
			OnClick(func() { onSelect(i) }),
			Separator(i > 0, 10),
			Text(it.Label, Color(tabColor(i == active)), BoldIf(i == active)),
			If(it.Badge > 0, Text(strconv.Itoa(it.Badge), Small(), Muted())),
		}
		if i == active {
			tab = append(tab, Underline(widget.AccentLine, 10), Reveal())
		}
		out[i] = Row(tab...)
	}
	return out
}

func tabColor(active bool) widget.Token {
	if active {
		return widget.Text
	}
	return widget.Muted
}

// TableRow is one row of a Table. Muted rows (disabled entries) are drawn dim.
type TableRow struct {
	Cells   []string
	Checked bool
	Muted   bool
	Flagged bool // draw the name in the error color, for a duplicate
}

// TableSpec describes a Table: header captions, relative column widths,
// the rows, and for an editable table the checkbox and delete handlers.
// An editable table also shows a dim placeholder row at the end.
type TableSpec struct {
	Columns     []string
	Weights     []float64
	Rows        []TableRow
	Placeholder []string
	Editable    bool
	OnToggle    func(row int)
	OnDelete    func(row int)

	// OnEditCell runs when a cell is clicked; Editing, when set, shows a
	// text field in that cell instead of its text.
	OnEditCell func(row, col int)
	Editing    *TableEdit

	// AllChecked and OnToggleAll drive the checkbox in the header's
	// action cell; Selected is the row the keyboard acts on.
	AllChecked  bool
	OnToggleAll func()
	Selected    *int

	// OnRowMenu runs when a data row is right-clicked, with the pointer position.
	OnRowMenu func(row, x, y int)

	// Widths, when set, makes the column dividers draggable. It is owned
	// by the app, so the sizes and a drag in progress survive the view
	// being rebuilt every frame.
	Widths *ColumnWidths
}

// ColumnWidths is the state of a resizable table: the column weights
// once the user has resized them (nil until then, so the spec's Weights
// apply) and the divider being dragged.
type ColumnWidths struct {
	Weights []float64

	dragging bool
	divider  int // the divider being dragged: between columns divider and divider+1
	grab     int // pointer offset inside the divider when the drag began
}

// weight is column i's relative width.
func (t TableSpec) weight(i int) float64 {
	if t.Widths != nil && len(t.Widths.Weights) == len(t.Columns) {
		return t.Widths.Weights[i]
	}
	return t.Weights[i]
}

func (t TableSpec) weights() []float64 {
	out := make([]float64, len(t.Columns))
	for i := range out {
		out[i] = t.weight(i)
	}
	return out
}

// TableEdit is the cell being edited: its position, the field holding
// the text, and what Enter and Tab do.
type TableEdit struct {
	Row, Col int
	Field    *widget.TextField
	OnEnter  func()
	OnTab    func(back bool)
	OnMenu   func(x, y int) // right click on the edit field
}

const tableActionW = 36

// Table is a zebra-striped grid under an uppercase header with a rule.
func Table(t TableSpec, opts ...Option) *El {
	rows := t.Rows
	if t.Editable {
		rows = append(append([]TableRow{}, rows...), TableRow{Cells: t.Placeholder, Muted: true})
	}
	return Col(with([]Item{
		Bg(widget.Sunken), Control("table"),
		tableHeader(t),
		For(rows, func(i int, r TableRow) Node {
			return tableRow(t, i, r, t.Editable && i == len(t.Rows), i == len(rows)-1)
		}),
	}, opts)...)
}

// headerCheck is the checkbox that enables or disables every row.
// tableHeader is the caption row; with a Widths state it carries the
// draggable column dividers on top.
func tableHeader(t TableSpec) Node {
	header := Row(H(widget.RowH), Bg(widget.Panel), BorderBottom(widget.Border),
		If(t.Editable, headerCheck(t)),
		For(t.Columns, func(i int, name string) Node {
			return Row(Grow(t.weight(i)), PadXY(8, 0), Separator(i > 0, 7),
				Text(strings.ToUpper(name), Small(), Bold(), Muted(), Grow(1)))
		}),
		If(t.Editable, Row(W(tableActionW))),
	)
	if t.Widths == nil {
		return header
	}
	return Stack(H(widget.RowH), header, newEl(colHandlesB{header: header, spec: t}, nil))
}

func headerCheck(t TableSpec) Node {
	items := []Item{W(tableActionW), JustifyContent(JustifyCenter), Tooltip("Enable or disable every row"), Checkbox(t.AllChecked)}
	if t.OnToggleAll != nil {
		items = append(items, OnClick(t.OnToggleAll))
	}
	return Row(items...)
}

func tableRow(t TableSpec, i int, r TableRow, placeholder, last bool) Node {
	clr := widget.Text
	if r.Muted {
		clr = widget.Muted
	}
	actions := t.Editable && !placeholder
	row := []Item{
		H(widget.RowH),
		If(actions, Row(W(tableActionW), JustifyContent(JustifyCenter), OnClick(func() { t.OnToggle(i) }), Tooltip("Enable or disable this row"), Checkbox(r.Checked))),
		If(placeholder, Row(W(tableActionW))),
		For(r.Cells, func(j int, s string) Node {
			cellClr := clr
			if j == 0 && r.Flagged {
				cellClr = widget.Error
			}
			return tableCell(t, i, j, s, cellClr)
		}),
		If(actions, Row(W(tableActionW), JustifyContent(JustifyCenter), OnClick(func() { t.OnDelete(i) }), Tooltip("Delete row"), Trash())),
		If(placeholder, Row(W(tableActionW))),
	}
	if t.OnRowMenu != nil && !placeholder {
		row = append(row, OnRightClick(func(x, y int) { t.OnRowMenu(i, x, y) }))
	}
	switch {
	case t.Selected != nil && *t.Selected == i && !placeholder:
		row = append(row, Bg(widget.Selected))
	}
	if !last { // hairlines between rows, as the dashboard's tables draw them
		row = append(row, BorderBottom(widget.BorderSoft))
	}
	if !placeholder {
		row = append(row, HoverBg(widget.Hover)) // the row under the pointer
	}
	return Row(row...)
}

// tableCell is one cell: its text, or the edit field when it is being
// edited. In an editable table a click on it asks to edit it.
func tableCell(t TableSpec, row, col int, text string, clr widget.Token) Node {
	if e := t.Editing; e != nil && e.Row == row && e.Col == col {
		// The field is inset in its cell so its rounded outline and focus ring have room.
		return Row(Grow(t.weight(col)), PadXY(3, 3), Separator(col > 0, 0),
			Field(e.Field, Grow(1), Bg(widget.Sunken), OnEnter(e.OnEnter), OnTab(e.OnTab), menuOption(e.OnMenu)),
		)
	}
	items := []Item{Grow(t.weight(col)), PadXY(8, 0), Clip(), Text(text, Ellipsis(), Color(clr), Grow(1)), Separator(col > 0, 0)}
	if t.Editable && t.OnEditCell != nil {
		items = append(items, OnClick(func() { t.OnEditCell(row, col) }))
	}
	return Row(items...)
}

// menuOption is OnRightClick for an optional handler.
func menuOption(f func(x, y int)) Option {
	if f == nil {
		return func(*El) {}
	}
	return OnRightClick(f)
}

// MinColumnPx is the narrowest a table column can be dragged.
const MinColumnPx = 56

// ResizePair moves the divider between columns i and i+1: the two share
// their combined weight so that column i is leftPx wide out of totalPx
// for the pair, neither narrower than minPx. The other weights and the
// total stay as they were. weights is not changed.
func ResizePair(weights []float64, i, leftPx, totalPx, minPx int) []float64 {
	out := append([]float64(nil), weights...)
	if i < 0 || i+1 >= len(out) || totalPx <= 0 {
		return out
	}
	pair := out[i] + out[i+1]
	if lo, hi := minPx, totalPx-minPx; hi < lo {
		leftPx = totalPx / 2 // no room for two minimums: split evenly
	} else {
		leftPx = min(max(leftPx, lo), hi)
	}
	out[i] = pair * float64(leftPx) / float64(totalPx)
	out[i+1] = pair - out[i]
	return out
}

// colHandlesB covers a table header and turns the boundaries between
// its column cells into draggable dividers.
type colHandlesB struct {
	plain
	header *El // the header row; its column cells are kids[offset : offset+columns]
	spec   TableSpec
}

func (colHandlesB) measure(*El, *widget.Ctx, int) (int, int) { return 0, 0 }

func (h colHandlesB) cells() []*El {
	off := 0
	if h.spec.Editable {
		off = 1 // the "enable all" cell comes first
	}
	return h.header.kids[off : off+len(h.spec.Columns)]
}

func (h colHandlesB) update(e *El, c *widget.Ctx) {
	st, in, cells := h.spec.Widths, c.In, h.cells()
	if st.dragging {
		if !in.Held {
			st.dragging = false
			return
		}
		c.Cursor = widget.CursorResizeH
		left, right := cells[st.divider].rect, cells[st.divider+1].rect
		st.Weights = ResizePair(h.spec.weights(), st.divider, in.X-left.X-st.grab, left.W+right.W, MinColumnPx)
		return
	}
	for i := 0; i < len(cells)-1; i++ {
		x := cells[i].rect.X + cells[i].rect.W
		grab := widget.Rect{X: x - handleSlop, Y: h.header.rect.Y, W: 1 + 2*handleSlop, H: h.header.rect.H}
		if !in.Over(grab) {
			continue
		}
		c.Cursor = widget.CursorResizeH
		if in.Click && in.Clicks >= 2 {
			st.Weights = nil // back to the defaults
		} else if in.Click {
			st.dragging, st.divider, st.grab = true, i, in.X-x
		}
		return
	}
}
