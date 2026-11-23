package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// paneKind is what builds a pane of the response area.
type paneKind int

const (
	paneResponse paneKind = iota // a pane of the response area, drawn by ui/response
	paneStream                   // a pane of a socket's response side, drawn by the editor
	paneKinds                    // how many kinds there are
)

// pane describes one tab of the response area. The request side's table is
// ui/editor's (panes.go); the two have a few lines of the same shape, which
// two small tables do not justify sharing.
type pane struct {
	label    string
	kind     paneKind
	respPane response.Pane                                            // paneResponse: which of the response component's panes
	stream   int                                                      // paneStream: its place among the editor's stream panes
	empty    func(app *Component, t *reqTab) (msg string, isErr bool) // shown instead of the content
}

func noResponse(app *Component, tab *reqTab) (string, bool) {
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

// responseEmpty is noResponse plus the case of a body that is not text.
func responseEmpty(app *Component, tab *reqTab) (string, bool) {
	if msg, isErr := noResponse(app, tab); msg != "" {
		return msg, isErr
	}
	return app.resp.Empty(tab.Tab)
}

// responsePanes are the response side of REST and of every protocol with no
// panes of its own: the tabs of the response area, drawn by ui/response.
var responsePanes = []pane{
	{label: "Response", kind: paneResponse, respPane: response.Body, empty: responseEmpty},
	{label: "Headers", kind: paneResponse, respPane: response.Headers, empty: noResponse},
	{label: "Timeline", kind: paneResponse, respPane: response.Timeline, empty: noResponse},
	{label: "Tests", kind: paneResponse, respPane: response.Tests},
}

// badge is the count shown on a pane's tab.
func (p pane) badge(app *Component, tab *reqTab) int {
	if p.kind != paneResponse {
		return 0
	}
	return responseBadge(app, p.respPane, tab)
}

// responseBadge is the count shown on a tab of the response area: how many
// headers came back, how many assertions failed.
func responseBadge(app *Component, shown response.Pane, tab *reqTab) int {
	switch shown {
	case response.Headers:
		if tab.Resp == nil {
			return 0
		}
		return len(tab.Resp.Headers)
	case response.Tests:
		failed := 0
		for _, result := range app.editor.Assertions.Results(tab.Tab) {
			if !result.OK {
				failed++
			}
		}
		return failed
	}
	return 0
}

func (a *Component) tabItems(panes []pane, tab *reqTab) []TabItem {
	items := make([]TabItem, len(panes))
	for i, p := range panes {
		items[i] = TabItem{Label: p.label, Badge: p.badge(a, tab)}
	}
	return items
}

func paneEmpty(app *Component, panel pane, tab *reqTab) (string, bool) {
	if panel.empty == nil {
		return "", false
	}
	return panel.empty(app, tab)
}

// paneBuilders turn a pane into its view, by kind.
var paneBuilders = [paneKinds]func(app *Component, panel pane, t *reqTab, scroll *ScrollState) Node{
	paneResponse: (*Component).responsePaneView,
	paneStream:   (*Component).streamPaneView,
}

// responsePanesFor are the panes of the response side of tab: the editor's
// stream panes while it is a socket.
func (a *Component) responsePanesFor(tab *reqTab) []pane {
	if !a.editor.Socket(tab.Tab) {
		return responsePanes
	}
	labels := a.editor.StreamPanes(tab.Tab)
	panes := make([]pane, len(labels))
	for i, label := range labels {
		panes[i] = pane{label: label, kind: paneStream, stream: i}
	}
	return panes
}

// responsePaneView is a pane of the response area, drawn by the response
// component. It scrolls with the tab's response scroll, whichever scroll it
// is handed: the Tests pane has always done so, and the panes of a socket tab
// share it.
func (a *Component) responsePaneView(panel pane, tab *reqTab, scroll *ScrollState) Node {
	return a.resp.View(a.ctx, tab.Tab, panel.respPane, &tab.respScroll)
}

// streamPaneView is a pane of a socket's response side, drawn by the
// editor's components.
func (a *Component) streamPaneView(panel pane, tab *reqTab, _ *ScrollState) Node {
	return a.editor.StreamView(a.ctx, tab.Tab, panel.stream)
}

// paneBody is what fills the area under a response pane's tab strip.
func (a *Component) paneBody(panel pane, tab *reqTab, scroll *ScrollState) Node {
	if msg, isErr := paneEmpty(a, panel, tab); msg != "" {
		return a.messagePane(panel, msg, isErr)
	}
	return paneBuilders[panel.kind](a, panel, tab, scroll)
}

var messagePaneTemplate = markup.Must("messagepane.vml", messagepaneSource)

// messagePane shows a status line ("Sending...", an error) in place of
// the pane's content.
func (a *Component) messagePane(panel pane, msg string, isErr bool) Node {
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

// Templates, parsed once above.
const (
	messagepaneSource = `<col grow="1" pad="{{gutter}}" bg="bg">
  <lines color="{{color}}">{{message}}</lines>
</col>
`
)
