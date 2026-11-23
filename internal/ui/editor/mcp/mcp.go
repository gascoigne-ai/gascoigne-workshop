// Package mcp is the MCP side of the request editor: the explorer of what a
// server offers (its tools, resources and prompts, with a form for the open
// entry and its answer), the call form of a tool that fills the composer, and
// the dialog that answers a server's sampling request. The tabs, the session
// and the dialog overlay stay with the application, which this package
// reaches through Deps; the form itself is the child package mcpform. The
// package is named like the protocol.
//
// Each open tab keeps its own state: the list shown, the open entry, the
// form of its arguments, its answer and the scrolls. The state is a map from
// tab to a pointer: the views hold addresses of the form fields and scroll
// states across a frame, which map values would not allow, and the answer of
// a call must reach the tab it was sent for. A closed tab's state goes in
// Tick. Rejected: fields on the app's tab record (the pane's state would stay
// spread over the app), a slice in tab order (indexes shift when a tab
// closes), one slot for the current tab (a half-typed form and the call in
// flight would be lost on a tab switch). Revisit when tabs get stable ids,
// or when the explorer state must outlive its tab.
//
// The answer of a call writes into the state pointer taken before the work
// started, never into a fresh lookup: the tab may be closed meanwhile, and a
// lookup would make a state for it again that nothing prunes. Only View and
// the click closures make a state; Calling and Tick only read, so asking
// about a tab that never showed the pane leaves no trace. What the server
// announced, its description and the session are the tab's connection, read
// and never written here.
//
// The call form of a tool, opened from the composer, is not in the map: its
// form lives in the closures of the dialog the application shows, and the
// application owns the one open dialog.
//
// There is no Typing or Keys method: the explorer's form fields were never
// among the fields the window's text-focus check knows, so a shortcut typed
// while one has focus still runs today. Adding them would change which keys
// reach the shortcuts, which a refactor must not do.
package mcp

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the MCP pane and the state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of the pane between frames. A form exists
// only while an entry is open that has arguments: a tool or a prompt.
type state struct {
	explorer mcpexplorer.State  // the list shown, the open entry, its answer
	form     *mcpform.Form      // the argument form of the open entry, nil for none
	result   widget.CodeView    // the last answer
	list     widget.ScrollState // scroll of the explorer's list
	detail   widget.ScrollState // scroll of the open entry
}

// New returns a component asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made on first use.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{explorer: mcpexplorer.New()}
		c.states[tab] = made
	}
	return made
}

// Calling reports whether a call of tab waits for the server's answer.
func (c *Component) Calling(tab *tabs.Tab) bool {
	made, ok := c.states[tab]
	return ok && made.explorer.IsCalling()
}

// Choose shows another list of the server.
func (c *Component) Choose(tab *tabs.Tab, kind mcpexplorer.Kind) {
	made := c.state(tab)
	made.explorer.Choose(kind)
	made.form = nil
	made.result.SetLines(nil)
}

// Open opens entry index of the shown list: tools and prompts get a form
// for their arguments, a resource has nothing to fill in.
func (c *Component) Open(tab *tabs.Tab, index int) {
	made := c.state(tab)
	made.explorer.Select(index)
	made.result.SetLines(nil)
	made.form = nil
	switch made.explorer.Kind {
	case mcpexplorer.Tools:
		made.form = mcpform.New(tab.Tools[index])
	case mcpexplorer.Prompts:
		prompt := tab.Prompts[index]
		made.form = mcpform.New(request.MCPTool{Name: prompt.Name, Schema: prompt.Schema()})
	}
}
