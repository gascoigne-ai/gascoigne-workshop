package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strings"
)

// Right-click menus for tabs, table rows, text widgets and the response.
// The tree's own menu is in treemenu.go. Every item does what its button
// or shortcut does, through the same code.

var menuSeparator = MenuItem{Separator: true}

// ---- request tabs ------------------------------------------------------------

func (a *App) tabMenuItems(index int) []MenuItem {
	t := a.widgets[a.list.Tabs[index]]
	return []MenuItem{
		{Label: "Close", Run: func() { a.closeTab(index) }},
		{Label: "Close others", Disabled: len(a.list.Tabs) == 1, Run: func() { a.closeOthers(index) }},
		{Label: "Close tabs to the right", Disabled: index == len(a.list.Tabs)-1, Run: func() { a.closeRightOf(index) }},
		menuSeparator,
		{Label: "Duplicate", Run: func() { a.duplicateTab(index) }},
		{Label: "Copy URL", Run: func() { a.copyRequestURL(t.Req) }},
		{Label: "Cookies...", Run: func() { a.showCookies(t) }},
	}
}

// closeOthers closes every tab but the tab at index, which becomes the active one.
func (a *App) closeOthers(index int) { a.updateTabs(tabs.CloseOthers{Index: index}) }

func (a *App) closeRightOf(index int) { a.updateTabs(tabs.CloseRightOf{Index: index}) }

// duplicateTab copies the tab's request next to the original in the
// tree (or into the root when it is not in the tree) and opens the copy.
func (a *App) duplicateTab(index int) {
	req := a.list.Tabs[index].Req
	owner, ok := a.entryFor(req)
	if !ok { // nothing is open: the copy is a tab of its own
		clone := copyOf(req)
		a.openRequest(&clone)
		return
	}
	cp, _ := owner.Collection.Folder.Duplicate(req).(*model.Request)
	if cp == nil {
		cp = owner.Collection.Folder.AddRequest(copyOf(req))
	}
	a.openRequest(cp)
}

// copyOf is a clone of req named as a copy.
func copyOf(req *model.Request) model.Request {
	clone := req.Clone()
	clone.Name += " copy"
	return clone
}

// ---- table rows ---------------------------------------------------------------

func (a *App) rowMenuItems(panel pane, tab *reqTab, row int) []MenuItem {
	n := len(*panel.rows(a, tab))
	a.rowSel = &rowSelection{pane: panel, tab: tab, row: row}
	change := func(callback func(rows []model.KV) []model.KV) func() {
		return func() {
			a.commitEdit()
			a.applyRows(panel, tab, callback)
		}
	}
	items := append(a.copyRowItems(*panel.rows(a, tab), row, false), append([]MenuItem{menuSeparator}, []MenuItem{
		{Label: "Insert row above", Run: change(func(rows []model.KV) []model.KV { return model.InsertRow(rows, row, model.KV{Enabled: true}) })},
		{Label: "Insert row below", Run: change(func(rows []model.KV) []model.KV { return model.InsertRow(rows, row+1, model.KV{Enabled: true}) })},
		{Label: "Duplicate", Run: change(func(rows []model.KV) []model.KV { return model.DuplicateRow(rows, row) })},
		{Label: "Move up", Disabled: row == 0, Run: change(func(rows []model.KV) []model.KV { m, _ := model.MoveRow(rows, row, -1); return m })},
		{Label: "Move down", Disabled: row >= n-1, Run: change(func(rows []model.KV) []model.KV { m, _ := model.MoveRow(rows, row, 1); return m })},
		menuSeparator,
		{Label: "Enable all", Run: change(func(rows []model.KV) []model.KV { return model.SetAll(rows, true) })},
		{Label: "Disable all", Run: change(func(rows []model.KV) []model.KV { return model.SetAll(rows, false) })},
		menuSeparator,
		{Label: "Delete", Run: change(func(rows []model.KV) []model.KV { return model.DeleteRow(rows, row) })},
	}...)...)
	return append(items, a.secretItems(panel, row)...)
}

// secretItems is the menu entry that marks a variable of the active
// environment as a secret; other tables have none.
func (a *App) secretItems(panel pane, row int) []MenuItem {
	e := a.activeEnv()
	if panel.label != "Vars" || e == nil || row >= len(e.Vars) {
		return nil
	}
	label := "Mark as secret"
	if e.Vars[row].Secret {
		label = "Not a secret"
	}
	return []MenuItem{menuSeparator, {Label: label, Run: func() { a.toggleSecret(row) }}}
}

// copyRowItems copy parts of table row index to the clipboard. They are the
// whole menu of a read-only table and lead the menu of an editable one.
func (a *App) copyRowItems(rows []model.KV, index int, withAll bool) []MenuItem {
	if index < 0 || index >= len(rows) {
		return nil
	}
	r := rows[index]
	copyText := func(text string) func() {
		return func() { _ = a.ctx.Clipboard.Write(text) } // the clipboard reports its own failures
	}
	var all strings.Builder
	for _, h := range rows {
		all.WriteString(h.Name + ": " + h.Value + "\n")
	}
	items := []MenuItem{
		{Label: "Copy value", Run: copyText(r.Value)},
		{Label: "Copy name", Run: copyText(r.Name)},
		{Label: "Copy as Name: Value", Run: copyText(r.Name + ": " + r.Value)},
	}
	if withAll {
		items = append(items, MenuItem{Label: "Copy all headers", Run: copyText(all.String())})
	}
	return items
}

// ---- the response ------------------------------------------------------------------

func (a *App) responseMenuItems(tab *reqTab) []MenuItem {
	noBody := tab.Resp == nil || tab.Resp.Err != ""
	return []MenuItem{
		{Label: "Copy", Shortcut: "Ctrl+C", Disabled: !tab.view.Selected(), Run: func() {
			_ = a.ctx.Clipboard.Write(tab.view.SelectedText())
		}},
		{Label: "Copy body", Run: func() { a.copyBody(tab) }},
		{Label: "Save body...", Disabled: noBody, Run: func() { a.saveBody(tab) }},
		{Label: "Open in browser", Disabled: noBody, Run: func() { a.openInBrowser(tab) }},
		{Label: "Copy as cURL", Run: func() { a.copyCurl(tab) }},
		menuSeparator,
		{Label: "Select all", Shortcut: "Ctrl+A", Run: func() { tab.view.SelectLines(0, len(tab.view.Lines)-1) }},
		{Label: "Find", Shortcut: a.shortcutHint("find"), Run: a.openFind},
	}
}
