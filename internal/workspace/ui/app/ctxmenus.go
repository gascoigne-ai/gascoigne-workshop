package ui

import (
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"net/url"

	"net/http/cookiejar"

	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Right-click menus for tabs, table rows, text widgets and the response.
// The tree's own menu is in treeops.go. Every item does what its button
// or shortcut does, through the same code.

var menuSeparator = MenuItem{Separator: true}

// ---- request tabs ------------------------------------------------------------

func (a *App) tabMenuItems(i int) []MenuItem {
	t := a.widgets[a.list.Tabs[i]]
	return []MenuItem{
		{Label: "Close", Run: func() { a.closeTab(i) }},
		{Label: "Close others", Disabled: len(a.list.Tabs) == 1, Run: func() { a.closeOthers(i) }},
		{Label: "Close tabs to the right", Disabled: i == len(a.list.Tabs)-1, Run: func() { a.closeRightOf(i) }},
		menuSeparator,
		{Label: "Duplicate", Run: func() { a.duplicateTab(i) }},
		{Label: "Copy URL", Run: func() { a.copyRequestURL(t.Req) }},
		{Label: "Cookies...", Run: func() { a.showCookies(t) }},
	}
}

// closeOthers closes every tab but tab i, which becomes the active one.
func (a *App) closeOthers(i int) { a.updateTabs(tabs.CloseOthers{Index: i}) }

func (a *App) closeRightOf(i int) { a.updateTabs(tabs.CloseRightOf{Index: i}) }

// duplicateTab copies the tab's request next to the original in the
// tree (or into the root when it is not in the tree) and opens the copy.
func (a *App) duplicateTab(i int) {
	req := a.list.Tabs[i].Req
	cp, _ := a.col.Folder.Duplicate(req).(*model.Request)
	if cp == nil {
		clone := req.Clone()
		clone.Name += " copy"
		cp = a.col.Folder.AddRequest(clone)
	}
	a.openRequest(cp)
}

// ---- table rows ---------------------------------------------------------------

func (a *App) rowMenuItems(p pane, t *reqTab, row int) []MenuItem {
	n := len(*p.rows(a, t))
	a.rowSel = &rowSelection{pane: p, tab: t, row: row}
	change := func(f func(rows []model.KV) []model.KV) func() {
		return func() {
			a.commitEdit()
			a.applyRows(p, t, f)
		}
	}
	items := append(a.copyRowItems(*p.rows(a, t), row, false), append([]MenuItem{menuSeparator}, []MenuItem{
		{Label: "Insert row above", Run: change(func(r []model.KV) []model.KV { return model.InsertRow(r, row, model.KV{Enabled: true}) })},
		{Label: "Insert row below", Run: change(func(r []model.KV) []model.KV { return model.InsertRow(r, row+1, model.KV{Enabled: true}) })},
		{Label: "Duplicate", Run: change(func(r []model.KV) []model.KV { return model.DuplicateRow(r, row) })},
		{Label: "Move up", Disabled: row == 0, Run: change(func(r []model.KV) []model.KV { m, _ := model.MoveRow(r, row, -1); return m })},
		{Label: "Move down", Disabled: row >= n-1, Run: change(func(r []model.KV) []model.KV { m, _ := model.MoveRow(r, row, 1); return m })},
		menuSeparator,
		{Label: "Enable all", Run: change(func(r []model.KV) []model.KV { return model.SetAll(r, true) })},
		{Label: "Disable all", Run: change(func(r []model.KV) []model.KV { return model.SetAll(r, false) })},
		menuSeparator,
		{Label: "Delete", Run: change(func(r []model.KV) []model.KV { return model.DeleteRow(r, row) })},
	}...)...)
	return append(items, a.secretItems(p, row)...)
}

// secretItems is the menu entry that marks a variable of the active
// environment as a secret; other tables have none.
func (a *App) secretItems(p pane, row int) []MenuItem {
	e := a.activeEnv()
	if p.label != "Vars" || e == nil || row >= len(e.Vars) {
		return nil
	}
	label := "Mark as secret"
	if e.Vars[row].Secret {
		label = "Not a secret"
	}
	return []MenuItem{menuSeparator, {Label: label, Run: func() { a.toggleSecret(row) }}}
}

// copyRowItems copy parts of table row i to the clipboard. They are the
// whole menu of a read-only table and lead the menu of an editable one.
func (a *App) copyRowItems(rows []model.KV, i int, withAll bool) []MenuItem {
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

// ---- text widgets ---------------------------------------------------------------

// textMenu is the menu of a text field or area. changed, when given,
// runs after the menu changed the text, so the model follows.
func (a *App) textMenu(f *TextField, changed func()) []MenuItem {
	done := func() {
		if changed != nil {
			changed()
		}
	}
	hasSel := f.SelectedText() != "" && !f.Masked // secrets never reach the clipboard
	return []MenuItem{
		{Label: "Undo", Shortcut: "Ctrl+Z", Disabled: !f.CanUndo(), Run: func() {
			if f.Undo() {
				done()
			}
		}},
		{Label: "Redo", Shortcut: "Ctrl+Y", Disabled: !f.CanRedo(), Run: func() {
			if f.Redo() {
				done()
			}
		}},
		menuSeparator,
		{Label: "Cut", Shortcut: "Ctrl+X", Disabled: !hasSel, Run: func() {
			_ = a.ctx.Clipboard.Write(f.SelectedText())
			if f.DeleteSelection() {
				done()
			}
		}},
		{Label: "Copy", Shortcut: "Ctrl+C", Disabled: !hasSel, Run: func() { _ = a.ctx.Clipboard.Write(f.SelectedText()) }},
		{Label: "Paste", Shortcut: "Ctrl+V", Run: func() {
			if text, err := a.ctx.Clipboard.Read(); err == nil && f.Insert(text) {
				done()
			}
		}},
		menuSeparator,
		{Label: "Select all", Shortcut: "Ctrl+A", Disabled: f.Text == "", Run: f.SelectAll},
	}
}

// textMenuOpt makes an element open the text menu of f on right click.
// Size is the window's size in logical pixels; with TextMenu it makes the
// App the host of the features that live in their own packages.
func (a *App) Size() (width, height int) { return a.w, a.h }

// SentStore, Toast (below), SetStatus, OpenRequest and Badge make the App
// the host of the sent-request log.
func (a *App) SentStore() port.SentStore {
	store := a.historyStore()
	if store == nil {
		return nil
	}
	return store.Sent()
}

// Toast shows a message over the window.
func (a *App) Toast(text string, isErr bool) { a.toast(text, isErr) }

// SetStatus replaces the status line.
func (a *App) SetStatus(text string) { a.status = text }

// OpenRequest focuses the request at path of the collection and reports
// whether there is one.
func (a *App) OpenRequest(path string) bool {
	var found *model.Request
	walkRequests(&a.col.Folder, "", func(candidate string, req *model.Request) {
		if candidate == path {
			found = req
		}
	})
	if found == nil {
		return false
	}
	a.openRequest(found)
	return true
}

// Badge is the method label shown before a request's name.
func (a *App) Badge(protocol, method string) Node {
	return Text(requestBadge(protocol, method), Small(), Color(appearance.MethodToken(method)), W(badgeW))
}

// TextMenu is the option that gives field the right-click text menu.
func (a *App) TextMenu(field *TextField, changed func()) Option { return a.textMenuOpt(field, changed) }

func (a *App) textMenuOpt(f *TextField, changed func()) Option {
	return OnRightClick(func(x, y int) { a.openMenu(x, y, a.textMenu(f, changed)) })
}

// ---- the response ------------------------------------------------------------------

func (a *App) responseMenuItems(t *reqTab) []MenuItem {
	return []MenuItem{
		{Label: "Copy", Shortcut: "Ctrl+C", Disabled: !t.view.Selected(), Run: func() {
			_ = a.ctx.Clipboard.Write(t.view.SelectedText())
		}},
		{Label: "Copy body", Run: func() { a.copyBody(t) }},
		{Label: "Save body...", Disabled: t.Resp == nil || t.Resp.Err != "", Run: func() { a.saveBody(t) }},
		{Label: "Open in browser", Disabled: t.Resp == nil || t.Resp.Err != "", Run: func() { a.openInBrowser(t) }},
		{Label: "Copy as cURL", Run: func() { a.copyCurl(t) }},
		menuSeparator,
		{Label: "Select all", Shortcut: "Ctrl+A", Run: func() { t.view.SelectLines(0, len(t.view.Lines)-1) }},
		{Label: "Find", Shortcut: a.shortcutHint("find"), Run: a.openFind},
	}
}

// clearCookies forgets every cookie the servers set. The jar is replaced,
// so connections made before keep nothing.
func (a *App) clearCookies() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		a.toast(err.Error(), true)
		return
	}
	a.jar = jar
	a.reg = a.newRegistry()
	a.toast("Cookies cleared", false)
}

// newRegistry builds the protocols from the cookie jar, the network
// settings (proxy, certificates) and the app's sampling dialog.
func (a *App) newRegistry() reqport.Registry {
	if a.protocols == nil {
		return reqport.Registry{}
	}
	s := a.settings
	return a.protocols(reqport.ProtocolConfig{Jar: a.jar, Proxy: s.Proxy, CACert: s.CACert, ClientCert: s.ClientCert, ClientKey: s.ClientKey, Sampler: a.sampleViaDialog})
}

// showCookies lists the cookies the jar holds for the request's URL and
// offers to clear all of them.
func (a *App) showCookies(t *reqTab) {
	u, err := url.Parse(t.Req.Expanded(a.varsFor(t.Req)).URL)
	if err != nil || u.Host == "" {
		a.toast("The URL has no host to look up cookies for", true)
		return
	}
	var lines []string
	for _, c := range a.jar.Cookies(u) {
		lines = append(lines, c.Name+"="+c.Value)
	}
	body := "No cookies for " + u.Host + "."
	if len(lines) > 0 {
		body = "Sent to " + u.Host + ":\n" + strings.Join(lines, "\n")
	}
	a.confirm("Cookies", body, "Clear all cookies", a.clearCookies)
}
