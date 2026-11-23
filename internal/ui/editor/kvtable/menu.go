package kvtable

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var separator = widget.MenuItem{Separator: true}

// rowMenuItems is the menu of row of an editable table: the copy entries,
// then the row operations, each one the same undoable change its shortcut
// or button makes. Opening the menu selects the row.
func (c *Component) rowMenuItems(ctx *core.Ctx, table logic.Table, row int) []widget.MenuItem {
	rows := *c.rows(table)
	n := len(rows)
	c.Select(table, row)
	change := func(callback func(rows []request.KV) []request.KV) func() {
		return func() {
			c.Commit()
			c.applyRows(table, callback)
		}
	}
	items := append(copyRowItems(ctx, rows, row, false), append([]widget.MenuItem{separator}, []widget.MenuItem{
		{Label: "Insert row above", Run: change(func(rows []request.KV) []request.KV { return request.InsertRow(rows, row, request.KV{Enabled: true}) })},
		{Label: "Insert row below", Run: change(func(rows []request.KV) []request.KV { return request.InsertRow(rows, row+1, request.KV{Enabled: true}) })},
		{Label: "Duplicate", Run: change(func(rows []request.KV) []request.KV { return request.DuplicateRow(rows, row) })},
		{Label: "Move up", Disabled: row == 0, Run: change(func(rows []request.KV) []request.KV { m, _ := request.MoveRow(rows, row, -1); return m })},
		{Label: "Move down", Disabled: row >= n-1, Run: change(func(rows []request.KV) []request.KV { m, _ := request.MoveRow(rows, row, 1); return m })},
		separator,
		{Label: "Enable all", Run: change(func(rows []request.KV) []request.KV { return request.SetAll(rows, true) })},
		{Label: "Disable all", Run: change(func(rows []request.KV) []request.KV { return request.SetAll(rows, false) })},
		separator,
		{Label: "Delete", Run: change(func(rows []request.KV) []request.KV { return request.DeleteRow(rows, row) })},
	}...)...)
	return append(items, c.secretItems(table, row)...)
}

// secretItems is the menu entry that marks a variable of the active
// environment as a secret; other tables have none.
func (c *Component) secretItems(table logic.Table, row int) []widget.MenuItem {
	secret, ok := c.deps.Secrets.Secret(table, row)
	if !ok {
		return nil
	}
	return []widget.MenuItem{separator, {Label: logic.SecretLabel(secret), Run: func() { c.deps.Secrets.ToggleSecret(row) }}}
}

// copyRowItems copy parts of table row index to the clipboard. They are the
// whole menu of a read-only table and lead the menu of an editable one.
func copyRowItems(ctx *core.Ctx, rows []request.KV, index int, withAll bool) []widget.MenuItem {
	if index < 0 || index >= len(rows) {
		return nil
	}
	row := rows[index]
	copyText := func(text string) func() {
		return func() { _ = ctx.Clipboard.Write(text) } // the clipboard reports its own failures
	}
	var all strings.Builder
	for _, each := range rows {
		all.WriteString(each.Name + ": " + each.Value + "\n")
	}
	items := []widget.MenuItem{
		{Label: "Copy value", Run: copyText(row.Value)},
		{Label: "Copy name", Run: copyText(row.Name)},
		{Label: "Copy as Name: Value", Run: copyText(row.Name + ": " + row.Value)},
	}
	if withAll {
		items = append(items, widget.MenuItem{Label: "Copy all headers", Run: copyText(all.String())})
	}
	return items
}
