package kvtable

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// applyRows is the one place a table's rows change: it runs change on a
// copy, and when the rows differ records the old ones for undo, stores the
// new ones and tells the application (the URL rewrite for Params). An edit
// or selection pointing past the new rows is dropped.
func (c *Component) applyRows(table logic.Table, change func(rows []request.KV) []request.KV) {
	if table.Tab.History(table.Kind.Label()).Apply(c.rows(table), change) {
		c.rowsAltered(table)
	}
}

// rowsAltered runs what must follow any change of a table's rows.
func (c *Component) rowsAltered(table logic.Table) {
	c.deps.Tables.Changed(table)
	var edit *logic.Cell
	if c.edit != nil {
		edit = &c.edit.Cell
	}
	kept, selected := logic.Settle(edit, c.selected, table, len(*c.rows(table)))
	if kept == nil {
		c.edit = nil
	}
	c.selected = selected
}

// undo takes back the last change of a table; redo reapplies it.
func (c *Component) undo(table logic.Table) {
	if table.Tab.History(table.Kind.Label()).Undo(c.rows(table)) {
		c.rowsAltered(table)
	}
}

func (c *Component) redo(table logic.Table) {
	if table.Tab.History(table.Kind.Label()).Redo(c.rows(table)) {
		c.rowsAltered(table)
	}
}

// target is the row the keyboard operations act on: the edited row, or the
// selected one while its table is showing.
func (c *Component) target() (cell logic.Cell, editing, ok bool) {
	var edit *logic.Cell
	if live := c.liveEdit(); live != nil {
		edit = &live.Cell
	}
	visible, shown := c.deps.Tabs.Visible()
	return logic.Target(edit, c.selected, visible, shown, func(table logic.Table) int { return len(*c.rows(table)) })
}

// afterRowOp puts the edit or the selection on the row, in its column.
func (c *Component) afterRowOp(table logic.Table, row, col int, editing bool) {
	if editing {
		c.BeginEdit(table, row, col)
		return
	}
	c.Select(table, row)
}

// InsertRowBelow adds an empty enabled row under the current one and edits it.
func (c *Component) InsertRowBelow() {
	cell, _, ok := c.target()
	if !ok {
		return
	}
	c.Commit()
	c.applyRows(cell.Table, func(rows []request.KV) []request.KV {
		return request.InsertRow(rows, cell.Row+1, request.KV{Enabled: true})
	})
	c.BeginEdit(cell.Table, cell.Row+1, cell.Col) // a new row is for typing into
}

// DuplicateRow copies the current row right below it.
func (c *Component) DuplicateRow() {
	cell, editing, ok := c.target()
	if !ok {
		return
	}
	c.Commit()
	c.applyRows(cell.Table, func(rows []request.KV) []request.KV { return request.DuplicateRow(rows, cell.Row) })
	c.afterRowOp(cell.Table, cell.Row+1, cell.Col, editing)
}

// MoveRow moves the current row up (delta -1) or down; an edit follows it.
func (c *Component) MoveRow(delta int) {
	cell, editing, ok := c.target()
	if !ok {
		return
	}
	c.Commit()
	to := cell.Row
	c.applyRows(cell.Table, func(rows []request.KV) []request.KV {
		moved, index := request.MoveRow(rows, cell.Row, delta)
		to = index
		return moved
	})
	c.afterRowOp(cell.Table, to, cell.Col, editing)
}

// DeleteRow removes the selected row. The caller keeps it from running
// while a cell is edited, when Delete belongs to the text.
func (c *Component) DeleteRow() {
	cell, _, ok := c.target()
	if !ok {
		return
	}
	c.applyRows(cell.Table, func(rows []request.KV) []request.KV { return request.DeleteRow(rows, cell.Row) })
	if row, any := logic.AfterDelete(cell.Row, len(*c.rows(cell.Table))); any {
		c.Select(cell.Table, row)
	}
}

// GrabRow starts dragging a row by its grip; the row becomes the selected one.
func (c *Component) GrabRow(table logic.Table, row int) {
	c.Commit()
	c.Select(table, row)
	c.dragged = &logic.Cell{Table: table, Row: row}
}

// Dragging reports whether a row is held by its grip.
func (c *Component) Dragging() bool { return c.dragged != nil }

// dragRowOver moves the dragged row to the place of the row the pointer is
// over, so the table shows where it would land. A release ends the drag
// (BeginFrame).
func (c *Component) dragRowOver(over int) {
	drag := c.dragged
	if drag == nil || drag.Row == over {
		return
	}
	from := drag.Row
	c.applyRows(drag.Table, func(rows []request.KV) []request.KV {
		moved, _ := request.MoveRow(rows, from, over-from)
		return moved
	})
	drag.Row = over
	c.Select(drag.Table, over)
}

// ToggleBulk shows table as text, or turns the text back into rows.
func (c *Component) ToggleBulk(table logic.Table) {
	if c.bulk != nil {
		c.closeBulk()
		return
	}
	c.Commit()
	c.bulk = &bulkEdit{Table: table, area: widget.NewTextArea(request.FormatBulk(*c.rows(table)))}
}

// closeBulk turns the text back into rows, as one change.
func (c *Component) closeBulk() {
	bulk := c.bulk
	if bulk == nil {
		return
	}
	c.bulk = nil
	text := bulk.area.Text
	c.applyRows(bulk.Table, func(rows []request.KV) []request.KV { return request.ParseBulk(text, rows) })
}
