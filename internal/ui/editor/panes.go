package editor

import (
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

type paneKind int

const (
	paneTable paneKind = iota
	paneBody
	paneSettings
	paneAuth
	paneMessages   // the response side of a socket: the log
	paneAssertions // the assertions being written
	paneExplorer   // the response side of MCP: what the server offers
	paneProto      // the request side of gRPC: reflection or .proto files
	paneSchema     // the request side of gRPC: the fields of the method's message
	paneKinds      // how many kinds there are
)

// pane describes one tab of the request area, or one of the response side
// of a socket. The tables below are the whole definition; there is no
// per-tab switch.
type pane struct {
	label          string
	kind           paneKind
	title          string                                          // caption above a table
	rows           func(c *Component, tab *tabs.Tab) *[]request.KV // paneTable
	changed        func(c *Component, tab *tabs.Tab)               // runs after an edit
	flagDuplicates bool                                            // mark rows that repeat a name
	caption        func(c *Component) string                       // the caption when it depends on the host
	table          kvlogic.Kind                                    // paneTable: which table of ui/editor/kvtable it shows
}

// captionOf is the caption above the pane's table.
func (p pane) captionOf(c *Component) string {
	if p.caption != nil {
		return p.caption(c)
	}
	return p.title
}

func paramsChanged(c *Component, tab *tabs.Tab) {
	*tab.Req = requestedit.Update(*tab.Req, requestedit.ParamsChanged{})
	c.URLBar.Reload(tab)
}

var (
	paramsPane = pane{label: "Params", kind: paneTable, title: "Query", changed: paramsChanged, table: kvlogic.Params,
		rows: func(_ *Component, tab *tabs.Tab) *[]request.KV { return &tab.Req.Params }}
	bodyPane = pane{label: "Body", kind: paneBody}
	// formPane is the table of form and multipart fields; the body pane
	// asks for it through its Slots.
	formPane = pane{label: "Body", kind: paneTable, title: "Fields", table: kvlogic.Form,
		rows: func(_ *Component, tab *tabs.Tab) *[]request.KV { return &tab.Req.Form }}
	// schemaPane is the Schema tab of a gRPC request: the fields of the request
	// message of its method, drawn by the body component.
	schemaPane  = pane{label: "Schema", kind: paneSchema}
	headersPane = pane{label: "Headers", kind: paneTable, title: "Headers", flagDuplicates: true, table: kvlogic.Headers,
		rows: func(_ *Component, tab *tabs.Tab) *[]request.KV { return &tab.Req.Headers }}
	authPane = pane{label: "Auth", kind: paneAuth}
	varsPane = pane{label: "Vars", kind: paneTable, title: "Collection variables", table: kvlogic.Vars,
		caption: func(c *Component) string { return c.deps.VarsPane.Caption() },
		rows:    func(c *Component, _ *tabs.Tab) *[]request.KV { return c.deps.VarsPane.Rows() }}
	testsPane    = pane{label: "Tests", kind: paneAssertions}
	settingsPane = pane{label: "Settings", kind: paneSettings}
	// protoPane is the Proto tab of a gRPC request: where its methods come
	// from, the server's reflection or .proto files, drawn by the gRPC component.
	protoPane = pane{label: "Proto", kind: paneProto}
)

// grpcRequestPanes are the request side of gRPC: the HTTP-like panes with
// Proto after the Body, which holds the message the files describe.
var grpcRequestPanes = []pane{paramsPane, bodyPane, protoPane, schemaPane, headersPane, authPane, varsPane, testsPane, settingsPane}

// requestPanes are the request side of REST and of every protocol with no
// panes of its own.
var requestPanes = []pane{paramsPane, bodyPane, headersPane, authPane, varsPane, testsPane, settingsPane}

// connectionRequestPanes are the request side of a protocol that opens a
// connection: what it sends is composed in the Messages pane, so there is
// no Params, Body or Tests tab.
var connectionRequestPanes = []pane{headersPane, authPane, varsPane, settingsPane}

// wsPanes are the panes of a socket's response side.
var wsPanes = []pane{{label: "Messages", kind: paneMessages}}

// mcpPanes are the response side of an MCP server: what it offers, and
// the JSON-RPC traffic with the composer for anything else.
var mcpPanes = []pane{{label: "Explorer", kind: paneExplorer}, {label: "JSON-RPC", kind: paneMessages}}

// protocolView is how the app shows one protocol: the UI half of its row,
// next to the request.Protocols row that says what the protocol does.
type protocolView struct {
	protocol     request.Protocol
	requestPanes []pane
	streamPanes  []pane                       // the response side while connected
	composer     ws.Composer                  // how the Messages pane composes: whether it has a field, offers { } and hints
	offered      func(settings.Settings) bool // whether the chip cycles to it; nil is always
}

// protocolViews has one row per row of request.Protocols, in the same order.
var protocolViews = []protocolView{
	{protocol: request.ProtocolREST, requestPanes: requestPanes},
	{protocol: request.ProtocolWebSocket, requestPanes: connectionRequestPanes, streamPanes: wsPanes,
		composer: ws.Composer{Shown: true, FormatsJSON: true}},
	{protocol: request.ProtocolMCP, requestPanes: connectionRequestPanes, streamPanes: mcpPanes,
		composer: ws.Composer{Shown: true, Hint: "call TOOL {json}  ·  read URI  ·  prompt NAME {json}  ·  method {json}  ·  or raw JSON-RPC"}},
	{protocol: request.ProtocolSSE, requestPanes: connectionRequestPanes, streamPanes: wsPanes,
		offered: func(current settings.Settings) bool { return current.SSE }},
	{protocol: request.ProtocolGRPC, requestPanes: grpcRequestPanes, streamPanes: wsPanes, composer: ws.Composer{Shown: true},
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

// tablePanes is the pane that shows each table, by kind: where its rows are
// and what follows a change of them.
var tablePanes = [kvlogic.KindCount]pane{
	kvlogic.Params:  paramsPane,
	kvlogic.Headers: headersPane,
	kvlogic.Vars:    varsPane,
	kvlogic.Form:    formPane,
}

// builders turn a pane into its view, by kind. The scroll is nil for a
// stream pane, which scrolls itself.
var builders = [paneKinds]func(c *Component, ctx *w.Ctx, panel pane, tab *tabs.Tab, scroll *widget.ScrollState) Node{
	paneBody:       (*Component).bodyView,
	paneSettings:   (*Component).settingsView,
	paneAuth:       (*Component).authView,
	paneMessages:   (*Component).messagesView,
	paneExplorer:   (*Component).explorerView,
	paneAssertions: (*Component).assertionsView,
	paneTable:      (*Component).tablePaneView,
	paneProto:      (*Component).protoView,
	paneSchema:     (*Component).schemaView,
}

func (c *Component) bodyView(ctx *w.Ctx, _ pane, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.Body.View(ctx, tab, scroll)
}

func (c *Component) settingsView(ctx *w.Ctx, _ pane, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.Assertions.SettingsView(ctx, tab, scroll)
}

func (c *Component) authView(ctx *w.Ctx, _ pane, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.Auth.View(ctx, tab, scroll)
}

func (c *Component) messagesView(ctx *w.Ctx, _ pane, tab *tabs.Tab, _ *widget.ScrollState) Node {
	return c.WS.View(ctx, tab)
}

func (c *Component) explorerView(ctx *w.Ctx, _ pane, tab *tabs.Tab, _ *widget.ScrollState) Node {
	return c.MCP.View(ctx, tab)
}

func (c *Component) assertionsView(ctx *w.Ctx, _ pane, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.Assertions.View(ctx, tab, scroll)
}

func (c *Component) tablePaneView(ctx *w.Ctx, panel pane, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.tableView(ctx, panel, tab, scroll, w.Gutter)
}

func (c *Component) protoView(ctx *w.Ctx, _ pane, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.GRPC.View(ctx, tab, scroll)
}

func (c *Component) schemaView(ctx *w.Ctx, _ pane, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.Body.SchemaView(ctx, tab, scroll)
}

// paneState is what a tab keeps of the request pane: the pane shown, the
// scroll of the pane and the scroll of its strip of tabs.
type paneState struct {
	pane   int
	scroll widget.ScrollState // of the table and note panes
	tabsX  widget.ScrollState // sideways, of the strip
}

// state is the pane state of tab, made when first asked for.
func (c *Component) state(tab *tabs.Tab) *paneState {
	state := c.panes[tab]
	if state == nil {
		state = &paneState{}
		c.panes[tab] = state
	}
	return state
}

// prunePanes drops the state of the tabs that closed.
func (c *Component) prunePanes() {
	for tab := range c.panes {
		if !c.deps.Tabs.IsOpen(tab) {
			delete(c.panes, tab)
		}
	}
}

// requestPanesFor are the panes of the request side of tab.
func requestPanesFor(tab *tabs.Tab) []pane { return viewOf(tab.Req.Protocol).requestPanes }

// PaneOf is the index of the request pane tab shows. It points into the
// panes of the tab's protocol, past their end until the next view clamps it.
func (c *Component) PaneOf(tab *tabs.Tab) int { return c.state(tab).pane }

// ShowPane shows the request pane at index.
func (c *Component) ShowPane(tab *tabs.Tab, index int) { c.state(tab).pane = index }

// PaneLabel is the label of the request pane tab shows, "" when its index
// points nowhere.
func (c *Component) PaneLabel(tab *tabs.Tab) string {
	panes, index := requestPanesFor(tab), c.state(tab).pane
	if index < 0 || index >= len(panes) {
		return ""
	}
	return panes[index].label
}

// ShowPaneLabeled shows the request pane called label and reports whether
// the tab has one; the pane stays as it was when it has not.
func (c *Component) ShowPaneLabeled(tab *tabs.Tab, label string) bool {
	index := slices.IndexFunc(requestPanesFor(tab), func(panel pane) bool { return panel.label == label })
	if index < 0 {
		return false
	}
	c.state(tab).pane = index
	return true
}

// showBody shows the Body pane of tab.
func (c *Component) showBody(tab *tabs.Tab) {
	if index := slices.IndexFunc(requestPanesFor(tab), func(panel pane) bool { return panel.kind == paneBody }); index >= 0 {
		c.state(tab).pane = index
	}
}

// PaneTabs are the tabs of the request pane's strip: each pane's label and
// its badge.
func (c *Component) PaneTabs(tab *tabs.Tab) []widget.TabItem {
	panes := requestPanesFor(tab)
	items := make([]widget.TabItem, len(panes))
	for i, panel := range panes {
		items[i] = widget.TabItem{Label: panel.label, Badge: panel.badge(c, tab)}
	}
	return items
}

// badge is the count shown on a pane's tab: the assertions written, the
// rows of a table that are switched on.
func (p pane) badge(c *Component, tab *tabs.Tab) int {
	switch p.kind {
	case paneAssertions:
		return assert.Count(tab.Req.Tests)
	case paneTable:
		enabled := 0
		for _, row := range *p.rows(c, tab) {
			if row.Enabled {
				enabled++
			}
		}
		return enabled
	}
	return 0
}

var paneTemplate = markup.Must("pane.vml", paneSource)

// RequestPane is the request pane of tab: a strip of tabs over the pane
// shown. The index it shows is kept within the panes of the tab's protocol.
func (c *Component) RequestPane(ctx *w.Ctx, tab *tabs.Tab) Node {
	state, panes := c.state(tab), requestPanesFor(tab)
	selectTab := func(index int) { state.pane, state.scroll.Y = index, 0 }
	state.pane = min(state.pane, len(panes)-1)
	return paneTemplate.MustBuild(markup.Scope{
		"tabsX":     &state.tabsX,
		"tabs":      c.PaneTabs(tab),
		"active":    state.pane,
		"selectTab": selectTab,
		"extra":     nil,
		"body":      c.PaneBody(ctx, tab, &state.scroll),
	})
}

// PaneBody is what fills the area under the strip: the pane shown, drawn by
// the component that owns it.
func (c *Component) PaneBody(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	panel := requestPanesFor(tab)[c.state(tab).pane]
	return builders[panel.kind](c, ctx, panel, tab, scroll)
}

// Socket reports whether the tab's protocol keeps a connection open: always,
// or on a stream the gRPC component says the tab has turned on.
func (c *Component) Socket(tab *tabs.Tab) bool {
	connection := tab.Req.Protocol.Info().Connection
	return connection == request.Always || (connection == request.OnStream && c.GRPC.Streaming(tab))
}

// StreamPanes are the labels of the tabs the response side shows while
// tab is a socket: its Messages, and the Explorer of an MCP server.
func (c *Component) StreamPanes(tab *tabs.Tab) []string {
	panes := viewOf(tab.Req.Protocol).streamPanes
	labels := make([]string, len(panes))
	for i, panel := range panes {
		labels[i] = panel.label
	}
	return labels
}

// StreamView is the stream pane at index of tab, drawn by the component
// that owns it. It scrolls itself.
func (c *Component) StreamView(ctx *w.Ctx, tab *tabs.Tab, index int) Node {
	panel := viewOf(tab.Req.Protocol).streamPanes[index]
	return builders[panel.kind](c, ctx, panel, tab, nil)
}

// Explores reports whether the response side of tab has an explorer, which
// is empty until the server has said who it is.
func (c *Component) Explores(tab *tabs.Tab) bool {
	return slices.ContainsFunc(viewOf(tab.Req.Protocol).streamPanes, func(panel pane) bool { return panel.kind == paneExplorer })
}

// Loading reports whether tab shows a pane that waits for what it reads:
// the Proto pane while the files are read, the Schema pane while the
// message is asked for, so a capture waits for them.
func (c *Component) Loading(tab *tabs.Tab) bool {
	panes, index := requestPanesFor(tab), c.state(tab).pane
	if index < 0 || index >= len(panes) {
		return false
	}
	switch panes[index].kind {
	case paneProto:
		return c.GRPC.ProtoPending(tab)
	case paneSchema:
		return c.Body.SchemaPending(tab)
	}
	return false
}

// Offered are the protocols the chip cycles through for a request on
// current: the experimental ones only when the settings switch them on,
// except the one the request already uses.
func (c *Component) Offered(current request.Protocol) []request.Protocol {
	var offered []request.Protocol
	for _, view := range protocolViews {
		if view.offered == nil || view.offered(c.deps.Settings.Settings()) || view.protocol == current {
			offered = append(offered, view.protocol)
		}
	}
	return offered
}

// Choose gives tab the protocol. A connection belongs to the protocol that
// opened it, so it is closed first; the pane indexes point into the panes of
// the protocol, so they start over: this one here, the response side
// through the host.
func (c *Component) Choose(tab *tabs.Tab, protocol request.Protocol) {
	c.deps.Connections.Disconnect(tab)
	*tab.Req = requestedit.Update(*tab.Req, requestedit.ProtocolChosen{Protocol: protocol})
	c.state(tab).pane = 0
	c.deps.Connections.ProtocolChosen(tab)
}

// TableRows are the rows of table, edited in place.
func (c *Component) TableRows(table kvlogic.Table) *[]request.KV {
	return tablePanes[table.Kind].rows(c, table.Tab)
}

// TableCaption is the caption above the table of kind.
func (c *Component) TableCaption(kind kvlogic.Kind) string { return tablePanes[kind].captionOf(c) }

// VisibleTable is the editable table the request pane shows, if any:
// Params, Headers or Vars as the pane shown, the form fields under a Form
// or Multipart body.
func (c *Component) VisibleTable() (kvlogic.Table, bool) {
	tab := c.deps.Tabs.Current()
	shown := requestPanesFor(tab)[c.state(tab).pane]
	switch {
	case shown.kind == paneTable:
		return kvlogic.Table{Tab: tab, Kind: shown.table}, true
	case shown.kind == paneBody && (tab.Req.BodyType == request.BodyForm || tab.Req.BodyType == request.BodyMultipart):
		return kvlogic.Table{Tab: tab, Kind: formPane.table}, true
	}
	return kvlogic.Table{}, false
}

// TableView is the table of kind in the padding pad, drawn by the tables.
func (c *Component) TableView(tab *tabs.Tab, kind kvlogic.Kind, scroll *widget.ScrollState, pad int) Node {
	return c.tableView(c.deps.Ctx, tablePanes[kind], tab, scroll, pad)
}

func (c *Component) tableView(ctx *w.Ctx, panel pane, tab *tabs.Tab, scroll *widget.ScrollState, pad int) Node {
	look := kvtable.Look{Caption: panel.captionOf(c), FlagDuplicates: panel.flagDuplicates}
	return c.Tables.View(ctx, kvlogic.Table{Tab: tab, Kind: panel.table}, look, scroll, pad)
}

// tableRows answers the tables' Tables: the rows of the pane table and what
// follows a change of them.
type tableRows struct{ c *Component }

func (rows tableRows) Rows(table kvlogic.Table) *[]request.KV { return rows.c.TableRows(table) }

// Changed runs the hook of the table's pane: the Params rewrite the URL.
func (rows tableRows) Changed(table kvlogic.Table) {
	if changed := tablePanes[table.Kind].changed; changed != nil {
		changed(rows.c, table.Tab)
	}
}

// tableTabs answers the tables' Tabs.
type tableTabs struct{ c *Component }

func (host tableTabs) IsOpen(tab *tabs.Tab) bool      { return host.c.deps.Tabs.IsOpen(tab) }
func (host tableTabs) Visible() (kvlogic.Table, bool) { return host.c.VisibleTable() }

// slots answers the body's Slots: the tables, the GraphQL editor and a
// status line, all of which are this composite's.
type slots struct{ c *Component }

func (slots slots) FormTable(tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return slots.c.TableView(tab, formPane.table, scroll, 0)
}

func (slots slots) GraphQL(tab *tabs.Tab) Node { return slots.c.GraphQL.View(slots.c.deps.Ctx, tab) }

func (slots slots) Message(text string) Node { return messagePane(text) }

// bodyWindow is the body's Window with the one thing the pane table
// answers: which pane is the Body.
type bodyWindow struct {
	Window
	c *Component
}

func (window bodyWindow) ShowBody(tab *tabs.Tab) { window.c.showBody(tab) }

// composers says how the Messages pane of each protocol composes: a column
// of the protocol table.
type composers struct{}

func (composers) Composer(tab *tabs.Tab) ws.Composer { return viewOf(tab.Req.Protocol).composer }

var messagePaneTemplate = markup.Must("messagepane.vml", messagepaneSource)

// messagePane shows a status line in place of a pane's content. The
// response side keeps a twin of it (ui/window), which also shows errors.
func messagePane(text string) Node {
	return messagePaneTemplate.MustBuild(markup.Scope{
		"gutter":  w.Gutter,
		"color":   w.Muted,
		"message": text,
	})
}

// Templates, parsed once above.
const (
	paneSource = `<col grow="1" bg="bg">
  <tab-bar>
    <scroll-tabs state="tabsX" items="tabs" active="{{active}}" on-select="selectTab"/>
    <slot if="extra" name="extra"/>
  </tab-bar>
  <slot name="body"/>
</col>
`

	messagepaneSource = `<col grow="1" pad="{{gutter}}" bg="bg">
  <lines color="{{color}}">{{message}}</lines>
</col>
`
)
