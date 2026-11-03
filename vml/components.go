package vml

import (
	"strconv"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Button is a dark chip with a centered label; it brightens on hover.
func Button(label string) *El {
	return Row(Text(label).Center().Grow(1).Color(widget.ChipText)).Bg(widget.Chip).Lighten()
}

// PrimaryButton is the gold call-to-action variant of Button.
func PrimaryButton(label string) *El {
	return Row(Text(label).Center().Grow(1).Color(widget.AccentText)).Bg(widget.Accent).Lighten()
}

// TabItem is one tab; Badge > 0 shows a count after the label.
type TabItem struct {
	Label string
	Badge int
}

// Tabs is a strip of labels with a gold bar under the active one.
func Tabs(items []TabItem, active int, onSelect func(i int)) *El {
	return Row(For(items, func(i int, it TabItem) Node {
		tab := Row(
			Text(it.Label).Color(tabColor(i == active)),
			If(it.Badge > 0, Text(strconv.Itoa(it.Badge)).Small().Muted()),
		).Gap(4).PadXY(10, 0).OnClick(func() { onSelect(i) })
		if i == active {
			tab.Underline(widget.Accent, 10)
		}
		return tab
	})).PadL(2)
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
}

const tableActionW = 36

// Table is a zebra-striped grid under an uppercase header with a rule.
func Table(t TableSpec) *El {
	rows := t.Rows
	if t.Editable {
		rows = append(append([]TableRow{}, rows...), TableRow{Cells: t.Placeholder, Muted: true})
	}
	return Col(
		Row(
			If(t.Editable, Row().W(tableActionW)),
			For(t.Columns, func(i int, name string) Node {
				return Row(Caption(name).Grow(1)).PadXY(8, 0).Grow(t.Weights[i])
			}),
			If(t.Editable, Row().W(tableActionW)),
		).H(widget.RowH).BorderBottom(widget.Rule),
		For(rows, func(i int, r TableRow) Node {
			isPlaceholder := t.Editable && i == len(t.Rows)
			return tableRow(t, i, r, isPlaceholder)
		}),
	).Bg(widget.Sunken)
}

func tableRow(t TableSpec, i int, r TableRow, placeholder bool) Node {
	clr := widget.Text
	if r.Muted {
		clr = widget.Muted
	}
	row := Row(
		If(t.Editable && !placeholder, Row(Checkbox(r.Checked)).W(tableActionW).OnClick(func() { t.OnToggle(i) })),
		If(placeholder, Row().W(tableActionW)),
		For(r.Cells, func(j int, s string) Node {
			return Row(Text(s).Ellipsis().Color(clr).Grow(1)).PadXY(8, 0).Grow(t.Weights[j]).Clip()
		}),
		If(t.Editable && !placeholder, Row(Cross()).W(tableActionW).OnClick(func() { t.OnDelete(i) })),
		If(placeholder, Row().W(tableActionW)),
	).H(widget.RowH)
	if i%2 == 1 {
		row.Bg(widget.Alt)
	}
	return row
}
