package ui

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// Right-click menus for tabs, table rows, text widgets and the response.
// The tree's own menu is in treeops.go. Every item does what its button
// or shortcut does, through the same code.

var menuSeparator = menuItem{separator: true}

// ---- request tabs ------------------------------------------------------------

func (a *App) tabMenuItems(i int) []menuItem {
	t := a.tabs[i]
	return []menuItem{
		{label: "Close", run: func() { a.closeTab(i) }},
		{label: "Close others", disabled: len(a.tabs) == 1, run: func() { a.closeOthers(i) }},
		{label: "Close tabs to the right", disabled: i == len(a.tabs)-1, run: func() { a.closeRightOf(i) }},
		menuSeparator,
		{label: "Duplicate", run: func() { a.duplicateTab(i) }},
		{label: "Copy URL", run: func() { a.copyRequestURL(t.req) }},
	}
}

// closeOthers closes every tab but tab i, which becomes the active one.
func (a *App) closeOthers(i int) {
	keep := a.tabs[i]
	for j := len(a.tabs) - 1; j >= 0; j-- {
		if a.tabs[j] != keep {
			a.closeTab(j)
		}
	}
	for j, t := range a.tabs {
		if t == keep {
			a.active = j
		}
	}
}

func (a *App) closeRightOf(i int) {
	for j := len(a.tabs) - 1; j > i; j-- {
		a.closeTab(j)
	}
}

// duplicateTab copies the tab's request next to the original in the
// tree (or into the root when it is not in the tree) and opens the copy.
func (a *App) duplicateTab(i int) {
	req := a.tabs[i].req
	cp, _ := a.col.Folder.Duplicate(req).(*model.Request)
	if cp == nil {
		clone := req.Clone()
		clone.Name += " copy"
		cp = a.col.Folder.AddRequest(clone)
	}
	a.openRequest(cp)
}

// ---- table rows ---------------------------------------------------------------

func (a *App) rowMenuItems(p pane, t *reqTab, row int) []menuItem {
	n := len(*p.rows(a, t))
	a.rowSel = &rowSelection{pane: p, tab: t, row: row}
	change := func(f func(rows []model.KV) []model.KV) func() {
		return func() {
			a.commitEdit()
			a.applyRows(p, t, f)
		}
	}
	items := append(a.copyRowItems(*p.rows(a, t), row, false), append([]menuItem{menuSeparator}, []menuItem{
		{label: "Insert row above", run: change(func(r []model.KV) []model.KV { return model.InsertRow(r, row, model.KV{Enabled: true}) })},
		{label: "Insert row below", run: change(func(r []model.KV) []model.KV { return model.InsertRow(r, row+1, model.KV{Enabled: true}) })},
		{label: "Duplicate", run: change(func(r []model.KV) []model.KV { return model.DuplicateRow(r, row) })},
		{label: "Move up", disabled: row == 0, run: change(func(r []model.KV) []model.KV { m, _ := model.MoveRow(r, row, -1); return m })},
		{label: "Move down", disabled: row >= n-1, run: change(func(r []model.KV) []model.KV { m, _ := model.MoveRow(r, row, 1); return m })},
		menuSeparator,
		{label: "Enable all", run: change(func(r []model.KV) []model.KV { return model.SetAll(r, true) })},
		{label: "Disable all", run: change(func(r []model.KV) []model.KV { return model.SetAll(r, false) })},
		menuSeparator,
		{label: "Delete", run: change(func(r []model.KV) []model.KV { return model.DeleteRow(r, row) })},
	}...)...)
	return append(items, a.secretItems(p, row)...)
}

// secretItems is the menu entry that marks a variable of the active
// environment as a secret; other tables have none.
func (a *App) secretItems(p pane, row int) []menuItem {
	e := a.activeEnv()
	if p.label != "Vars" || e == nil || row >= len(e.Vars) {
		return nil
	}
	label := "Mark as secret"
	if e.Vars[row].Secret {
		label = "Not a secret"
	}
	return []menuItem{menuSeparator, {label: label, run: func() { a.toggleSecret(row) }}}
}

// copyRowItems copy parts of table row i to the clipboard. They are the
// whole menu of a read-only table and lead the menu of an editable one.
func (a *App) copyRowItems(rows []model.KV, i int, withAll bool) []menuItem {
	if i < 0 || i >= len(rows) {
		return nil
	}
	r := rows[i]
	copyText := func(text string) func() {
		return func() { _ = a.ctx.Clipboard.Write(text) } // the clipboard reports its own failures
	}
	var all strings.Builder
	for _, h := range rows {
		all.WriteString(h.Name + ": " + h.Value + "\n")
	}
	items := []menuItem{
		{label: "Copy value", run: copyText(r.Value)},
		{label: "Copy name", run: copyText(r.Name)},
		{label: "Copy as Name: Value", run: copyText(r.Name + ": " + r.Value)},
	}
	if withAll {
		items = append(items, menuItem{label: "Copy all headers", run: copyText(all.String())})
	}
	return items
}

// ---- text widgets ---------------------------------------------------------------

// textMenu is the menu of a text field or area. changed, when given,
// runs after the menu changed the text, so the model follows.
func (a *App) textMenu(f *TextField, changed func()) []menuItem {
	done := func() {
		if changed != nil {
			changed()
		}
	}
	hasSel := f.SelectedText() != "" && !f.Masked // secrets never reach the clipboard
	return []menuItem{
		{label: "Undo", shortcut: "Ctrl+Z", disabled: !f.CanUndo(), run: func() {
			if f.Undo() {
				done()
			}
		}},
		{label: "Redo", shortcut: "Ctrl+Y", disabled: !f.CanRedo(), run: func() {
			if f.Redo() {
				done()
			}
		}},
		menuSeparator,
		{label: "Cut", shortcut: "Ctrl+X", disabled: !hasSel, run: func() {
			_ = a.ctx.Clipboard.Write(f.SelectedText())
			if f.DeleteSelection() {
				done()
			}
		}},
		{label: "Copy", shortcut: "Ctrl+C", disabled: !hasSel, run: func() { _ = a.ctx.Clipboard.Write(f.SelectedText()) }},
		{label: "Paste", shortcut: "Ctrl+V", run: func() {
			if text, err := a.ctx.Clipboard.Read(); err == nil && f.Insert(text) {
				done()
			}
		}},
		menuSeparator,
		{label: "Select all", shortcut: "Ctrl+A", disabled: f.Text == "", run: f.SelectAll},
	}
}

// textMenuOpt makes an element open the text menu of f on right click.
func (a *App) textMenuOpt(f *TextField, changed func()) Option {
	return OnRightClick(func(x, y int) { a.openMenu(x, y, a.textMenu(f, changed)) })
}

// ---- the response ------------------------------------------------------------------

func (a *App) responseMenuItems(t *reqTab) []menuItem {
	return []menuItem{
		{label: "Copy", shortcut: "Ctrl+C", disabled: !t.view.Selected(), run: func() {
			_ = a.ctx.Clipboard.Write(t.view.SelectedText())
		}},
		{label: "Copy body", run: func() { a.copyBody(t) }},
		{label: "Copy as cURL", run: func() { a.copyCurl(t) }},
		menuSeparator,
		{label: "Select all", shortcut: "Ctrl+A", run: func() { t.view.SelectLines(0, len(t.view.Lines)-1) }},
		{label: "Find", shortcut: a.shortcutHint("find"), run: a.openFind},
	}
}
