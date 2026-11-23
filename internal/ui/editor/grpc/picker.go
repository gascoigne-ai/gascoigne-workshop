package grpc

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// PickMethod asks the server for its methods in the background and opens a
// menu of them where the pointer is now: the pointer is read here, before
// the answer, because the user has moved on by then. Choosing one sets the
// request's method. Proto files are read without a server; reflection asks
// one, so a request without a URL is told to enter it first.
func (c *Component) PickMethod(ctx *core.Ctx, tab *tabs.Tab) {
	if tab.Req.Proto.Kind() == request.SourceReflection && strings.TrimSpace(tab.Req.URL) == "" {
		c.deps.Window.Toast("Enter the gRPC server first, for example grpc://localhost:50051", severity.Error)
		return
	}
	req := tab.Req.Expanded(c.deps.Vars.VarsFor(tab.Req))
	posX, posY := ctx.In.X, ctx.In.Y
	c.deps.Window.SetStatus("Listing methods of " + req.URL)
	c.deps.Methods.List(req, func(methods []string, err error) {
		if err != nil {
			c.deps.Window.Toast("Listing methods failed: "+err.Error(), severity.Error)
			return
		}
		items := make([]widget.MenuItem, 0, len(methods))
		for _, method := range methods {
			items = append(items, widget.MenuItem{Label: method, Run: func() { editRequest(tab, requestedit.MethodChosen{Method: method}) }})
		}
		c.deps.Window.Open(posX, posY, items)
	})
}

// Streaming reports whether tab streams instead of making single calls. It
// makes no state: a tab nobody switched does not stream.
func (c *Component) Streaming(tab *tabs.Tab) bool {
	made, ok := c.states[tab]
	return ok && made.streaming
}

// ToggleStream switches tab between single calls and a stream shown in the
// Messages pane. The open stream ends first: it belongs to the mode that
// opened it.
func (c *Component) ToggleStream(tab *tabs.Tab) {
	c.deps.Connections.Disconnect(tab)
	made := c.state(tab)
	made.streaming = !made.streaming
}
