package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/textfmt"
	"strconv"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

type paneKind int

const (
	paneTable paneKind = iota
	paneCode
	paneNote
	paneBody
	paneSettings
	paneAuth
	paneResponse
	paneMessages
)

// pane describes one tab of the request or response area. The tables
// below are the whole definition; there is no per-tab switch.
type pane struct {
	label          string
	kind           paneKind
	title          string                                   // caption above a table
	rows           func(a *App, t *reqTab) *[]model.KV      // paneTable
	editable       bool                                     // table rows can be toggled and deleted
	changed        func(t *reqTab)                          // runs after an edit
	flagDuplicates bool                                     // mark rows that repeat a name
	code           func(t *reqTab) *w.CodeView              // paneCode
	note           func(t *reqTab) string                   // paneNote
	empty          func(t *reqTab) (msg string, isErr bool) // shown instead of the content
	caption        func(a *App) string                      // the caption when it depends on the app state
}

// captionOf is the caption above the pane's table.
func (p pane) captionOf(a *App) string {
	if p.caption != nil {
		return p.caption(a)
	}
	return p.title
}

func paramsChanged(t *reqTab) {
	t.req.URL = model.WithParams(t.req.URL, t.req.Params)
	t.url = w.NewTextField(t.req.URL)
}

var requestPanes = []pane{
	{label: "Params", kind: paneTable, title: "Query", editable: true, changed: paramsChanged,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.req.Params }},
	{label: "Body", kind: paneBody},
	{label: "Headers", kind: paneTable, title: "Headers", editable: true, flagDuplicates: true,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.req.Headers }},
	{label: "Auth", kind: paneAuth},
	{label: "Vars", kind: paneTable, title: "Collection variables", editable: true,
		caption: (*App).varsCaption,
		rows:    func(a *App, _ *reqTab) *[]model.KV { return a.varsRows() }},
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
	{label: "Response", kind: paneResponse, empty: responseEmpty},
	{label: "Headers", kind: paneTable, empty: noResponse,
		rows: func(_ *App, t *reqTab) *[]model.KV {
			if t.resp == nil {
				return &[]model.KV{}
			}
			return &t.resp.Headers
		}},
	{label: "Timeline", kind: paneNote, empty: noResponse,
		note: func(t *reqTab) string {
			return fmt.Sprintf("Duration  %s\nSize      %s", textfmt.Duration(t.resp.Duration), textfmt.Size(t.resp.Size))
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
	paneResponse: (*App).responsePane,
	paneMessages: (*App).messagesPane,
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
	if b := a.bulk; b != nil && b.tab == t && b.pane.label == p.label {
		return a.bulkView(p, t, pad)
	}
	rows := p.rows(a, t)
	spec := TableSpec{
		Widths:  a.widthsFor(p),
		Columns: []string{"Name", "Value"},
		Weights: []float64{35, 65},
		Rows:    kvRows(*rows, false, false),
	}
	if !p.editable {
		spec.OnRowMenu = func(row, x, y int) { a.openMenu(x, y, a.copyRowItems(*rows, row, true)) }
	}
	if p.editable {
		spec.Editable = true
		spec.Columns = []string{"Name", "Value", "Description"}
		spec.Weights = []float64{28, 40, 32}
		spec.Rows = kvRows(*rows, true, p.flagDuplicates)
		spec.Placeholder = spec.Columns
		spec.AllChecked = model.AllEnabled(*rows)
		spec.OnToggleAll = func() {
			a.commitEdit()
			a.applyRows(p, t, func(r []model.KV) []model.KV { return model.SetAll(r, !model.AllEnabled(r)) })
		}
		spec.OnToggle = func(i int) {
			a.commitEdit()
			a.applyRows(p, t, func(r []model.KV) []model.KV {
				r[i].Enabled = !r[i].Enabled
				return r
			})
		}
		spec.OnDelete = func(i int) {
			a.commitEdit()
			a.applyRows(p, t, func(r []model.KV) []model.KV { return model.DeleteRow(r, i) })
		}
		spec.OnEditCell = func(row, col int) { a.beginEdit(p, t, row, col) }
		spec.OnRowMenu = func(row, x, y int) { a.openMenu(x, y, a.rowMenuItems(p, t, row)) }
		if e := a.edit; e != nil && e.pane.label == p.label && e.tab == t {
			spec.Editing = &TableEdit{Row: e.row, Col: e.col, Field: &e.field, OnEnter: a.commitEdit, OnTab: a.tabEdit,
				OnMenu: func(x, y int) { a.openMenu(x, y, a.textMenu(&e.field, nil)) }}
		}
		if sel := a.rowSel; sel != nil && sel.tab == t && sel.pane.label == p.label && sel.row < len(*rows) {
			spec.Selected = &sel.row
		}
	}
	return Scroll(scroll, Pad(pad), Gap(6),
		a.tableCaption(p, t),
		Table(spec),
	)
}

// tableCaption is the row above a table: its title, and for editable
// tables the switch to bulk editing.
func (a *App) tableCaption(p pane, t *reqTab) Node {
	title := p.captionOf(a)
	if title == "" {
		return nil
	}
	return Row(H(w.RowH),
		Caption(title),
		Spacer(),
		If(p.editable, LinkButton(bulkLabel(a.bulk != nil), Tooltip("Edit all rows as text, one per line"), OnClick(func() { a.toggleBulk(p, t) }))),
	)
}

func bulkLabel(inBulk bool) string {
	if inBulk {
		return "Table"
	}
	return "Bulk edit"
}

// bulkView shows the rows as text under the same caption.
func (a *App) bulkView(p pane, t *reqTab, pad int) Node {
	return Col(Grow(1), Pad(pad), Gap(6),
		a.tableCaption(p, t),
		TextArea(a.bulk.area, Grow(1), Bg(w.Sunken), a.textMenuOpt(&a.bulk.area.TextField, nil)),
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
		sync := func() { t.req.Body = t.body.Text }
		return TextArea(t.body, Grow(1), Bg(w.Sunken), OnChange(sync), a.textMenuOpt(&t.body.TextField, sync))
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
		spans[i] = w.Span{Text: s.Text, Style: respview.Tokens[s.Kind]}
	}
	return spans
}

// settingsPane holds the per-request options: timeout and redirects.
func (a *App) settingsPane(_ pane, t *reqTab, scroll *ScrollState) Node {
	return Scroll(scroll, Pad(w.Pad), Gap(8),
		Row(H(w.RowH), Gap(12),
			Text("Timeout (seconds)", W(160)),
			Field(&t.timeout, W(80), Bg(w.Sunken),
				OnChange(func() { a.timeoutEdited(t) }),
				a.textMenuOpt(&t.timeout, func() { a.timeoutEdited(t) })),
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

// findCount is the counter shown in the find bar: "3/12", "No matches",
// "3 matches" when none is active, or nothing for an empty query.
func findCount(query string, matches, index int) string {
	switch {
	case query == "":
		return ""
	case matches == 0:
		return "No matches"
	case index < 0:
		return fmt.Sprintf("%d matches", matches)
	}
	return fmt.Sprintf("%d/%d", index+1, matches)
}

// responseEmpty is noResponse plus the case of a body that is not text.
func responseEmpty(t *reqTab) (string, bool) {
	if msg, isErr := noResponse(t); msg != "" {
		return msg, isErr
	}
	if respview.LooksBinary(t.resp.Body) {
		return fmt.Sprintf("Binary body, %s (not shown)", textfmt.Size(len(t.resp.Body))), false
	}
	return "", false
}

// responsePane is the response body with its toolbar and find bar.
func (a *App) responsePane(_ pane, t *reqTab, _ *ScrollState) Node {
	return Col(Grow(1),
		Spacer(H(w.Pad)), // same top inset as the request pane, so first rows line up
		Row(H(w.RowH+8), PadX(w.Pad), Gap(6), AlignItems(AlignCenter), BorderBottom(w.Border),
			a.viewChip("Compact", t.compact, 72, func() { a.setView(t, false, true) }),
			a.viewChip("Pretty", !t.raw && !t.compact, 56, func() { a.setView(t, false, false) }),
			a.viewChip("Raw", t.raw, 44, func() { a.setView(t, true, false) }),
			Spacer(),
			IconButton(SearchIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(a.openFind), Tooltip(a.hint("Find in response", "find"))),
			IconButton(CopyIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(func() { a.copyBody(t) }), Tooltip("Copy the response body")),
			IconButton(PromptIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(func() { a.copyCurl(t) }), Tooltip("Copy the request as a cURL command")),
		),
		If(t.findOpen, a.findBar(t)),
		Code(&t.view, Grow(1), Bg(w.Sunken), OnRightClick(func(x, y int) { a.openMenu(x, y, a.responseMenuItems(t)) })),
	)
}

func (a *App) viewChip(label string, selected bool, width int, onClick func()) Node {
	if selected {
		return PrimaryButton(label, W(width), H(w.CompactH), OnClick(onClick))
	}
	return Button(label, W(width), H(w.CompactH), OnClick(onClick))
}

func (a *App) findBar(t *reqTab) Node {
	return Row(H(w.RowH), PadX(w.Pad), Gap(6), BorderBottom(w.Border),
		Field(&t.find, Grow(1), Placeholder("Find"),
			OnChange(func() { a.runFind(t) }),
			OnEnter(func() { a.findEnter(t) }),
			a.textMenuOpt(&t.find, func() { a.runFind(t) }),
		),
		Text(findCount(t.find.Text, len(t.view.Matches()), t.matchIdx), Small(), Muted(), W(64)),
		IconButton(ChevronUp(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(func() { a.findStep(t, -1) }), Tooltip("Previous match")),
		IconButton(ChevronDown(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(func() { a.findStep(t, 1) }), Tooltip("Next match")),
		IconButton(Cross(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(func() { a.closeFind(t) }), Tooltip("Close search (Esc)")),
	)
}

// setView picks the response view: pretty-printed, as received, or compact.
func (a *App) setView(t *reqTab, raw, compact bool) {
	t.raw, t.compact = raw, compact
	a.refreshView(t)
}

// refreshView rebuilds the response view for the current mode and
// redoes an open search. Bodies that are not text are not loaded.
func (a *App) refreshView(t *reqTab) {
	switch {
	case t.resp == nil || respview.LooksBinary(t.resp.Body):
		t.view.SetLines(nil)
	case t.raw:
		t.view.SetLines(respview.RawLines(t.resp.Body))
	case t.compact:
		t.compactAll = respview.CompactLines(t.resp.Body)
		a.showCompact(t)
		return // showCompact searches too
	default:
		t.view.SetLines(respview.CodeLines(t.resp.Body))
	}
	if t.findOpen {
		a.runFind(t)
	}
}

func (a *App) copyBody(t *reqTab) {
	if t.resp != nil {
		_ = a.ctx.Clipboard.Write(string(t.resp.Body)) // the clipboard reports its own failures
	}
}

func (a *App) copyCurl(t *reqTab) {
	_ = a.ctx.Clipboard.Write(proto.Curl(t.req.Expanded(a.vars())))
}

// openFind shows the find bar for the current response and focuses it.
func (a *App) openFind() {
	a.commitEdit()
	t := a.cur()
	t.respTab = 0
	t.findOpen = true
	t.find.Focused = true
	t.find.SelectAll()
	a.runFind(t)
}

// showCompact fills the view with the compact lines: all of them with
// the deep blocks folded, or, while a search has text, only the matching
// lines with their children and the blocks around them (filterCompact).
func (a *App) showCompact(t *reqTab) {
	if t.findOpen && t.find.Text != "" {
		t.view.SetLines(respview.Filter(t.compactAll, t.find.Text))
	} else {
		t.view.SetLines(t.compactAll)
		respview.FoldDeep(&t.view, respview.CompactFoldDepth)
	}
	if t.findOpen {
		a.searchView(t)
	}
}

// runFind searches for the typed text and goes to the first match. The
// compact view is filtered to the matches first.
func (a *App) runFind(t *reqTab) {
	if t.compact && t.resp != nil && !respview.LooksBinary(t.resp.Body) {
		a.showCompact(t)
		return
	}
	a.searchView(t)
}

// searchView highlights the matches in the view and reveals the first.
func (a *App) searchView(t *reqTab) {
	matches := t.view.Find(t.find.Text)
	t.matchIdx = -1
	if len(matches) > 0 {
		t.matchIdx = 0
		t.view.RevealLine(matches[0].Line)
	}
	t.view.SetMatches(matches, t.matchIdx)
}

// findStep moves to the next (dir 1) or previous (dir -1) match, wrapping.
func (a *App) findStep(t *reqTab, dir int) {
	matches := t.view.Matches()
	if len(matches) == 0 {
		return
	}
	next := (t.matchIdx + dir + len(matches)) % len(matches)
	if t.matchIdx < 0 && dir < 0 {
		next = len(matches) - 1
	}
	t.matchIdx = next
	t.view.SetMatches(matches, next)
	t.view.RevealLine(matches[next].Line)
}

// findEnter is Enter in the find field: next match, previous with Shift.
func (a *App) findEnter(t *reqTab) {
	dir := 1
	if a.ctx.In.Shift {
		dir = -1
	}
	a.findStep(t, dir)
}

func (a *App) closeFind(t *reqTab) {
	t.findOpen = false
	t.find.Focused = false
	t.view.SetMatches(nil, -1)
	if t.compact && t.resp != nil && !respview.LooksBinary(t.resp.Body) {
		a.showCompact(t) // leave the filtered view
	}
}
