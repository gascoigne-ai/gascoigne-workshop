package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
	paneProto      // the request side of gRPC: reflection or .proto files
	paneSchema     // the request side of gRPC: the fields of the method's message
)

// pane describes one tab of the request or response area. The tables
// below are the whole definition; there is no per-tab switch.
type pane struct {
	label          string
	kind           paneKind
	title          string                                             // caption above a table
	rows           func(app *App, t *reqTab) *[]request.KV            // paneTable
	editable       bool                                               // table rows can be toggled and deleted
	changed        func(app *App, t *reqTab)                          // runs after an edit
	flagDuplicates bool                                               // mark rows that repeat a name
	note           func(t *reqTab) string                             // paneNote
	empty          func(app *App, t *reqTab) (msg string, isErr bool) // shown instead of the content
	caption        func(app *App) string                              // the caption when it depends on the app state
	table          kvlogic.Kind                                       // paneTable: which table of ui/editor/kvtable it shows
}

// captionOf is the caption above the pane's table.
func (p pane) captionOf(app *App) string {
	if p.caption != nil {
		return p.caption(app)
	}
	return p.title
}

func paramsChanged(app *App, tab *reqTab) {
	editRequest(tab, requestedit.ParamsChanged{})
	app.urlbar.Reload(tab.Tab)
}

