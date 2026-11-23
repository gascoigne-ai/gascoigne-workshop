package app

import (
	"context"
	request "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/outbox"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"slices"
)

// reqTab is one open request with its editor and response state.
type reqTab struct {
	*tabs.Tab
	socketBlock
	grpcBlock
	mcpBlock
	responseBlock
	url         widget.TextField
	paneTab     int
	bodyView    widget.CodeView                             // request body, read-only
	tests       widget.TextArea                             // the assertions editor
	gqlVars     widget.TextArea                             // the variables of a GraphQL body
	file        widget.TextField                            // the path of a file body
	fileCheck   bodyFileCheck                               // what the file system said about file's path
	body        widget.TextArea                             // request body editor
	reqScroll   widget.ScrollState                          // scroll of the table and note panes
	bodyTypesX  widget.ScrollState                          // sideways scroll of the body type chips
	reqTabsX    widget.ScrollState                          // sideways scroll of the two pane tab strips
	timeout     widget.TextField                            // the request's timeout seconds, as typed
	cancel      context.CancelFunc                          // stops the running request
	auth        map[requestedit.AuthField]*widget.TextField // Auth pane inputs by field id, created on first use
	showSecrets bool                                        // show masked credential fields
	schema      *send.Schema                                // what a GraphQL introspection answered
}

// socketBlock holds the state of a WebSocket tab: the message log, the message being composed and the sends waiting their turn.
type socketBlock struct {
	wsCancel context.CancelFunc // ends the timeout of the connect in progress
	outbox   outbox.Outbox      // the sends and pings in order, and whether one is in flight
	wsView   widget.CodeView    // the message log
	msg      widget.TextField   // the message being composed
}

// grpcBlock holds the state of a gRPC tab: streaming and the Proto and Schema panes.
type grpcBlock struct {
	grpcStream    bool               // a gRPC tab streams instead of making single calls
	proto         protoState         // the Proto pane of a gRPC tab
	messageSchema messageSchemaState // the Schema pane of a gRPC tab
}

// mcpBlock holds the state of an MCP tab: the explorer, the form of the open entry and its answer.
type mcpBlock struct {
	explorer     mcpexplorer.State  // the MCP explorer: the list shown, the open entry, its answer
	form         *mcpform.Form      // the argument form of the open entry, nil for none
	resultView   widget.CodeView    // the last answer of the explorer
	listScroll   widget.ScrollState // scroll of the explorer's list
	detailScroll widget.ScrollState // scroll of the open entry
}

// responseBlock holds the state of the response pane: the body shown, the response mode and find bar, the assertion results and the pane scroll.
type responseBlock struct {
	respTab     int
	view        widget.CodeView    // response body
	picture     *ebiten.Image      // the response body when it is an image
	pictureInfo string             // "PNG 640x480, 12.5KB"
	testResults []assert.Result    // what the assertions found in the shown response
	look        viewer.Model       // the response mode, the find bar and its active match
	compactAll  []widget.Line      // the unfiltered compact lines, kept while a search filters the view
	find        widget.TextField   // the search text
	respScroll  widget.ScrollState // scroll of the table and note panes
	respTabsX   widget.ScrollState // sideways scroll of the two pane tab strips
}

// makeTab is newTab with the response view the settings ask for.
func (a *App) makeTab(tab *tabs.Tab) *reqTab {
	t := newReqTab(tab)
	t.look = viewer.New(a.settings.ResponseView)
	t.Earlier = a.loadHistory(tab.Req)
	return t
}

// updateTabs applies msg to the tab list, then does what the change means
// for the window: closed tabs lose their widgets and new ones get theirs.
func (a *App) updateTabs(msg tabs.Msg) {
	was := a.activeRequest()
	a.tabs.Apply(msg)
	// Widgets are made first: a restore reads cur() right after an Open.
	for _, tab := range a.tabs.Tabs() {
		if a.widgets[tab] == nil {
			a.widgets[tab] = a.makeTab(tab)
		}
	}
	for tab, widgets := range a.widgets {
		if slices.Contains(a.tabs.Tabs(), tab) {
			continue
		}
		a.cancelSend(widgets) // a closed tab keeps neither a request nor a socket
		a.disconnect(widgets)
		delete(a.widgets, tab)
	}
	if now := a.activeRequest(); now != nil && now != was {
		a.revealRequest(now)
	}
}

// activeRequest is the request of the current tab, nil when none is open.
func (a *App) activeRequest() *request.Request {
	current := a.tabs.Current()
	if current == nil {
		return nil
	}
	return current.Req
}

// openTabs are the widget states of the open tabs, in the order of the strip.
func (a *App) openTabs() []*reqTab {
	open := make([]*reqTab, len(a.tabs.Tabs()))
	for i, tab := range a.tabs.Tabs() {
		open[i] = a.widgets[tab]
	}
	return open
}

func newTab(req *request.Request) *reqTab { return newReqTab(tabs.New(req)) }

func newReqTab(tab *tabs.Tab) *reqTab {
	req := tab.Req
	t := &reqTab{Tab: tab, url: widget.NewTextField(req.URL)}
	t.look.Match = -1
	t.wsView.Follow = true
	t.wsCancel = func() {}
	t.explorer = mcpexplorer.New()
	t.timeout = widget.NewTextField(timeoutText(req.TimeoutSec))
	t.body = widget.NewTextArea(req.Body)
	t.tests = widget.NewTextArea(req.Tests)
	t.file = widget.NewTextField(req.Body)
	t.gqlVars = widget.NewTextArea(req.GraphQLVars)
	return t
}

func (a *App) cur() *reqTab { return a.widgets[a.tabs.Current()] }

// openRequest focuses the tab of req, opening one if needed.
func (a *App) openRequest(req *request.Request) {
	a.side.Tree().Select(request.Node{Req: req})
	a.updateTabs(tabs.Open{Req: req})
}

// editRequest applies an edit to the request of tab, which is the collection's
// own, so the sidebar and the autosave see it at once.
func editRequest(tab *reqTab, msg requestedit.Msg) { *tab.Req = requestedit.Update(*tab.Req, msg) }

// dirtyMark is shown on tabs whose request changed since it was opened.
const dirtyMark = "\u2022"

// dirty reports whether the request differs from its state when opened.
func dirty(tab *reqTab) bool { return tab.Dirty() }
