// Package ws is the Messages pane of the request editor's response side:
// the log of a socket's traffic, its toolbar and the composer the user
// sends from. It serves WebSocket, SSE, gRPC streams and the JSON-RPC pane
// of an MCP server.
//
// The pane draws from what the tab already knows: its connection (state,
// session, the tools, resources and prompts an MCP server announced, the
// auto-reconnect policy) lives in workspace/tabs and is read from the
// *tabs.Tab the host hands out. The component owns only widget state, one
// entry per tab: the log and the message being composed. The connection
// lifecycle (connect, retry, the goroutines that deliver events) stays in
// the host: it serves gRPC streams and MCP too, owns goroutines and needs
// the auth gate, the call log and the status line. It reaches the log
// through Log and Note.
//
// A tab's entry is made on first use. Log and Note make it too, because an
// event of a tab that is not on screen must land in its log; an event that
// arrives for a tab that just closed makes an entry that Tick drops in the
// same frame. Lines and Typing never make one.
//
// The entries are a map keyed by *tabs.Tab: the current tab is looked up
// every frame, any open tab on an event or a failed send, and the whole
// map is walked only to prune it. A field on the host's per-tab struct and
// a single slot for the current tab were rejected: the first keeps widget
// state in the host, the second would lose the log and the draft of every
// tab in the background. Revisit when tabs get stable ids, when a log must
// outlive its tab, or when a tab holds two sessions.
//
// The log is not bounded: every event appends a line for as long as the
// connection lives. It needs a limit of its own and a row in
// architecture/limits.md; a ring buffer was left out because it would
// change what the pane shows.
//
// Which protocol shows a composer, whether it offers the { } button and
// what it hints is a row of the host's protocol table, asked through
// Deps.Composers; this package never switches on a protocol.
package ws

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the Messages pane and the state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of the pane between frames.
type state struct {
	log      widget.CodeView  // the message log; follows the end while it grows
	composer widget.TextField // the message being composed
}

// New returns a component asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made on first use. The log follows its end
// from the start, as a log should.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{}
		made.log.Follow = true
		c.states[tab] = made
	}
	return made
}

// Lines are the lines of the log of tab, nil when it has none. It makes no
// state.
func (c *Component) Lines(tab *tabs.Tab) []widget.Line {
	made, ok := c.states[tab]
	if !ok {
		return nil
	}
	return made.log.Lines
}

// Composer is the field tab's message is composed in; the host replaces its
// text through SetComposer and tests set it directly.
func (c *Component) Composer(tab *tabs.Tab) *widget.TextField {
	return &c.state(tab).composer
}