var (
	paramsPane = pane{label: "Params", kind: paneTable, title: "Query", editable: true, changed: paramsChanged, table: kvlogic.Params,
		rows: func(_ *App, tab *reqTab) *[]request.KV { return &tab.Req.Params }}
	bodyPane = pane{label: "Body", kind: paneBody}
	// formPane is the table of form and multipart fields; the body pane
	// asks for it through its Slots.
	formPane = pane{label: "Body", kind: paneTable, title: "Fields", editable: true, table: kvlogic.Form,
		rows: func(_ *App, tab *reqTab) *[]request.KV { return &tab.Req.Form }}
	// schemaPane is the Schema tab of a gRPC request: the fields of the request
	// message of its method, drawn by the body component.
	schemaPane  = pane{label: "Schema", kind: paneSchema}
	headersPane = pane{label: "Headers", kind: paneTable, title: "Headers", editable: true, flagDuplicates: true, table: kvlogic.Headers,
		rows: func(_ *App, tab *reqTab) *[]request.KV { return &tab.Req.Headers }}
	authPane = pane{label: "Auth", kind: paneAuth}
	varsPane = pane{label: "Vars", kind: paneTable, title: "Collection variables", editable: true, table: kvlogic.Vars,
		caption: (*App).varsCaption,
		rows:    func(app *App, _ *reqTab) *[]request.KV { return app.varsRows() }}
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

// protocolView is how the app shows one protocol: the UI half of its row,
// next to the request.Protocols row that says what the protocol does.
type protocolView struct {
	protocol     request.Protocol
	requestPanes []pane
	streamPanes  []pane                       // the response side while connected
	composer     bool                         // the Messages pane has a field to send from
	formatsJSON  bool                         // the composer offers { } to make the message one line of JSON
	composerHint string                       // what the composer says while connected; "" is "Message"
	offered      func(settings.Settings) bool // whether the chip cycles to it; nil is always
}

// protocolViews has one row per row of request.Protocols, in the same order.
var protocolViews = []protocolView{
	{protocol: request.ProtocolREST, requestPanes: requestPanes},
	{protocol: request.ProtocolWebSocket, requestPanes: connectionRequestPanes, streamPanes: wsPanes,
		composer: true, formatsJSON: true},
	{protocol: request.ProtocolMCP, requestPanes: connectionRequestPanes, streamPanes: mcpPanes, composer: true,
		composerHint: "call TOOL {json}  ·  read URI  ·  prompt NAME {json}  ·  method {json}  ·  or raw JSON-RPC"},
	{protocol: request.ProtocolSSE, requestPanes: connectionRequestPanes, streamPanes: wsPanes,
		offered: func(current settings.Settings) bool { return current.SSE }},
	{protocol: request.ProtocolGRPC, requestPanes: grpcRequestPanes, streamPanes: wsPanes, composer: true,
		offered: func(current settings.Settings) bool { return current.GRPC }},
}

// viewOf is the protocolView of protocol. A request with no known protocol
// (only one built in code) is shown as the first row.
func viewOf(protocol request.Protocol) protocolView {
	for _, view := range protocolViews {
		if view.protocol == protocol {
			return view
		}
	}
	return protocolViews[0]
}

func noResponse(app *App, tab *reqTab) (string, bool) {
	switch {
	case app.sends.Sending(tab.Req):
		return "Sending...", false
	case tab.Resp == nil:
		return "Send the request to see the response", false
	case tab.Resp.Err != "":
		return tab.Resp.Err, true
	}
	return "", false
}

var responsePanes = []pane{
	{label: "Response", kind: paneResponse, empty: responseEmpty},
	{label: "Headers", kind: paneTable, empty: noResponse, table: kvlogic.ResponseHeaders,
		rows: func(_ *App, tab *reqTab) *[]request.KV {
			if tab.Resp == nil {
				return &[]request.KV{}
			}
			return &tab.Resp.Headers
		}},
	{label: "Timeline", kind: paneNote, empty: noResponse,
		note: func(tab *reqTab) string { return respview.Timeline(tab.Resp) }},
	{label: "Tests", kind: paneResults},
}

// badge is the count shown on a pane's tab.
func (p pane) badge(app *App, tab *reqTab) int {
	switch p.kind {
	case paneAssertions: // how many assertions are written
		return assert.Count(tab.Req.Tests)
	case paneResults: // how many failed
		n := 0
		for _, r := range tab.testResults {
			if !r.OK {
				n++
			}
		}
		return n
	}
	if p.kind != paneTable {
		return 0
	}
	rows := *p.rows(app, tab)
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

func (a *App) tabItems(panes []pane, tab *reqTab) []TabItem {
	items := make([]TabItem, len(panes))
	for i, p := range panes {
		items[i] = TabItem{Label: p.label, Badge: p.badge(a, tab)}
	}
	return items
}

func paneEmpty(app *App, panel pane, tab *reqTab) (string, bool) {
	if panel.empty == nil {
		return "", false
	}
	return panel.empty(app, tab)
}

// paneBuilders turns a pane into its view, by kind.
var paneBuilders = map[paneKind]func(app *App, panel pane, t *reqTab, scroll *ScrollState) Node{
	paneNote:       (*App).notePane,
	paneBody:       (*App).bodyPaneView,
	paneSettings:   (*App).settingsPane,
	paneAuth:       (*App).authPaneView,
	paneResponse:   (*App).responsePane,
	paneMessages:   (*App).messagesPane,
	paneExplorer:   (*App).explorerPane,
	paneAssertions: (*App).assertionsPane,
	paneResults:    (*App).resultsPane,
	paneTable:      (*App).tablePane,
	paneProto:      (*App).protoPane,
	paneSchema:     (*App).schemaPaneView,
}

// bodyPaneView is the Body pane, drawn by the body component.
func (a *App) bodyPaneView(panel pane, tab *reqTab, scroll *ScrollState) Node {
	return a.body.View(a.ctx, tab.Tab, scroll)
}

// authPaneView is the Auth pane, drawn by the auth component.
func (a *App) authPaneView(panel pane, tab *reqTab, scroll *ScrollState) Node {
	return a.auth.View(a.ctx, tab.Tab, scroll)
}

// schemaPaneView is the Schema pane of a gRPC tab, drawn by the body component.
func (a *App) schemaPaneView(panel pane, tab *reqTab, scroll *ScrollState) Node {
	return a.body.SchemaView(a.ctx, tab.Tab, scroll)
}

// schemaLoading reports whether tab shows its Schema pane while the message
// is still being asked for, so a capture waits for it.
func (a *App) schemaLoading(tab *reqTab) bool {
	panes := requestPanesFor(tab)
	if tab.paneTab < 0 || tab.paneTab >= len(panes) || panes[tab.paneTab].kind != paneSchema {
		return false
	}
	return a.body.SchemaPending(tab.Tab)
}

// paneBody is what fills the area under a pane's tab strip.
func (a *App) paneBody(panel pane, tab *reqTab, scroll *ScrollState) Node {
	if msg, isErr := paneEmpty(a, panel, tab); msg != "" {
		return a.messagePane(panel, msg, isErr)
	}
	return paneBuilders[panel.kind](a, panel, tab, scroll)
}

var messagePaneTemplate = markup.Must("messagepane.vml", messagepaneSource)

// messagePane shows a status line ("Sending...", an error) in place of
// the pane's content.
func (a *App) messagePane(panel pane, msg string, isErr bool) Node {
	color := w.Muted
	if isErr {
		color = w.Error
	}
	return messagePaneTemplate.MustBuild(markup.Scope{
		"gutter":  w.Gutter,
		"color":   color,
		"message": msg,
	})
}

var notePaneTemplate = markup.Must("notepane.vml", notepaneSource)

// notePane shows the pane's text, such as the response's timeline.
func (a *App) notePane(panel pane, tab *reqTab, scroll *ScrollState) Node {
	return notePaneTemplate.MustBuild(markup.Scope{
		"gutter": w.Gutter,
		"scroll": scroll,
		"note":   panel.note(tab),
	})
}

// Templates, parsed once above.
const (
	messagepaneSource = `<col grow="1" pad="{{gutter}}" bg="bg">
  <lines color="{{color}}">{{message}}</lines>
</col>
`

	notepaneSource = `<scroll state="scroll" pad="{{gutter}}">
  <lines color="text">{{note}}</lines>
</scroll>
`
)

// requestPanesFor are the panes of the request side of tab.
func requestPanesFor(tab *reqTab) []pane {
	return viewOf(tab.Req.Protocol).requestPanes
}
