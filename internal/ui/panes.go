package ui

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	. "github.com/gascoigne-ai/gascoigne-workshop/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/widget"
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
	code     func(t *reqTab) *w.CodeView              // paneCode
	note     func(t *reqTab) string                   // paneNote
	empty    func(t *reqTab) (msg string, isErr bool) // shown instead of the content
}

func paramsChanged(t *reqTab) {
	t.req.URL = model.WithParams(t.req.URL, t.req.Params)
	t.url = w.NewTextField(t.req.URL)
}

var requestPanes = []pane{
	{label: "Params", kind: paneTable, title: "Query", editable: true, changed: paramsChanged,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.req.Params }},
	{label: "Body", kind: paneCode,
		code: func(t *reqTab) *w.CodeView { return &t.bodyView },
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
		code: func(t *reqTab) *w.CodeView { return &t.view }},
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

func (a *App) tabItems(panes []pane, t *reqTab) []TabItem {
	items := make([]TabItem, len(panes))
	for i, p := range panes {
		items[i] = TabItem{Label: p.label, Badge: p.badge(a, t)}
	}
	return items
}

func paneEmpty(p pane, t *reqTab) (string, bool) {
	if p.empty == nil {
		return "", false
	}
	return p.empty(t)
}

// paneBuilders turns a pane into its view, by kind.
var paneBuilders = map[paneKind]func(a *App, p pane, t *reqTab) Node{
	paneCode:  (*App).codePane,
	paneNote:  (*App).notePane,
	paneTable: (*App).tablePane,
}

// paneBody is what fills the area under a pane's tab strip.
func (a *App) paneBody(p pane, t *reqTab) Node {
	if msg, isErr := paneEmpty(p, t); msg != "" {
		return a.messagePane(p, msg, isErr)
	}
	return paneBuilders[p.kind](a, p, t)
}

// messagePane shows a status line ("Sending...", an error) in place of
// the pane's content.
func (a *App) messagePane(p pane, msg string, isErr bool) Node {
	color, bg := w.Muted, w.Bg
	if isErr {
		color = w.Error
	}
	if p.kind == paneCode {
		bg = w.Sunken
	}
	return Col(Grow(1), Pad(w.Pad), Bg(bg),
		Lines(msg, color),
	)
}

func (a *App) codePane(p pane, t *reqTab) Node {
	return Code(p.code(t), Grow(1), Bg(w.Sunken))
}

func (a *App) notePane(p pane, t *reqTab) Node {
	return Col(Grow(1), Pad(w.Pad),
		Lines(p.note(t), w.Text),
	)
}

func (a *App) tablePane(p pane, t *reqTab) Node {
	rows := p.rows(a, t)
	spec := TableSpec{
		Columns: []string{"Name", "Value"},
		Weights: []float64{35, 65},
		Rows:    kvRows(*rows, false),
	}
	if p.editable {
		spec.Editable = true
		spec.Columns = []string{"Name", "Value", "Description"}
		spec.Weights = []float64{28, 40, 32}
		spec.Rows = kvRows(*rows, true)
		spec.Placeholder = spec.Columns
		spec.OnToggle = func(i int) {
			(*rows)[i].Enabled = !(*rows)[i].Enabled
			a.rowsChanged(p, t)
		}
		spec.OnDelete = func(i int) {
			*rows = append((*rows)[:i], (*rows)[i+1:]...)
			a.rowsChanged(p, t)
		}
		spec.OnEditCell = func(row, col int) { a.beginEdit(p, t, row, col) }
		if e := a.edit; e != nil && e.pane.label == p.label && e.tab == t {
			spec.Editing = &TableEdit{Row: e.row, Col: e.col, Field: &e.field, OnEnter: a.commitEdit, OnTab: a.tabEdit}
		}
	}
	return Col(Grow(1), Pad(w.Pad),
		If(p.title != "", Row(H(w.RowH), Caption(p.title))),
		Table(spec),
	)
}

func (a *App) rowsChanged(p pane, t *reqTab) {
	if p.changed != nil {
		p.changed(t)
	}
}
