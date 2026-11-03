package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// editColumns is the number of editable columns of a key/value table:
// name, value, description.
const editColumns = 3

// cellEdit is the table cell being edited: which pane's table, where,
// and the field holding the text.
type cellEdit struct {
	pane     pane
	tab      *reqTab
	row, col int
	field    widget.TextField
}

// cellText is the text of a cell of rows, "" for the placeholder row.
func cellText(rows []model.KV, row, col int) string {
	if row >= len(rows) {
		return ""
	}
	return *kvField(&rows[row], col)
}

// kvField points at the name, value or description of a row.
func kvField(kv *model.KV, col int) *string {
	return [editColumns]*string{&kv.Name, &kv.Value, &kv.Description}[col]
}

// beginEdit starts editing a cell, committing any edit in progress.
// The old text is selected so typing replaces it.
func (a *App) beginEdit(p pane, t *reqTab, row, col int) {
	a.commitEdit()
	field := widget.NewTextField(cellText(*p.rows(a, t), row, col))
	if field.Text != "" {
		field.SelectAll()
	}
	field.Focused = true
	a.edit = &cellEdit{pane: p, tab: t, row: row, col: col, field: field}
}

// commitEdit writes the edited text into its row. Text typed into the
// placeholder row creates a row; an empty placeholder cell adds nothing.
func (a *App) commitEdit() {
	e := a.edit
	if e == nil {
		return
	}
	a.edit = nil
	rows := e.pane.rows(a, e.tab)
	if e.row >= len(*rows) {
		if e.field.Text == "" {
			return
		}
		*rows = append(*rows, model.KV{Enabled: true})
	}
	*kvField(&(*rows)[e.row], e.col) = e.field.Text
	a.rowsChanged(e.pane, e.tab)
}

// cancelEdit drops the edit without writing anything.
func (a *App) cancelEdit() { a.edit = nil }

// tabEdit commits and moves to the next cell (the previous one with
// shift), wrapping across rows and ending in the placeholder row.
func (a *App) tabEdit(back bool) {
	e := a.edit
	if e == nil {
		return
	}
	p, t, row, col := e.pane, e.tab, e.row, e.col
	a.commitEdit()
	if back {
		col--
		if col < 0 {
			col, row = editColumns-1, row-1
		}
	} else {
		col++
		if col >= editColumns {
			col, row = 0, row+1
		}
	}
	last := len(*p.rows(a, t)) // index of the placeholder row
	if row < 0 {
		row, col = 0, 0
	}
	if row > last {
		row, col = last, editColumns-1
	}
	a.beginEdit(p, t, row, col)
}
