package app

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"slices"
)

// reqTab is one open request with its editor and response state.
type reqTab struct {
	*tabs.Tab
	url                   widget.TextField
	paneTab               int
	respTab               int
	view                  widget.CodeView              // response body
	bodyView              widget.CodeView              // request body, read-only
	picture               *ebiten.Image                // the response body when it is an image
	pictureInfo           string                       // "PNG 640x480, 12.5KB"
	tests                 widget.TextArea              // the assertions editor
	testResults           []assert.Result              // what the assertions found in the shown response
	gqlVars               widget.TextArea              // the variables of a GraphQL body
	file                  widget.TextField             // the path of a file body
	fileCheck             bodyFileCheck                // what the file system said about file's path
	body                  widget.TextArea              // request body editor
	reqScroll, respScroll widget.ScrollState           // scroll of the table and note panes
	bodyTypesX            widget.ScrollState           // sideways scroll of the body type chips
	reqTabsX, respTabsX   widget.ScrollState           // sideways scroll of the two pane tab strips
	timeout               widget.TextField             // the request's timeout seconds, as typed
	cancel                context.CancelFunc           // stops the running request
	auth                  map[string]*widget.TextField // Auth pane inputs by field id, created on first use
	showSecrets           bool                         // show masked credential fields
	look                  viewer.Model                 // the response mode, the find bar and its active match
	compactAll            []widget.Line                // the unfiltered compact lines, kept while a search filters the view
	find                  widget.TextField             // the search text
	wsCancel              context.CancelFunc           // ends the timeout of the connect in progress
	outbox                []queuedSend                 // sends and pings waiting for the one in flight, in order
	outboxBusy            bool                         // a send is in flight
	wsView                widget.CodeView              // the message log
	msg                   widget.TextField             // the message being composed
	grpcStream            bool                         // a gRPC tab streams instead of making single calls
	proto                 protoState                   // the Proto pane of a gRPC tab
	messageSchema         messageSchemaState           // the Schema pane of a gRPC tab
	schema                *model.Schema                // what a GraphQL introspection answered
	explorer              mcpexplorer.State            // the MCP explorer: the list shown, the open entry, its answer
	form                  *mcpform.Form                // the argument form of the open entry, nil for none
	resultView            widget.CodeView              // the last answer of the explorer
	listScroll            widget.ScrollState           // scroll of the explorer's list
	detailScroll          widget.ScrollState           // scroll of the open entry
}

// makeTab is newTab with the response view the settings ask for.
func (a *App) makeTab(tab *tabs.Tab) *reqTab {
	t := newReqTab(tab)
	t.look = viewer.New(a.settings.ResponseView)
	t.Earlier = a.loadHistory(tab.Req)
	return t
}

// updateTabs applies msg to the tab list, then does what the change means
// for the window: an inline edit in a closed tab ends, closed tabs lose
// their widgets and new ones get theirs.
func (a *App) updateTabs(msg tabs.Msg) {
	next := tabs.Update(a.list, msg)
	for tab, widgets := range a.widgets {
		if slices.Contains(next.Tabs, tab) {
			continue
		}
		if a.edit != nil && a.edit.tab == widgets {
			a.cancelEdit()
		}
		a.cancelSend(widgets) // a closed tab keeps neither a request nor a socket
		a.disconnect(widgets)
		delete(a.widgets, tab)
	}
	for _, tab := range next.Tabs {
		if a.widgets[tab] == nil {
			a.widgets[tab] = a.makeTab(tab)
		}
	}
	was := a.activeRequest()
	a.list = next
	if now := a.activeRequest(); now != nil && now != was {
		a.revealRequest(now)
	}
}

// activeRequest is the request of the active tab, nil when none is open.
func (a *App) activeRequest() *model.Request {
	if a.list.Active < 0 || a.list.Active >= len(a.list.Tabs) {
		return nil
	}
	return a.list.Tabs[a.list.Active].Req
}

// openTabs are the widget states of the open tabs, in the order of the strip.
func (a *App) openTabs() []*reqTab {
	open := make([]*reqTab, len(a.list.Tabs))
	for i, tab := range a.list.Tabs {
		open[i] = a.widgets[tab]
	}
	return open
}

func newTab(req *model.Request) *reqTab { return newReqTab(tabs.New(req)) }

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

func (a *App) cur() *reqTab { return a.widgets[a.list.Tabs[a.list.Active]] }

// openRequest focuses the tab of req, opening one if needed.
func (a *App) openRequest(req *model.Request) {
	a.selected = model.Of(req)
	a.updateTabs(tabs.Open{Req: req})
}

// editRequest applies an edit to the request of tab, which is the collection's
// own, so the sidebar and the autosave see it at once.
func editRequest(tab *reqTab, msg editor.Msg) { *tab.Req = editor.Update(*tab.Req, msg) }

// dirtyMark is shown on tabs whose request changed since it was opened.
const dirtyMark = "\u2022"

// dirty reports whether the request differs from its state when opened.
func dirty(tab *reqTab) bool { return tab.Dirty() }
