package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// bulkEdit is a table shown as text, one "name: value" line per row.
type bulkEdit struct {
	pane pane
	tab  *reqTab
	area view.TextArea
}

// rowSelection is the table row the keyboard operations act on.
type rowSelection struct {
	pane pane
	tab  *reqTab
	row  int
}

// applyRows is the one place a table's rows change: it runs change on a
// copy, and when the rows differ records the old ones for undo, stores
// the new ones and runs the pane's changed hook (the URL rewrite for
// Params). An edit or selection pointing past the new rows is dropped.
func (a *App) applyRows(p pane, t *reqTab, change func(rows []model.KV) []model.KV) {
	if t.History(p.label).Apply(p.rows(a, t), change) {
		a.rowsAltered(p, t)
	}
}

// rowsAltered runs what must follow any change of a table's rows.
func (a *App) rowsAltered(p pane, t *reqTab) {
	a.rowsChanged(p, t)
	n := len(*p.rows(a, t))
	if e := a.edit; e != nil && e.pane.label == p.label && e.tab == t && e.row > n {
		a.edit = nil
	}
	if s := a.rowSel; s != nil && s.pane.label == p.label && s.tab == t && s.row >= n {
		a.rowSel = nil
	}
}

// undoRows takes back the last change of a table; redoRows reapplies it.
func (a *App) undoRows(p pane, t *reqTab) {
	if t.History(p.label).Undo(p.rows(a, t)) {
		a.rowsAltered(p, t)
	}
}

func (a *App) redoRows(p pane, t *reqTab) {
	if t.History(p.label).Redo(p.rows(a, t)) {
		a.rowsAltered(p, t)
	}
}

// visibleTable is the editable table the request pane shows, if any:
// Params, Headers, Vars, or the form fields of the Body pane.
func (a *App) visibleTable() (pane, *reqTab, bool) {
	t := a.cur()
	p := requestPanes[t.paneTab]
	switch {
	case p.kind == paneTable && p.editable:
		return p, t, true
	case p.kind == paneBody && (t.Req.BodyType == model.BodyForm || t.Req.BodyType == model.BodyMultipart):
		return formPane, t, true
	}
	return pane{}, nil, false
}

// rowTarget is the row the keyboard operations act on: the edited row,
// or the selected one while its table is showing.
func (a *App) rowTarget() (p pane, t *reqTab, row, col int, editing, ok bool) {
	if e := a.edit; e != nil {
		if e.row < len(*e.pane.rows(a, e.tab)) {
			return e.pane, e.tab, e.row, e.col, true, true
		}
		return pane{}, nil, 0, 0, false, false
	}
	vp, vt, visible := a.visibleTable()
	if s := a.rowSel; s != nil && visible && s.tab == vt && s.pane.label == vp.label && s.row < len(*vp.rows(a, vt)) {
		return vp, vt, s.row, 0, false, true
	}
	return pane{}, nil, 0, 0, false, false
}

// afterRowOp puts the edit or the selection on row, in column col.
func (a *App) afterRowOp(p pane, t *reqTab, row, col int, editing bool) {
	if editing {
		a.beginEdit(p, t, row, col)
		return
	}
	a.rowSel = &rowSelection{pane: p, tab: t, row: row}
}

// insertRowBelow adds an empty enabled row under the current one and edits it.
func (a *App) insertRowBelow() {
	p, t, row, col, editing, ok := a.rowTarget()
	if !ok {
		return
	}
	a.commitEdit()
	a.applyRows(p, t, func(rows []model.KV) []model.KV { return model.InsertRow(rows, row+1, model.KV{Enabled: true}) })
	a.beginEdit(p, t, row+1, col) // a new row is for typing into
	_ = editing
}

// duplicateRow copies the current row right below it.
func (a *App) duplicateRow() {
	p, t, row, col, editing, ok := a.rowTarget()
	if !ok {
		return
	}
	a.commitEdit()
	a.applyRows(p, t, func(rows []model.KV) []model.KV { return model.DuplicateRow(rows, row) })
	a.afterRowOp(p, t, row+1, col, editing)
}

// moveRow moves the current row up (delta -1) or down; an edit follows it.
func (a *App) moveRow(delta int) {
	p, t, row, col, editing, ok := a.rowTarget()
	if !ok {
		return
	}
	a.commitEdit()
	to := row
	a.applyRows(p, t, func(rows []model.KV) []model.KV {
		moved, idx := model.MoveRow(rows, row, delta)
		to = idx
		return moved
	})
	a.afterRowOp(p, t, to, col, editing)
}

// grabRow starts dragging a row by its grip; the row becomes the
// selected one.
func (a *App) grabRow(p pane, t *reqTab, row int) {
	a.commitEdit()
	a.rowSel = &rowSelection{pane: p, tab: t, row: row}
	a.dragRow = &rowSelection{pane: p, tab: t, row: row}
}

// dragRowOver moves the dragged row to the place of the row the pointer
// is over, so the table shows where it would land. A release ends the
// drag (see step).
func (a *App) dragRowOver(over int) {
	drag := a.dragRow
	if drag == nil || drag.row == over {
		return
	}
	a.applyRows(drag.pane, drag.tab, func(rows []model.KV) []model.KV {
		moved, _ := model.MoveRow(rows, drag.row, over-drag.row)
		return moved
	})
	drag.row = over
	a.rowSel = &rowSelection{pane: drag.pane, tab: drag.tab, row: over}
}

// deleteRow removes the selected row (not while a cell is edited, when
// Delete belongs to the text).
func (a *App) deleteRow() {
	if a.edit != nil || a.textFocused() || a.treeFocus {
		return
	}
	p, t, row, _, _, ok := a.rowTarget()
	if !ok {
		return
	}
	a.applyRows(p, t, func(rows []model.KV) []model.KV { return model.DeleteRow(rows, row) })
	if n := len(*p.rows(a, t)); n > 0 {
		a.rowSel = &rowSelection{pane: p, tab: t, row: min(row, n-1)}
	}
}

// ---- bulk edit -------------------------------------------------------------

func (a *App) toggleBulk(p pane, t *reqTab) {
	if a.bulk != nil {
		a.closeBulk()
		return
	}
	a.commitEdit()
	a.bulk = &bulkEdit{pane: p, tab: t, area: view.NewTextArea(model.FormatBulk(*p.rows(a, t)))}
}

// closeBulk turns the text back into rows, as one change.
func (a *App) closeBulk() {
	b := a.bulk
	if b == nil {
		return
	}
	a.bulk = nil
	text := b.area.Text
	a.applyRows(b.pane, b.tab, func(rows []model.KV) []model.KV { return model.ParseBulk(text, rows) })
}

// settleBulk ends bulk mode when its table is no longer showing, so
// nothing typed is lost by switching panes or tabs.
func (a *App) settleBulk() {
	if a.bulk == nil {
		return
	}
	if p, t, ok := a.visibleTable(); !ok || p.label != a.bulk.pane.label || t != a.bulk.tab {
		a.closeBulk()
	}
}
