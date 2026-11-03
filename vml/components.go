package vml

import (
	"strconv"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Button is a dark chip with a centered label; it brightens on hover.
func Button(label string, opts ...Option) *El {
	return Row(with([]Item{
		Bg(widget.Chip), Lighten(),
		Text(label, Center(), Grow(1), Color(widget.ChipText)),
	}, opts)...)
}

// PrimaryButton is the gold call-to-action variant of Button.
func PrimaryButton(label string, opts ...Option) *El {
	return Row(with([]Item{
		Bg(widget.Accent), Lighten(),
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
	return Row(with([]Item{
		PadL(2),
		For(items, func(i int, it TabItem) Node {
			tab := []Item{
				Gap(4), PadXY(10, 0),
				OnClick(func() { onSelect(i) }),
				Text(it.Label, Color(tabColor(i == active))),
				If(it.Badge > 0, Text(strconv.Itoa(it.Badge), Small(), Muted())),
			}
			if i == active {
				tab = append(tab, Underline(widget.Accent, 10))
			}
			return Row(tab...)
		}),
	}, opts)...)
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
}

// TableEdit is the cell being edited: its position, the field holding
// the text, and what Enter and Tab do.
type TableEdit struct {
	Row, Col int
	Field    *widget.TextField
	OnEnter  func()
	OnTab    func(back bool)
}

const tableActionW = 36

// Table is a zebra-striped grid under an uppercase header with a rule.
func Table(t TableSpec, opts ...Option) *El {
	rows := t.Rows
	if t.Editable {
		rows = append(append([]TableRow{}, rows...), TableRow{Cells: t.Placeholder, Muted: true})
	}
	return Col(with([]Item{
		Bg(widget.Sunken),
		Row(H(widget.RowH), BorderBottom(widget.Rule),
			If(t.Editable, headerCheck(t)),
			For(t.Columns, func(i int, name string) Node {
				return Row(Grow(t.Weights[i]), PadXY(8, 0), Caption(name, Grow(1)))
			}),
			If(t.Editable, Row(W(tableActionW))),
		),
		For(rows, func(i int, r TableRow) Node {
			return tableRow(t, i, r, t.Editable && i == len(t.Rows))
		}),
	}, opts)...)
}

// headerCheck is the checkbox that enables or disables every row.
func headerCheck(t TableSpec) Node {
	items := []Item{W(tableActionW), Checkbox(t.AllChecked)}
	if t.OnToggleAll != nil {
		items = append(items, OnClick(t.OnToggleAll))
	}
	return Row(items...)
}

func tableRow(t TableSpec, i int, r TableRow, placeholder bool) Node {
	clr := widget.Text
	if r.Muted {
		clr = widget.Muted
	}
	actions := t.Editable && !placeholder
	row := []Item{
		H(widget.RowH),
		If(actions, Row(W(tableActionW), OnClick(func() { t.OnToggle(i) }), Checkbox(r.Checked))),
		If(placeholder, Row(W(tableActionW))),
		For(r.Cells, func(j int, s string) Node {
			cellClr := clr
			if j == 0 && r.Flagged {
				cellClr = widget.Error
			}
			return tableCell(t, i, j, s, cellClr)
		}),
		If(actions, Row(W(tableActionW), OnClick(func() { t.OnDelete(i) }), Cross())),
		If(placeholder, Row(W(tableActionW))),
	}
	switch {
	case t.Selected != nil && *t.Selected == i && !placeholder:
		row = append(row, Bg(widget.Selected))
	case i%2 == 1:
		row = append(row, Bg(widget.Alt))
	}
	return Row(row...)
}

// tableCell is one cell: its text, or the edit field when it is being
// edited. In an editable table a click on it asks to edit it.
func tableCell(t TableSpec, row, col int, text string, clr widget.Token) Node {
	if e := t.Editing; e != nil && e.Row == row && e.Col == col {
		return Row(Grow(t.Weights[col]), Border(widget.Accent), Bg(widget.Panel),
			Field(e.Field, Grow(1), OnEnter(e.OnEnter), OnTab(e.OnTab)),
		)
	}
	items := []Item{Grow(t.Weights[col]), PadXY(8, 0), Clip(), Text(text, Ellipsis(), Color(clr), Grow(1))}
	if t.Editable && t.OnEditCell != nil {
		items = append(items, OnClick(func() { t.OnEditCell(row, col) }))
	}
	return Row(items...)
}
