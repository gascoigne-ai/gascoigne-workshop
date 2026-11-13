package ui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/respview"
	"github.com/hajimehoshi/ebiten/v2"
	"strconv"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/jsonview"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

type paneKind int

const (
	paneTable paneKind = iota
	paneNote
	paneBody
	paneSettings
	paneAuth
	paneResponse
	paneMessages
	paneAssertions // the request side: the assertions being written
	paneResults    // the response side: what they found
	paneExplorer   // the response side of MCP: what the server offers
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
	editRequest(t, editor.ParamsChanged{})
	t.url = NewTextField(t.Req.URL)
}

var (
	paramsPane = pane{label: "Params", kind: paneTable, title: "Query", editable: true, changed: paramsChanged,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.Req.Params }}
	bodyPane    = pane{label: "Body", kind: paneBody}
	headersPane = pane{label: "Headers", kind: paneTable, title: "Headers", editable: true, flagDuplicates: true,
		rows: func(_ *App, t *reqTab) *[]model.KV { return &t.Req.Headers }}
	authPane = pane{label: "Auth", kind: paneAuth}
	varsPane = pane{label: "Vars", kind: paneTable, title: "Collection variables", editable: true,
		caption: (*App).varsCaption,
		rows:    func(a *App, _ *reqTab) *[]model.KV { return a.varsRows() }}
	testsPane    = pane{label: "Tests", kind: paneAssertions}
	settingsPane = pane{label: "Settings", kind: paneSettings}
)

// requestPanes are the request side of REST and of every protocol with no
// panes of its own.
var requestPanes = []pane{paramsPane, bodyPane, headersPane, authPane, varsPane, testsPane, settingsPane}

// connectionRequestPanes are the request side of a protocol that opens a
// connection: what it sends is composed in the Messages pane, so there is
// no Params, Body or Tests tab.
var connectionRequestPanes = []pane{headersPane, authPane, varsPane, settingsPane}

// requestPanesByProtocol lists the protocols whose request side differs
// from requestPanes.
var requestPanesByProtocol = map[string][]pane{
	"websocket": connectionRequestPanes,
	"sse":       connectionRequestPanes,
	"mcp":       connectionRequestPanes,
}

func noResponse(t *reqTab) (string, bool) {
	switch {
	case t.Sending:
		return "Sending...", false
	case t.Resp == nil:
		return "Send the request to see the response", false
	case t.Resp.Err != "":
		return t.Resp.Err, true
	}
	return "", false
}

var responsePanes = []pane{
	{label: "Response", kind: paneResponse, empty: responseEmpty},
	{label: "Headers", kind: paneTable, empty: noResponse,
		rows: func(_ *App, t *reqTab) *[]model.KV {
			if t.Resp == nil {
				return &[]model.KV{}
			}
			return &t.Resp.Headers
		}},
	{label: "Timeline", kind: paneNote, empty: noResponse,
		note: func(t *reqTab) string { return respview.Timeline(t.Resp) }},
	{label: "Tests", kind: paneResults},
}

// badge is the count shown on a pane's tab.
func (p pane) badge(a *App, t *reqTab) int {
	switch p.kind {
	case paneAssertions: // how many assertions are written
		n := 0
		for _, l := range strings.Split(t.Req.Tests, "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
				n++
			}
		}
		return n
	case paneResults: // how many failed
		n := 0
		for _, r := range t.testResults {
			if !r.OK {
				n++
			}
		}
		return n
	}
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
	paneNote:       (*App).notePane,
	paneBody:       (*App).bodyPane,
	paneSettings:   (*App).settingsPane,
	paneAuth:       (*App).authPane,
	paneResponse:   (*App).responsePane,
	paneMessages:   (*App).messagesPane,
	paneExplorer:   (*App).explorerPane,
	paneAssertions: (*App).assertionsPane,
	paneResults:    (*App).resultsPane,
	paneTable:      (*App).tablePane,
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
	return Col(Grow(1), Pad(w.Gutter), Bg(bg),
		Lines(msg, color),
	)
}

