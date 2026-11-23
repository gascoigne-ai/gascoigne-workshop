// Package assertions holds the last two tabs of the request editor, Tests and
// Settings, and the Tests tab of the response pane: the area the assertions
// are written in, the timeout field and what the assertions found.
//
// The two panes of the request share one package because both are small and
// both write only fields of the request that no other pane edits (Tests,
// TimeoutSec, NoRedirects, InsecureTLS); the response's Tests pane lists
// what the request's assertions found, so it lives with them.
//
// Per-tab state is a map of pointers made on first use and dropped in Tick.
// Rejected: fields on the host's tab type (the reorganization takes them
// out), a slice in tab order (indexes shift when a tab closes) and one slot
// for the current tab (it loses the caret, the undo history and the results
// of the others). Revisit when tabs get stable ids or results must outlive a
// tab.
//
// The results are a stored cache of assert.Run(request.Tests, response) as of
// the last check, not computed at draw. A check happens when the assertions
// are typed (no captures) and when a send result arrives (captures and the
// status line). Cancelling a send replaces the response but not the results:
// that stale list is how it worked before and is filed, not fixed. Computing
// it per frame would change what users see. Revisit with a gutter mark per
// line, which needs a lookup by line.
//
// The response pane draws ResultsView: ui/response asks for it through a host
// slot, as the body does for its form table.
//
// There is no Keys method: a frame that only types text is skipped by the
// window's own text-focus check, and no key needs claiming (as in auth).
package assertions

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the Tests and Settings panes and the state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of these panes between frames.
type state struct {
	tests   widget.TextArea  // made from Req.Tests on first use, then the edit buffer: every edit writes back
	timeout widget.TextField // made from the timeout of the request on first use
	results []assert.Result  // what the last check found; nil is not checked
}

// New returns a component asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made from its request on first use.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{
			tests:   widget.NewTextArea(tab.Req.Tests),
			timeout: widget.NewTextField(timeoutText(tab.Req.TimeoutSec)),
		}
		c.states[tab] = made
	}
	return made
}

// edit applies an edit to the request of tab, which is the collection's.
func (c *Component) edit(tab *tabs.Tab, msg requestedit.Msg) {
	*tab.Req = requestedit.Update(*tab.Req, msg)
}
