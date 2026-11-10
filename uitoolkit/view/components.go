package view

import (
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// buttonPadX is the inset between the edge of a button and its label.
const buttonPadX = 10

// Button is an outlined chip with a centered label; it brightens on hover.
func Button(label string, opts ...Option) *El {
	return Row(with([]Item{
		Bg(core.Sunken), Control("button"), Lighten(), PadX(buttonPadX),
		Text(label, TextCenter(), Ellipsis(), Grow(1), Color(core.ChipText)),
	}, opts)...)
}

// LinkButton is a quiet text button: accent-colored text on no
// background, lit softly under the pointer. It sizes to its label.
func LinkButton(label string, opts ...Option) *El {
	return Row(with([]Item{
		PadX(8), HoverBg(core.Hover),
		Text(label, TextCenter(), Color(core.AccentLine)),
	}, opts)...)
}

// PrimaryButton is the accent-colored call-to-action variant of Button.
func PrimaryButton(label string, opts ...Option) *El {
	return Row(with([]Item{
		Bg(core.Accent), Control("primary"), Lighten(), PadX(buttonPadX),
		Text(label, TextCenter(), Ellipsis(), Grow(1), Color(core.AccentText)),
	}, opts)...)
}

// TabItem is one tab; Badge > 0 shows a count after the label.
type TabItem struct {
	Label string
	Badge int
}

// Tabs is a strip of labels with a gold bar under the active one.
func Tabs(items []TabItem, active int, opts ...Option) *El {
	var strip *El
	strip = Row(with(append([]Item{PadL(2)}, tabItems(items, active, selector(&strip))...), opts)...)
	return strip
}

// selector is the click handler of the tabs of *strip: it runs the strip's
// OnSelect, which is set only after the tabs are built.
func selector(strip **El) func(i int) {
	return func(i int) {
		if (*strip).onSelect != nil {
			(*strip).onSelect(i)
		}
	}
}

// OnSelect runs with the index of the tab that was clicked.
func OnSelect(f func(i int)) Option { return func(e *El) { e.onSelect = f } }

// ScrollTabs is Tabs in a strip that scrolls sideways when the labels do
// not fit, keeping the active tab in view. It grows to the free width.
func ScrollTabs(state *ScrollState, items []TabItem, active int, opts ...Option) *El {
	var strip *El
	strip = HScroll(state, with(append([]Item{Grow(1), PadL(2)}, tabItems(items, active, selector(&strip))...), opts)...)
	return strip
}

func tabItems(items []TabItem, active int, onSelect func(i int)) []Item {
	out := make([]Item, len(items))
	for i, it := range items {
		i, it := i, it
		tab := []Item{
			Gap(4), PadX(10), AlignItems(AlignBaseline),
			OnClick(func() { onSelect(i) }),
			When(i > 0, Separator(10)),
			Text(it.Label, Color(tabColor(i == active))),
			If(it.Badge > 0, Text(strconv.Itoa(it.Badge), Small(), Muted())),
		}
		if i == active { // outlined top, left and right in the accent and open at the bottom, where the strip's line runs under the others
			tab = append(tab, Bg(core.Bg), EdgeT(core.AccentLine), EdgeL(core.AccentLine), EdgeR(core.AccentLine), KeepInView())
		}
		out[i] = Row(tab...)
	}
	return out
}

func tabColor(active bool) core.Token {
	if active {
		return core.Text
	}
	return core.Muted
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

	// CellStyler, when set, may color parts of a cell's text: it returns
	// the text cut into spans (an empty Style keeps the cell's own color),
	// or nil to draw the cell as plain text.
	CellStyler func(text string) []Span

	// OnRowMenu runs when a data row is right-clicked, with the pointer position.
	OnRowMenu func(row, x, y int)

	// OnGrabRow and OnDragOverRow add a drag handle at the start of every
	// data row: pressing it runs OnGrabRow, and while the button is held
	// each row under the pointer runs OnDragOverRow. The app owns the drag
	// (which row, when it ends), so it survives the view being rebuilt
	// every frame and is done on a release.
	OnGrabRow     func(row int)
	OnDragOverRow func(row int)

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

	drag    grab
	divider int // the divider being dragged: between columns divider and divider+1
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
	Field    *TextField
	OnEnter  func()
	OnTab    func(back bool)
	OnMenu   func(x, y int) // right click on the edit field
}

const tableActionW = 36

// tableGripW is the width of the drag handle column.
const tableGripW = 20

// Table is a zebra-striped grid under an uppercase header with a rule.
func Table(t TableSpec, opts ...Option) *El {
	rows := t.Rows
	if t.Editable {
		rows = append(append([]TableRow{}, rows...), TableRow{Cells: t.Placeholder, Muted: true})
	}
	return Col(with([]Item{
		Bg(core.Sunken), Control("table"),
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
	header := Row(H(core.RowH), Bg(core.Panel), BorderB(core.Border),
		If(t.hasGrip(), Row(W(tableGripW))),
		If(t.Editable, headerCheck(t)),
		For(t.Columns, func(i int, name string) Node {
			return Row(Grow(t.weight(i)), PadX(8), When(i > 0, Separator(7)),
				Text(strings.ToUpper(name), Small(), Bold(), Muted(), Grow(1)))
		}),
		If(t.Editable, Row(W(tableActionW))),
	)
	if t.Widths == nil {
		return header
	}
	return Stack(H(core.RowH), header, newEl(colHandlesB{header: header, spec: t}, nil))
}

// hasGrip reports whether rows have a drag handle.
func (t TableSpec) hasGrip() bool { return t.OnGrabRow != nil }

func headerCheck(t TableSpec) Node {
	items := []Item{W(tableActionW), JustifyContent(JustifyCenter), Tooltip("Enable or disable every row"), Checkbox(t.AllChecked)}
	if t.OnToggleAll != nil {
		items = append(items, OnClick(t.OnToggleAll))
	}
	return Row(items...)
}

func tableRow(t TableSpec, i int, r TableRow, placeholder, last bool) Node {
	clr := core.Text
	if r.Muted {
		clr = core.Muted
	}
	actions := t.Editable && !placeholder
	row := []Item{
		H(core.RowH),
		If(t.hasGrip(), gripCell(t, i, placeholder)),
		If(actions, Row(W(tableActionW), JustifyContent(JustifyCenter), OnClick(func() { t.OnToggle(i) }), Tooltip("Enable or disable this row"), Checkbox(r.Checked))),
		If(placeholder, Row(W(tableActionW))),
		For(r.Cells, func(j int, s string) Node {
			cellClr := clr
			if j == 0 && r.Flagged {
				cellClr = core.Error
			}
			return tableCell(t, i, j, s, cellClr)
		}),
		If(actions, Row(W(tableActionW), JustifyContent(JustifyCenter), OnClick(func() { t.OnDelete(i) }), Tooltip("Delete row"), TrashIcon())),
		If(placeholder, Row(W(tableActionW))),
	}
	if t.OnDragOverRow != nil && !placeholder {
		row = append(row, OnDragOver(func() { t.OnDragOverRow(i) }))
	}
	if t.OnRowMenu != nil && !placeholder {
		row = append(row, OnRightClick(func(x, y int) { t.OnRowMenu(i, x, y) }))
	}
	switch {
	case t.Selected != nil && *t.Selected == i && !placeholder:
		row = append(row, Bg(core.Selected))
	}
	if !last { // hairlines between rows, as the dashboard's tables draw them
		row = append(row, BorderB(core.BorderSoft))
	}
	if !placeholder {
		row = append(row, HoverBg(core.Hover)) // the row under the pointer
	}
	return Row(row...)
}

// gripCell is the drag handle of row i; the placeholder row keeps an
// empty cell so the columns line up.
func gripCell(t TableSpec, i int, placeholder bool) Node {
	if placeholder {
		return Row(W(tableGripW))
	}
	return Row(W(tableGripW), JustifyContent(JustifyCenter), Tooltip("Drag to move the row"),
		OnClick(func() { t.OnGrabRow(i) }), GripIcon(Muted()))
}

// tableCell is one cell: its text, or the edit field when it is being
// edited. In an editable table a click on it asks to edit it.
func tableCell(t TableSpec, row, col int, text string, clr core.Token) Node {
	if e := t.Editing; e != nil && e.Row == row && e.Col == col {
		// The field is inset in its cell so its rounded outline and focus ring have room.
		return Row(Grow(t.weight(col)), PadX(3), PadY(3), When(col > 0, Separator(0)),
			Field(e.Field, Grow(1), Bg(core.Sunken), FieldPad(5) /* 3 + 5 = the 8 px of a shown cell, so the text does not move */, OnEnter(e.OnEnter), OnTab(e.OnTab), menuOption(e.OnMenu)),
		)
	}
	var label *El = Text(text, Ellipsis(), Color(clr), Grow(1))
	if t.CellStyler != nil {
		if spans := t.CellStyler(text); len(spans) > 0 {
			parts := make([]Item, 0, len(spans)+1)
			parts = append(parts, Grow(1))
			for _, sp := range spans {
				color := sp.Style
				if color == "" {
					color = clr
				}
				parts = append(parts, Text(sp.Text, Color(color)))
			}
			label = Row(parts...)
		}
	}
	items := []Item{Grow(t.weight(col)), PadX(8), Clip(), label, When(col > 0, Separator(0))}
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

// resizePair moves the divider between columns i and i+1: the two share
// their combined weight so that column i is leftPx wide out of totalPx
// for the pair, neither narrower than minPx. The other weights and the
// total stay as they were. weights is not changed.
func resizePair(weights []float64, i, leftPx, totalPx, minPx int) []float64 {
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

func (colHandlesB) measure(*El, *core.Ctx, int) (int, int) { return 0, 0 }

func (h colHandlesB) cells() []*El {
	off := 0
	if h.spec.Editable {
		off = 1 // the "enable all" cell comes first
	}
	if h.spec.hasGrip() {
		off++ // and the drag handle's before it
	}
	return h.header.kids[off : off+len(h.spec.Columns)]
}

func (h colHandlesB) update(e *El, c *core.Ctx) {
	st, in, cells := h.spec.Widths, c.In, h.cells()
	if st.drag.on {
		if !st.drag.held(in) {
			return
		}
		c.Cursor = core.CursorResizeH
		left, right := cells[st.divider].rect, cells[st.divider+1].rect
		st.Weights = resizePair(h.spec.weights(), st.divider, in.X-left.X-st.drag.off, left.W+right.W, MinColumnPx)
		return
	}
	for i := 0; i < len(cells)-1; i++ {
		x := cells[i].rect.X + cells[i].rect.W
		grab := core.Rect{X: x - handleSlop, Y: h.header.rect.Y, W: 1 + 2*handleSlop, H: h.header.rect.H}
		if !in.Over(grab) {
			continue
		}
		c.Cursor = core.CursorResizeH
		if in.Click && in.Clicks >= 2 {
			st.Weights = nil // back to the defaults
		} else if in.Click {
			st.drag.begin(in.X - x)
			st.divider = i
		}
		return
	}
}
