package assertions

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Run checks the assertions of tab against its shown response after a send:
// it keeps the results, stores the captured values as variables and adds a
// summary to the status line of a response that did not fail.
func (c *Component) Run(tab *tabs.Tab) {
	state := c.state(tab)
	state.results = assert.Run(tab.Req.Tests, tab.Resp)
	for _, captured := range assert.Captures(tab.Req.Tests, tab.Resp) {
		c.deps.Vars.SetVar(tab.Req, captured.Name, captured.Value)
	}
	if len(state.results) == 0 {
		return
	}
	if sum := assert.Summary(state.results); sum != "" && tab.Resp != nil && tab.Resp.Err == "" {
		c.deps.Status.SetStatus(c.deps.Status.Status() + "  Tests: " + sum)
	}
}

// Results are what the last check of tab found, nil when it was never
// checked. It makes no state.
func (c *Component) Results(tab *tabs.Tab) []assert.Result {
	state, ok := c.states[tab]
	if !ok {
		return nil
	}
	return state.results
}
