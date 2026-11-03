package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	. "github.com/gascoigne-ai/gascoigne-workshop/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/widget"
)

type paneKind int

const (
	paneTable paneKind = iota
	paneCode
	paneNote
	paneBody
	paneSettings
	paneAuth
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
	{label: "Body", kind: paneBody},
	{label: "Headers", kind: paneTable, title: "Headers", editable: true,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.req.Headers }},
	{label: "Auth", kind: paneAuth},
	{label: "Vars", kind: paneTable, title: "Collection variables", editable: true,
		rows: func(a *App, _ *reqTab) *[]model.KV { return &a.col.Vars }},
	{label: "Settings", kind: paneSettings},
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
var paneBuilders = map[paneKind]func(a *App, p pane, t *reqTab, scroll *ScrollState) Node{
	paneCode:     (*App).codePane,
	paneNote:     (*App).notePane,
	paneBody:     (*App).bodyPane,
	paneSettings: (*App).settingsPane,
	paneAuth:     (*App).authPane,
	paneTable:    (*App).tablePane,
}

// paneBody is what fills the area under a pane's tab strip.
func (a *App) paneBody(p pane, t *reqTab, scroll *ScrollState) Node {
	if msg, isErr := paneEmpty(p, t); msg != "" {
		return a.messagePane(p, msg, isErr)
	}
	return paneBuilders[p.kind](a, p, t, scroll)
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

func (a *App) codePane(p pane, t *reqTab, _ *ScrollState) Node {
	return Code(p.code(t), Grow(1), Bg(w.Sunken))
}

func (a *App) notePane(p pane, t *reqTab, scroll *ScrollState) Node {
	return Scroll(scroll, Pad(w.Pad),
		Lines(p.note(t), w.Text),
	)
}

func (a *App) tablePane(p pane, t *reqTab, scroll *ScrollState) Node {
	return a.tableView(p, t, scroll, w.Pad)
}

// tableView is a pane's table with the given padding around it.
func (a *App) tableView(p pane, t *reqTab, scroll *ScrollState, pad int) Node {
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
	return Scroll(scroll, Pad(pad),
		If(p.title != "", Row(H(w.RowH), Caption(p.title))),
		Table(spec),
	)
}

func (a *App) rowsChanged(p pane, t *reqTab) {
	if p.changed != nil {
		p.changed(t)
	}
}

// jsonStatus describes whether text is valid JSON, for the body pane.
// An empty body has nothing to report.
func jsonStatus(text string) (msg string, valid bool) {
	if strings.TrimSpace(text) == "" {
		return "", false
	}
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return "Invalid JSON: " + err.Error(), false
	}
	return "Valid JSON", true
}

// bodyTypes are the choices of the body type row.
var bodyTypes = []struct {
	id, label string
	width     int
}{
	{model.BodyNone, "None", 64},
	{model.BodyJSON, "JSON", 64},
	{model.BodyText, "Text", 64},
	{model.BodyForm, "Form", 64},
	{model.BodyMultipart, "Multipart", 84},
}

// formPane is the table of form and multipart fields.
var formPane = pane{label: "Body", kind: paneTable, editable: true,
	rows: func(_ *App, t *reqTab) *[]model.KV { return &t.req.Form }}

// bodyPane is the body editor: a type chooser, then an editor that fits the type.
func (a *App) bodyPane(p pane, t *reqTab, scroll *ScrollState) Node {
	kind := t.req.BodyType
	isJSON := kind == model.BodyJSON
	status, valid := jsonStatus(t.body.Text)
	statusColor := w.Error
	if valid {
		statusColor = w.OK
	}
	return Col(Grow(1), Pad(w.Pad), Gap(8),
		Row(H(w.RowH), Gap(6),
			For(bodyTypes, func(_ int, bt struct {
				id, label string
				width     int
			}) Node {
				choose := OnClick(func() { a.setBodyType(t, bt.id) })
				if bt.id == kind {
					return PrimaryButton(bt.label, W(bt.width), choose)
				}
				return Button(bt.label, W(bt.width), choose)
			}),
			Spacer(),
			If(isJSON && status != "", Text(status, Small(), Color(statusColor), Ellipsis(), Optional())),
			If(isJSON, Button("Format", W(70), OnClick(func() { a.formatBody(t) }))),
		),
		a.bodyEditor(t, scroll),
	)
}

// bodyEditor is the part of the body pane under the type chooser.
func (a *App) bodyEditor(t *reqTab, scroll *ScrollState) Node {
	switch t.req.BodyType {
	case model.BodyJSON, model.BodyText:
		t.body.Styler = nil
		if t.req.BodyType == model.BodyJSON {
			t.body.Styler = jsonStyler
		}
		return TextArea(t.body, Grow(1), Bg(w.Sunken), OnChange(func() { t.req.Body = t.body.Text }))
	case model.BodyForm, model.BodyMultipart:
		return a.tableView(formPane, t, scroll, 0)
	}
	return Lines("This request has no body", w.Muted)
}

func (a *App) setBodyType(t *reqTab, id string) {
	a.commitEdit()
	t.req.BodyType = id
}

// formatBody re-indents a JSON body. Invalid JSON is left alone.
func (a *App) formatBody(t *reqTab) {
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(t.body.Text), "", "  "); err != nil {
		a.status = "Cannot format body: " + err.Error()
		return
	}
	t.body.SelectAll()
	t.body.Insert(out.String()) // as an edit, so it can be undone
	t.req.Body = t.body.Text
}

// jsonStyler colors one line of a JSON body.
func jsonStyler(line string) []w.Span {
	src := jsonview.Tokenize(line)
	spans := make([]w.Span, len(src))
	for i, s := range src {
		spans[i] = w.Span{Text: s.Text, Style: jsonTokens[s.Kind]}
	}
	return spans
}

// settingsPane holds the per-request options: timeout and redirects.
func (a *App) settingsPane(_ pane, t *reqTab, scroll *ScrollState) Node {
	return Scroll(scroll, Pad(w.Pad), Gap(8),
		Row(H(w.RowH), Gap(12),
			Text("Timeout (seconds)", W(160)),
			Field(&t.timeout, W(80), Bg(w.Sunken), Border(w.Border),
				OnChange(func() { a.timeoutEdited(t) })),
			Text(fmt.Sprintf("blank = default (%d s)", a.timeoutFor(model.Request{})/time.Second), Small(), Muted()),
		),
		Row(H(w.RowH), Gap(8), OnClick(func() { t.req.NoRedirects = !t.req.NoRedirects }),
			Checkbox(!t.req.NoRedirects),
			Text("Follow redirects"),
		),
	)
}

// timeoutEdited applies the typed timeout; text that is not a whole
// number of seconds is undone.
func (a *App) timeoutEdited(t *reqTab) {
	text := strings.TrimSpace(t.timeout.Text)
	if text == "" {
		t.req.TimeoutSec = 0
		return
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 0 {
		t.timeout = w.NewTextField(timeoutText(t.req.TimeoutSec))
		t.timeout.Focused = true
		return
	}
	t.req.TimeoutSec = n
}
