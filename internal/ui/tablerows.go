package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// bulkEdit is a table shown as text, one "name: value" line per row.
type bulkEdit struct {
	pane pane
	tab  *reqTab
	area *widget.TextArea
}

// rowSelection is the table row the keyboard operations act on.
type rowSelection struct {
	pane pane
	tab  *reqTab
	row  int
}

// historyLimit is how many table changes can be undone per table.
const historyLimit = 100

// rowHistory is the undo and redo stack of one table of a tab.
type rowHistory struct{ undo, redo [][]model.KV }

func (t *reqTab) history(label string) *rowHistory {
	if t.hist == nil {
		t.hist = map[string]*rowHistory{}
	}
	if h, ok := t.hist[label]; ok {
		return h
	}
	h := &rowHistory{}
	t.hist[label] = h
	return h
}

func cloneRows(rows []model.KV) []model.KV { return append([]model.KV(nil), rows...) }

func sameRows(a, b []model.KV) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// applyRows is the one place a table's rows change: it runs change on a
// copy, and when the rows differ records the old ones for undo, stores
// the new ones and runs the pane's changed hook (the URL rewrite for
// Params). An edit or selection pointing past the new rows is dropped.
func (a *App) applyRows(p pane, t *reqTab, change func(rows []model.KV) []model.KV) {
	rows := p.rows(a, t)
	before := cloneRows(*rows)
	after := change(cloneRows(*rows))
	if sameRows(before, after) {
		return
	}
	h := t.history(p.label)
	h.undo = append(h.undo, before)
	if len(h.undo) > historyLimit {
		h.undo = h.undo[len(h.undo)-historyLimit:]
	}
	h.redo = nil
	*rows = after
	a.rowsAltered(p, t)
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
func (a *App) undoRows(p pane, t *reqTab) { a.stepHistory(p, t, true) }
func (a *App) redoRows(p pane, t *reqTab) { a.stepHistory(p, t, false) }

func (a *App) stepHistory(p pane, t *reqTab, undo bool) {
	h := t.history(p.label)
	from, to := &h.undo, &h.redo
	if !undo {
		from, to = to, from
	}
	if len(*from) == 0 {
		return
	}
	rows := p.rows(a, t)
	*to = append(*to, cloneRows(*rows))
	*rows = (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	a.rowsAltered(p, t)
}

// visibleTable is the editable table the request pane shows, if any:
// Params, Headers, Vars, or the form fields of the Body pane.
func (a *App) visibleTable() (pane, *reqTab, bool) {
	t := a.cur()
	p := requestPanes[t.paneTab]
	switch {
	case p.kind == paneTable && p.editable:
		return p, t, true
	case p.kind == paneBody && (t.req.BodyType == model.BodyForm || t.req.BodyType == model.BodyMultipart):
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
	a.bulk = &bulkEdit{pane: p, tab: t, area: widget.NewTextArea(model.FormatBulk(*p.rows(a, t)))}
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