func (a *App) notePane(p pane, t *reqTab, scroll *ScrollState) Node {
	return Scroll(scroll, Pad(w.Gutter),
		Lines(p.note(t), w.Text),
	)
}

func (a *App) tablePane(p pane, t *reqTab, scroll *ScrollState) Node {
	return a.tableView(p, t, scroll, w.Gutter)
}

// tableView is a pane's table with the given padding around it.
func (a *App) tableView(p pane, t *reqTab, scroll *ScrollState, pad int) Node {
	if b := a.bulk; b != nil && b.tab == t && b.pane.label == p.label {
		return a.bulkView(p, t, pad)
	}
	rows := p.rows(a, t)
	spec := TableSpec{
		CellStyler: a.cellStyler(t.Req),
		Widths:     a.widthsFor(p),
		Columns:    []string{"Name", "Value"},
		Weights:    []float64{35, 65},
		Rows:       kvRows(*rows, false, false),
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
		spec.OnGrabRow = func(row int) { a.grabRow(p, t, row) }
		spec.OnDragOverRow = func(row int) { a.dragRowOver(row) }
		spec.OnEditCell = func(row, col int) { a.beginEdit(p, t, row, col) }
		spec.OnRowMenu = func(row, x, y int) { a.openMenu(x, y, a.rowMenuItems(p, t, row)) }
		if e := a.edit; e != nil && e.pane.label == p.label && e.tab == t {
			spec.Editing = &TableEdit{Row: e.row, Col: e.col, Field: &e.field, OnEnter: a.commitEdit, OnTab: a.tabEdit, Options: TextMenu(&e.field)}
		}
		if sel := a.rowSel; sel != nil && sel.tab == t && sel.pane.label == p.label && sel.row < len(*rows) {
			spec.Selected = &sel.row
		}
	}
	scope := a.tableCaptionScope(p, t, pad)
	scope["scroll"] = scroll
	scope["spec"] = spec
	return tableViewTemplate.MustBuild(scope)
}

//go:embed tableview.vml bulkview.vml
var tableViewSources embed.FS

// The table and its rows as text are two roots, so two templates; Go
// picks one in tableView.
var (
	tableViewTemplate = markup.MustLoad(tableViewSources, "tableview.vml")
	bulkViewTemplate  = markup.MustLoad(tableViewSources, "bulkview.vml")
)

// tableCaptionScope is what both templates show in the row above the
// rows: the pane's title, and for editable tables the switch to bulk
// editing. An empty title leaves the row out.
func (a *App) tableCaptionScope(p pane, t *reqTab, pad int) markup.Scope {
	return markup.Scope{
		"pad":        pad,
		"rowH":       w.RowH,
		"caption":    p.captionOf(a),
		"editable":   p.editable,
		"bulkLabel":  bulkLabel(a.bulk != nil),
		"toggleBulk": func() { a.toggleBulk(p, t) },
	}
}

func bulkLabel(inBulk bool) string {
	if inBulk {
		return "Table"
	}
	return "Bulk edit"
}

// bulkView shows the rows as text under the same caption.
func (a *App) bulkView(p pane, t *reqTab, pad int) Node {
	scope := a.tableCaptionScope(p, t, pad)
	scope["area"] = &a.bulk.area
	scope["areaText"] = &a.bulk.area.TextField
	return bulkViewTemplate.MustBuild(scope)
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
	{model.BodyGraphQL, "GraphQL", 80},
	{model.BodyFile, "File", 64},
}

// formPane is the table of form and multipart fields.
var formPane = pane{label: "Body", kind: paneTable, title: "Fields", editable: true,
	rows: func(_ *App, t *reqTab) *[]model.KV { return &t.Req.Form }}

