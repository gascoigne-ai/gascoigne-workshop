package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"net/url"

	"net/http/cookiejar"

	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Right-click menus for tabs, table rows, text widgets and the response.
// The tree's own menu is in treeops.go. Every item does what its button
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

// closeOthers closes every tab but tab i, which becomes the active one.
func (a *App) closeOthers(index int) { a.updateTabs(tabs.CloseOthers{Index: index}) }

func (a *App) closeRightOf(index int) { a.updateTabs(tabs.CloseRightOf{Index: index}) }

// duplicateTab copies the tab's request next to the original in the
// tree (or into the root when it is not in the tree) and opens the copy.
func (a *App) duplicateTab(index int) {
	req := a.list.Tabs[index].Req
	owner, ok := a.entryFor(req)
	if !ok { // nothing is open: the copy is a tab of its own
		clone := req.Clone()
		clone.Name += " copy"
		a.openRequest(&clone)
		return
	}
	cp, _ := owner.Collection.Folder.Duplicate(req).(*model.Request)
	if cp == nil {
		clone := req.Clone()
		clone.Name += " copy"
		cp = owner.Collection.Folder.AddRequest(clone)
	}
	a.openRequest(cp)
}

// ---- table rows ---------------------------------------------------------------

func (a *App) rowMenuItems(pane pane, reqTab *reqTab, row int) []MenuItem {
	n := len(*pane.rows(a, reqTab))
	a.rowSel = &rowSelection{pane: pane, tab: reqTab, row: row}
	change := func(callback func(rows []model.KV) []model.KV) func() {
		return func() {
			a.commitEdit()
			a.applyRows(pane, reqTab, callback)
		}
	}
	items := append(a.copyRowItems(*pane.rows(a, reqTab), row, false), append([]MenuItem{menuSeparator}, []MenuItem{
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
	return append(items, a.secretItems(pane, row)...)
}

// secretItems is the menu entry that marks a variable of the active
// environment as a secret; other tables have none.
func (a *App) secretItems(pane pane, row int) []MenuItem {
	e := a.activeEnv()
	if pane.label != "Vars" || e == nil || row >= len(e.Vars) {
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

// Size is the window's size in logical pixels; with the methods below it
// makes the App the host of the features that live in their own packages.
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

// OpenRequest focuses the request at path of the collection saved in
// folder and reports whether there is one. A row without a folder is of a
// collection not saved, or older than the folder in a row, when the one
// collection open was the first.
func (a *App) OpenRequest(folder, path string) bool {
	var found *model.Request
	for _, entry := range a.sentCandidates(folder) {
		walkRequests(&entry.Collection.Folder, "", func(candidate string, req *model.Request) {
			if candidate == path && found == nil {
				found = req
			}
		})
	}
	if found == nil {
		return false
	}
	a.openRequest(found)
	return true
}

// sentCandidates are the open collections a sent row of folder may name,
// in the order they are searched.
func (a *App) sentCandidates(folder string) []collections.Entry {
	if folder != "" {
		entry, ok := a.open.ByDir(folder)
		if !ok {
			return nil
		}
		return []collections.Entry{entry}
	}
	var candidates []collections.Entry
	for _, entry := range a.open.Entries {
		if entry.Dir == "" {
			candidates = append(candidates, entry)
		}
	}
	if len(a.open.Entries) > 0 {
		candidates = append(candidates, a.open.Entries[0])
	}
	return candidates
}

// Badge is the method label shown before a request's name.
func (a *App) Badge(protocol, method string) Node {
	return Text(requestBadge(protocol, method), Small(), Color(appearance.MethodToken(method)), W(badgeW))
}

// ---- the response ------------------------------------------------------------------

func (a *App) responseMenuItems(tab *reqTab) []MenuItem {
	return []MenuItem{
		{Label: "Copy", Shortcut: "Ctrl+C", Disabled: !tab.view.Selected(), Run: func() {
			_ = a.ctx.Clipboard.Write(tab.view.SelectedText())
		}},
		{Label: "Copy body", Run: func() { a.copyBody(tab) }},
		{Label: "Save body...", Disabled: tab.Resp == nil || tab.Resp.Err != "", Run: func() { a.saveBody(tab) }},
		{Label: "Open in browser", Disabled: tab.Resp == nil || tab.Resp.Err != "", Run: func() { a.openInBrowser(tab) }},
		{Label: "Copy as cURL", Run: func() { a.copyCurl(tab) }},
		menuSeparator,
		{Label: "Select all", Shortcut: "Ctrl+A", Run: func() { tab.view.SelectLines(0, len(tab.view.Lines)-1) }},
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
func (a *App) showCookies(tab *reqTab) {
	u, err := url.Parse(tab.Req.Expanded(a.varsFor(tab.Req)).URL)
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
