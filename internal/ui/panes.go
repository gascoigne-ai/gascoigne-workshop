package ui

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

type paneKind int

const (
	paneTable paneKind = iota
	paneCode
	paneNote
)

// pane describes one tab of the request or response area. The tables
// below are the whole definition; there is no per-tab switch.
type pane struct {
	label    string
	kind     paneKind
	title    string                                   // caption above a table
	rows     func(a *App, t *reqTab) *[]model.KV      // paneTable
	editable bool                                     // table rows can be toggled and deleted
	changed  func(t *reqTab)                          // runs after an edit
	code     func(t *reqTab) *CodeView                // paneCode
	note     func(t *reqTab) string                   // paneNote
	empty    func(t *reqTab) (msg string, isErr bool) // shown instead of the content
}

func paramsChanged(t *reqTab) {
	t.req.URL = model.WithParams(t.req.URL, t.req.Params)
	t.url = NewTextField(t.req.URL)
}

var requestPanes = []pane{
	{label: "Params", kind: paneTable, title: "Query", editable: true, changed: paramsChanged,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.req.Params }},
	{label: "Body", kind: paneCode,
		code: func(t *reqTab) *CodeView { return &t.bodyView },
		empty: func(t *reqTab) (string, bool) {
			if t.req.Body == "" {
				return "This request has no body", false
			}
			return "", false
		}},
	{label: "Headers", kind: paneTable, title: "Headers", editable: true,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.req.Headers }},
	{label: "Auth", kind: paneNote,
		note: func(*reqTab) string { return "No authentication configured" }},
	{label: "Vars", kind: paneTable, title: "Collection variables", editable: true,
		rows: func(a *App, _ *reqTab) *[]model.KV { return &a.col.Vars }},
}

func noResponse(t *reqTab) (string, bool) {
	switch {
	case t.sending:
		return "Sending...", false
	case t.resp == nil:
		return "Send the request to see the response", false
	case t.resp.Err != "":
		return t.resp.Err, true
	}
	return "", false
}

var responsePanes = []pane{
	{label: "Response", kind: paneCode, empty: noResponse,
		code: func(t *reqTab) *CodeView { return &t.view }},
	{label: "Headers", kind: paneTable, empty: noResponse,
		rows: func(_ *App, t *reqTab) *[]model.KV {
			if t.resp == nil {
				return &[]model.KV{}
			}
			return &t.resp.Headers
		}},
	{label: "Timeline", kind: paneNote, empty: noResponse,
		note: func(t *reqTab) string {
			return fmt.Sprintf("Duration  %s\nSize      %s", formatDuration(t.resp.Duration), formatSize(t.resp.Size))
		}},
	{label: "Tests", kind: paneNote,
		note: func(*reqTab) string { return "No tests" }},
}

// badge is the count shown on a pane's tab.
func (p pane) badge(a *App, t *reqTab) int {
	if p.kind != paneTable {
		return 0
	}
	rows := *p.rows(a, t)
	if !p.editable {
		return len(rows)
	}
	n := 0
	for _, r := range rows {
		if r.Enabled {
			n++
		}
	}
	return n
}

func (a *App) tabItems(panes []pane, t *reqTab) []tabItem {
	items := make([]tabItem, len(panes))
	for i, p := range panes {
		items[i] = tabItem{Label: p.label, Badge: p.badge(a, t)}
	}
	return items
}

// contentRect is the area below a pane tab strip, padded.
func (a *App) contentRect(r Rect) Rect {
	_, content := r.SplitTop(paneTabsH)
	return content.Inset(a.ctx.Th.Pad)
}

// codeRect is the full-bleed area of a code pane: the code view pads
// itself, so it spans the pane width below the tab strip.
func (a *App) codeRect(r Rect) Rect {
	_, below := r.SplitTop(paneTabsH)
	return below
}

// tableRect is the table area under the caption row of a content rect.
func (a *App) tableRect(content Rect, p pane) Rect {
	if p.title == "" {
		return content
	}
	_, table := content.SplitTop(a.ctx.Th.RowH)
	return table
}

func (a *App) updatePanes(t *reqTab, r Rect, panes []pane, active *int) {
	ctx := &a.ctx
	tabsR, _ := r.SplitTop(paneTabsH)
	if i := ctx.hitTab(tabsR, a.tabItems(panes, t)); i >= 0 {
		*active = i
	}
	p := panes[*active]
	if msg, _ := paneEmpty(p, t); msg != "" {
		return
	}
	content := a.contentRect(r)
	switch p.kind {
	case paneCode:
		ctx.updateCodeView(a.codeRect(r), p.code(t))
	case paneTable:
		if !p.editable {
			return
		}
		rows := p.rows(a, t)
		table := a.tableRect(content, p)
		i, act := ctx.hitTable(table, len(*rows), true)
		switch act {
		case tableToggle:
			(*rows)[i].Enabled = !(*rows)[i].Enabled
		case tableDelete:
			*rows = append((*rows)[:i], (*rows)[i+1:]...)
		default:
			return
		}
		if p.changed != nil {
			p.changed(t)
		}
	}
}

func paneEmpty(p pane, t *reqTab) (string, bool) {
	if p.empty == nil {
		return "", false
	}
	return p.empty(t)
}