// bodyPane is the body editor: a type chooser, then an editor that fits the type.
func (a *App) bodyPane(p pane, t *reqTab, scroll *ScrollState) Node {
	kind := t.Req.BodyType
	isJSON := kind == model.BodyJSON
	status, valid := jsonStatus(t.body.Text)
	statusColor := w.Error
	if valid {
		statusColor = w.OK
	}
	return Col(Grow(1), Pad(w.Gutter), Gap(8),
		Row(H(w.RowH), Gap(6),
			HScroll(&t.bodyTypesX, Grow(1), Gap(6), // the type chips scroll sideways when the pane is narrow
				For(bodyTypes, func(_ int, bt struct {
					id, label string
					width     int
				}) Node {
					choose := OnClick(func() { a.setBodyType(t, bt.id) })
					return Choice(bt.label, bt.id == kind, W(bt.width), choose, When(bt.id == kind, KeepInView()))
				})),
			If(isJSON && status != "", Text(status, Small(), Color(statusColor), Ellipsis(), Optional())),
			If(isJSON, Button("Format", W(70), OnClick(func() { a.formatBody(t) }))),
		),
		a.bodyEditor(t, scroll),
	)
}

// bodyEditor is the part of the body pane under the type chooser.
func (a *App) bodyEditor(t *reqTab, scroll *ScrollState) Node {
	switch t.Req.BodyType {
	case model.BodyJSON, model.BodyText:
		base := plainStyler
		if t.Req.BodyType == model.BodyJSON {
			base = jsonStyler
		}
		t.body.Styler = a.withVariables(base, t.Req) // {{variables}} stand out, as in the URL
		sync := func() { editRequest(t, editor.BodyTyped{Text: t.body.Text}) }
		return Area(&t.body, Grow(1), Bg(w.Sunken), OnChange(sync), TextMenu(&t.body.TextField))
	case model.BodyForm, model.BodyMultipart:
		return a.tableView(formPane, t, scroll, 0)
	case model.BodyFile:
		return a.fileBody(t)
	case model.BodyGraphQL:
		return a.graphQLBody(t)
	}
	return Lines("This request has no body", w.Muted)
}

func (a *App) setBodyType(t *reqTab, id string) {
	a.commitEdit()
	editRequest(t, editor.BodyTypeChosen{Type: id, Text: t.body.Text, FilePath: t.file.Text})
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
	editRequest(t, editor.BodyTyped{Text: t.body.Text})
}

// jsonStyler colors one line of a JSON body.
func jsonStyler(line string) []Span {
	src := jsonview.Tokenize(line)
	spans := make([]Span, len(src))
	for i, s := range src {
		spans[i] = Span{Text: s.Text, Style: respview.Tokens[s.Kind]}
	}
	return spans
}

// settingsPane holds the per-request options: timeout and redirects.
func (a *App) settingsPane(_ pane, t *reqTab, scroll *ScrollState) Node {
	return Scroll(scroll, Pad(w.Gutter), Gap(8),
		FormRow("Timeout (seconds)", 160,
			Field(&t.timeout, W(80), Bg(w.Sunken),
				OnChange(func() { a.timeoutEdited(t) }),
				TextMenu(&t.timeout)),
			Text(fmt.Sprintf("blank = default (%d s)", a.timeoutFor(model.Request{})/time.Second), Small(), Muted()),
		),
		CheckRow(!t.Req.NoRedirects, OnClick(func() { editRequest(t, editor.NoRedirectsToggled{}) }),
			Text("Follow redirects"),
		),
		CheckRow(!t.Req.InsecureTLS, OnClick(func() { editRequest(t, editor.InsecureTLSToggled{}) }),
			Text("Verify the server certificate"),
			Text("off accepts self-signed certificates", Small(), Muted()),
		),
	)
}

