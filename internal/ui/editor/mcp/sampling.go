package mcp

import (
	"context"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// samplingRetry is how often a sampling request checks whether the
// dialog it wants to show is free.
const samplingRetry = 100 * time.Millisecond

// Sample is the request.Sampler of the application: it shows the server's
// request and the person types the reply, or declines. It runs on the
// session's goroutine and blocks; the dialog itself is opened on the UI
// goroutine. A request that arrives while another dialog is open waits for
// it to close. It touches nothing but Deps.Bus.Dispatch and its own
// channels, never the states of the tabs, which belong to the UI goroutine.
func (c *Component) Sample(ctx context.Context, call request.SamplingRequest) (request.SamplingResult, bool) {
	answer := make(chan request.SamplingResult, 1)
	declined := make(chan struct{}, 1)
	var closeDialog func()
	for closeDialog == nil {
		opened := make(chan func(), 1)
		c.deps.Bus.Dispatch(func() { opened <- c.openSampling(call, answer, declined) })
		select {
		case closeDialog = <-opened:
		case <-ctx.Done():
			return request.SamplingResult{}, false
		}
		if closeDialog != nil {
			break
		}
		select {
		case <-time.After(samplingRetry):
		case <-ctx.Done():
			return request.SamplingResult{}, false
		}
	}
	select {
	case result := <-answer:
		return result, true
	case <-declined:
		return request.SamplingResult{}, false
	case <-ctx.Done():
		c.deps.Bus.Dispatch(closeDialog) // the session ended: take the question off the screen
		return request.SamplingResult{}, false
	}
}

// openSampling shows the request when no other dialog is open and returns
// what closes it, nil when the screen is busy.
func (c *Component) openSampling(call request.SamplingRequest, answer chan<- request.SamplingResult, declined chan<- struct{}) func() {
	if c.deps.Dialogs.Busy() {
		return nil
	}
	var body strings.Builder
	body.WriteString("The MCP server asks for a model reply. Type one, or decline.")
	if call.SystemPrompt != "" {
		body.WriteString("\n\nSystem: " + call.SystemPrompt)
	}
	for _, message := range call.Messages {
		body.WriteString("\n\n" + message.Role + ": " + message.Text)
	}
	input := widget.NewTextField("")
	input.Focused = true
	var closeDialog func()
	closeDialog = c.deps.Dialogs.Open(widget.DialogState{
		Title: "Sampling request",
		Body:  body.String(),
		Input: &input,
		Buttons: []widget.DialogButton{
			{Label: "Decline", Run: func() { closeDialog(); declined <- struct{}{} }},
			{Label: "Send reply", Primary: true, Run: func() {
				closeDialog()
				answer <- request.SamplingResult{Text: input.Text, Model: "user"}
			}},
		},
	}, nil)
	return closeDialog
}
