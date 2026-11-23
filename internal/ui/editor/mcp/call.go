package mcp

import (
	"context"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// callTimeout is how long the explorer waits for the answer of a call.
const callTimeout = 60 * time.Second

// Call sends the open entry of tab and shows the answer when it comes. A
// wrong value in the form is named on the form instead.
func (c *Component) Call(tab *tabs.Tab) {
	made, ok := c.states[tab]
	if !ok {
		return
	}
	requester, ok := tab.Session().(request.Requester)
	if !ok || made.explorer.Selected == mcpexplorer.None || made.explorer.IsCalling() {
		return
	}
	text, err := command(tab, made)
	if err != nil {
		made.form.Problem = err.Error() // only a form can hold a wrong value
		return
	}
	if made.form != nil {
		made.form.Problem = ""
	}
	made.explorer.Begin()
	c.deps.Bus.Run(func() func() {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()
		result, err := requester.Request(ctx, text)
		return func() { // on the UI goroutine, into the state the call was made from
			made.explorer.Finish(result, err)
			made.result.SetLines(c.resultLines(made.explorer))
		}
	})
}