// timeoutEdited applies the typed timeout; text that is not a whole
// number of seconds is undone.
func (a *App) timeoutEdited(t *reqTab) {
	text := strings.TrimSpace(t.timeout.Text)
	if text == "" {
		editRequest(t, editor.TimeoutSet{Seconds: 0})
		return
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 0 {
		t.timeout = NewTextField(timeoutText(t.Req.TimeoutSec))
		t.timeout.Focused = true
		return
	}
	editRequest(t, editor.TimeoutSet{Seconds: n})
}

// responseEmpty is noResponse plus the case of a body that is not text.
func responseEmpty(t *reqTab) (string, bool) {
	if msg, isErr := noResponse(t); msg != "" {
		return msg, isErr
	}
	if t.picture != nil {
		return "", false
	}
	if respview.LooksBinary(t.Resp.Body) {
		return fmt.Sprintf("Binary body, %s (not shown)", textfmt.Size(len(t.Resp.Body))), false
	}
	return "", false
}

//go:embed responsepane.vml responsepicture.vml
var responsePaneSources embed.FS

// The text view and the picture are two roots, so two templates; Go
// picks one in responsePane.
var (
	responsePaneTemplate    = markup.MustLoad(responsePaneSources, "responsepane.vml")
	responsePictureTemplate = markup.MustLoad(responsePaneSources, "responsepicture.vml")
)

// responsePane is the response body with its toolbar and find bar, or
// the picture of an image body with its size and a Save button.
func (a *App) responsePane(_ pane, t *reqTab, _ *ScrollState) Node {
	if t.picture != nil {
		return responsePictureTemplate.MustBuild(markup.Scope{
			"gutter":      w.Gutter,
			"toolbarH":    w.ToolbarH,
			"compactH":    w.CompactH,
			"pictureInfo": t.pictureInfo,
			"picture":     t.picture,
			"save":        func() { a.saveBody(t) },
		})
	}
	var findBar Node // stays nil, so the slot is left out, while the bar is closed
	if t.look.FindOpen {
		findBar = a.findBar(t)
	}
	return responsePaneTemplate.MustBuild(markup.Scope{
		"gutter":      w.Gutter,
		"toolbarH":    w.ToolbarH,
		"compactH":    w.CompactH,
		"iconBtnW":    w.IconBtnW,
		"compact":     t.look.Compact(),
		"pretty":      t.look.Mode == viewer.Pretty,
		"raw":         t.look.Raw(),
		"wrap":        t.view.Wrap,
		"history":     len(t.Earlier) > 0,
		"findTip":     a.hint("Find in response", "find"),
		"findBar":     findBar,
		"view":        &t.view,
		"showCompact": func() { a.setView(t, viewer.Compact) },
		"showPretty":  func() { a.setView(t, viewer.Pretty) },
		"showRaw":     func() { a.setView(t, viewer.Raw) },
		"toggleWrap":  func() { a.toggleWrap(t) },
		"openHistory": func(x, y int) { a.openMenu(x, y, a.historyMenu(t)) },
		"openFind":    a.openFind,
		"copyBody":    func() { a.copyBody(t) },
		"copyCurl":    func() { a.copyCurl(t) },
		"openMenu":    func(x, y int) { a.openMenu(x, y, a.responseMenuItems(t)) },
	})
}

func (a *App) viewChip(label string, selected bool, width int, onClick func()) Node {
	return Choice(label, selected, W(width), H(w.CompactH), OnClick(onClick))
}

//go:embed findbar.vml
var findBarSource embed.FS

var findBarTemplate = markup.MustLoad(findBarSource, "findbar.vml")

// findBar is the search row above the response: the query, the hit
// counter and the previous, next and close buttons.
func (a *App) findBar(t *reqTab) Node {
	return findBarTemplate.MustBuild(markup.Scope{
		"find":     &t.find,
		"count":    viewer.Count(t.find.Text, len(t.view.Matches()), t.look.Match),
		"rowH":     w.RowH,
		"gutter":   w.Gutter,
		"iconBtnW": w.IconBtnW,
		"compactH": w.CompactH,
		"search":   func() { a.runFind(t) },
		"enter":    func() { a.findEnter(t) },
		"previous": func() { a.findStep(t, -1) },
		"next":     func() { a.findStep(t, 1) },
		"close":    func() { a.closeFind(t) },
	})
}

// setView picks the response view: pretty-printed, as received, or compact.
func (a *App) setView(t *reqTab, mode viewer.Mode) {
	t.look = viewer.Update(t.look, viewer.ModeChosen{Mode: mode})
	a.refreshView(t)
}

// refreshView rebuilds the response view for the current mode and
// redoes an open search. Bodies that are not text are not loaded.
func (a *App) refreshView(t *reqTab) {
	t.picture, t.pictureInfo = nil, ""
	if t.Resp != nil {
		if p, ok := respview.Decode(t.Resp.Body); ok {
			t.picture = ebiten.NewImageFromImage(p.Image)
			t.pictureInfo = fmt.Sprintf("%s %dx%d, %s", strings.ToUpper(p.Format), p.Width, p.Height, textfmt.Size(len(t.Resp.Body)))
			t.view.SetLines(nil)
			return
		}
	}
	switch {
	case t.Resp == nil || respview.LooksBinary(t.Resp.Body):
		t.view.SetLines(nil)
	case t.look.Raw():
		t.view.SetLines(respview.RawLines(t.Resp.Body))
	case t.look.Compact():
		t.compactAll = respview.CompactLines(t.Resp.Body)
		a.showCompact(t)
		return // showCompact searches too
	default:
		t.view.SetLines(respview.CodeLines(t.Resp.Body))
	}
	if t.look.FindOpen {
		a.runFind(t)
	}
}

func (a *App) copyBody(t *reqTab) {
	if t.Resp != nil {
		_ = a.ctx.Clipboard.Write(string(t.Resp.Body)) // the clipboard reports its own failures
	}
}

func (a *App) copyCurl(t *reqTab) {
	_ = a.ctx.Clipboard.Write(a.curlOf(t.Req.Expanded(a.varsFor(t.Req))))
}

// openFind shows the find bar for the current response and focuses it.
func (a *App) openFind() {
	a.commitEdit()
	t := a.cur()
	t.respTab = 0
	t.look = viewer.Update(t.look, viewer.FindOpened{})
	t.find.Focused = true
	t.find.SelectAll()
	a.runFind(t)
}

// showCompact fills the view with the compact lines: all of them with
// the deep blocks folded, or, while a search has text, only the matching
// lines with their children and the blocks around them (filterCompact).
func (a *App) showCompact(t *reqTab) {
	if t.look.FindOpen && t.find.Text != "" {
		t.view.SetLines(respview.Filter(t.compactAll, t.find.Text))
	} else {
		t.view.SetLines(t.compactAll)
		respview.FoldDeep(&t.view, respview.CompactFoldDepth)
	}
	if t.look.FindOpen {
		a.searchView(t)
	}
}

// runFind searches for the typed text and goes to the first match. The
// compact view is filtered to the matches first.
func (a *App) runFind(t *reqTab) {
	if t.look.Compact() && t.Resp != nil && !respview.LooksBinary(t.Resp.Body) {
		a.showCompact(t)
		return
	}
	a.searchView(t)
}

// searchView highlights the matches in the view and reveals the first.
func (a *App) searchView(t *reqTab) {
	matches := t.view.Find(t.find.Text)
	t.look = viewer.Update(t.look, viewer.Searched{Matches: len(matches)})
	if len(matches) > 0 {
		t.view.RevealLine(matches[0].Line)
	}
	t.view.SetMatches(matches, t.look.Match)
}

// findStep moves to the next (dir 1) or previous (dir -1) match, wrapping.
func (a *App) findStep(t *reqTab, dir int) {
	matches := t.view.Matches()
	if len(matches) == 0 {
		return
	}
	t.look = viewer.Update(t.look, viewer.Stepped{Dir: dir, Matches: len(matches)})
	next := t.look.Match
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
	t.look = viewer.Update(t.look, viewer.FindClosed{})
	t.find.Focused = false
	t.view.SetMatches(nil, -1)
	if t.look.Compact() && t.Resp != nil && !respview.LooksBinary(t.Resp.Body) {
		a.showCompact(t) // leave the filtered view
	}
}

// curlOf is the cURL command of req, "" when the app has no formatter.
func (a *App) curlOf(req model.Request) string {
	if a.curl == nil {
		return ""
	}
	return a.curl(req)
}
