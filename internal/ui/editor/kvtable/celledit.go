package kvtable

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// liveEdit is the cell edit, unless its tab closed since: that edit is
// dropped, its text lost. Every read of the edit goes through here, so a
// closed tab's edit is never written or shown.
func (c *Component) liveEdit() *cellEdit {
	if c.edit != nil && !c.deps.Tabs.IsOpen(c.edit.Tab) {
		c.edit = nil
	}
	return c.edit
}

// Editing reports whether a cell is being edited.
func (c *Component) Editing() bool { return c.liveEdit() != nil }

// Typing reports whether the keyboard is in a cell or in the bulk text.
func (c *Component) Typing() bool {
	return c.liveEdit() != nil || (c.bulk != nil && c.bulk.area.Focused)
}

// BeginEdit starts editing a cell, committing any edit in progress.
// The old text is selected so typing replaces it.
func (c *Component) BeginEdit(table logic.Table, row, col int) {
	c.Commit()
	rows := *c.rows(table)
	field := widget.NewTextField(logic.CellText(rows, row, col))
	if field.Text != "" {
		field.SelectAll()
	}
	field.Focused = true
	c.edit = &cellEdit{Cell: logic.Cell{Table: table, Row: row, Col: col}, field: field}
	if row < len(rows) {
		c.Select(table, row)
	}
}

// Commit writes the edited text into its row and reports whether there was
// an edit. Text typed into the placeholder row creates a row; an empty
// placeholder cell adds nothing.
func (c *Component) Commit() bool {
	edit := c.liveEdit()
	if edit == nil {
		return false
	}
	c.edit = nil
	text := edit.field.Text
	c.applyRows(edit.Table, func(rows []request.KV) []request.KV {
		return logic.SetCell(rows, edit.Row, edit.Col, text)
	})
	return true
}

// CancelEdit drops the edit without writing anything.
func (c *Component) CancelEdit() { c.edit = nil }

// tabEdit commits and moves to the next cell (the previous one with
// shift), wrapping across rows and ending in the placeholder row.
func (c *Component) tabEdit(back bool) {
	edit := c.liveEdit()
	if edit == nil {
		return
	}
	cell := edit.Cell
	c.Commit()
	next := logic.NextCell(cell, back, len(*c.rows(cell.Table)))
	c.BeginEdit(next.Table, next.Row, next.Col)
}
