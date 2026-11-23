// Package editor is the composite of the request pane's components: the URL
// bar, the body, the auth, assertions, WebSocket, GraphQL, gRPC and MCP
// panes and the key-value tables. It builds them, wires the ones that lean
// on each other, and answers the questions the window asks of all of them at
// once: who has the keyboard, which keys are typing, what to tidy each frame
// and which shortcut commands they add. The children stay exported fields:
// the window reaches a pane's own methods through them, so the composite
// forwards only what is the same for every child.
//
// Each child keeps its own narrow Deps, answered by the host (the window);
// the composite takes them as they are because a second copy of the same
// interfaces would only forward calls. Revisit when the pane table moves in
// and the host answers one interface per concern.
package editor

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/assertions"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/auth"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/body"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/graphql"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/grpc"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/urlbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Deps is what each child asks of the host. GraphQL.Query is filled by New:
// the query area is the body's.
type Deps struct {
	URLBar     urlbar.Deps
	Body       body.Deps
	GraphQL    graphql.Deps
	Auth       auth.Deps
	Assertions assertions.Deps
	GRPC       grpc.Deps
	MCP        mcp.Deps
	WS         ws.Deps
	Tables     kvtable.Deps
}

// Component is the request pane's components.
type Component struct {
	URLBar     *urlbar.Component     // the bar above the request pane: URL fields, chips and the Send button
	Body       *body.Component       // the Body and Schema panes: text areas, file paths and the gRPC message schema of each tab
	Auth       *auth.Component       // the Auth pane, the trust prompt and the cookie jar view
	Assertions *assertions.Component // the Tests and Settings panes, the response Tests pane and the assertion results of each tab
	WS         *ws.Component         // the Messages pane: the log, its toolbar and the composer of each socket tab
	GraphQL    *graphql.Component    // the variables and schema buttons of a GraphQL body, per tab
	GRPC       *grpc.Component       // the gRPC method picker, the Stream switch and the Proto pane of each tab
	MCP        *mcp.Component        // the MCP explorer, the tool call form and the sampling dialog of each tab
	Tables     *kvtable.Component    // the key-value tables of the request pane
}

// New builds the components, the GraphQL one over the body's query area.
func New(deps Deps) *Component {
	c := &Component{}
	c.Tables = kvtable.New(deps.Tables)
	c.URLBar = urlbar.New(deps.URLBar)
	c.Body = body.New(deps.Body)
	deps.GraphQL.Query = c.Body
	c.GraphQL = graphql.New(deps.GraphQL)
	c.Auth = auth.New(deps.Auth)
	c.Assertions = assertions.New(deps.Assertions)
	c.GRPC = grpc.New(deps.GRPC)
	c.MCP = mcp.New(deps.MCP)
	c.WS = ws.New(deps.WS)
	return c
}

// Keys reports whether the frame's keys are typing in the URL field or the
// body, which no shortcut may take. The URL bar claims first.
func (c *Component) Keys(in core.Input) bool { return c.URLBar.Keys(in) || c.Body.Keys(in) }

// TableKeys gives the tables the keys nothing above took; typing is whether
// any text widget has the keyboard.
func (c *Component) TableKeys(in core.Input, typing bool) { c.Tables.Keys(in, typing) }

// Typing reports whether a text field of the request pane has the keyboard.
// The gRPC path fields and the MCP explorer are left out: they never held
// back the window's shortcuts, and holding them back now would change which
// keys fire.
func (c *Component) Typing() bool {
	return c.URLBar.Typing() || c.Body.Typing() || c.GraphQL.Typing() || c.WS.Typing() ||
		c.Tables.Typing() || c.Auth.Typing() || c.Assertions.Typing()
}

// Tick tidies every child after the view was built and reports whether any
// of them changed what the view shows. All nine run on every call: one that
// reports a change must not keep a later one from dropping its closed tabs.
//
// The tables tick first so that the test of the rule has a child that
// reports a change before the others run; the children share no state, so
// the order changes nothing else.
func (c *Component) Tick() bool {
	changed := false
	for _, tick := range []func() bool{
		c.Tables.Tick, c.URLBar.Tick, c.Body.Tick, c.Auth.Tick, c.Assertions.Tick,
		c.WS.Tick, c.GraphQL.Tick, c.GRPC.Tick, c.MCP.Tick,
	} {
		changed = tick() || changed
	}
	return changed
}

// Commands are the shortcut commands the request pane adds: the tables'.
func (c *Component) Commands() []keymap.Command { return c.Tables.Commands() }
