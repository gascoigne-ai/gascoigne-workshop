package ui

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"strings"
	"time"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// samplingRetry is how often a sampling request checks whether the
// dialog it wants to show is free.
const samplingRetry = 100 * time.Millisecond

// sampleViaDialog is the app's reqport.Sampler: it shows the server's
// request and the person types the reply, or declines. It runs on the
// session's goroutine and blocks; the dialog itself is opened on the UI
// goroutine. A request that arrives while another dialog is open waits
// for it to close.
func (a *App) sampleViaDialog(ctx context.Context, request reqport.SamplingRequest) (reqport.SamplingResult, bool) {
	answer := make(chan reqport.SamplingResult, 1)
	declined := make(chan struct{}, 1)
	var shown *dialog
	for shown == nil {
		opened := make(chan *dialog, 1)
		a.bus.Dispatch(func() { opened <- a.openSamplingDialog(request, answer, declined) })
		select {
		case shown = <-opened:
		case <-ctx.Done():
			return reqport.SamplingResult{}, false
		}
		if shown != nil {
			break
		}
		select {
		case <-time.After(samplingRetry):
		case <-ctx.Done():
			return reqport.SamplingResult{}, false
		}
	}
	select {
	case result := <-answer:
		return result, true
	case <-declined:
		return reqport.SamplingResult{}, false
	case <-ctx.Done():
		a.bus.Dispatch(func() { // the session ended: take the question off the screen
			if a.dialog == shown {
				a.closeDialog()
			}
		})
		return reqport.SamplingResult{}, false
	}
}

// openSamplingDialog shows the request when no other dialog is open and
// returns it, nil when the screen is busy.
func (a *App) openSamplingDialog(request reqport.SamplingRequest, answer chan<- reqport.SamplingResult, declined chan<- struct{}) *dialog {
	if a.dialog != nil {
		return nil
	}
	var body strings.Builder
	body.WriteString("The MCP server asks for a model reply. Type one, or decline.")
	if request.SystemPrompt != "" {
		body.WriteString("\n\nSystem: " + request.SystemPrompt)
	}
	for _, message := range request.Messages {
		body.WriteString("\n\n" + message.Role + ": " + message.Text)
	}
	input := NewTextField("")
	input.Focused = true
	a.dialog = &dialog{DialogState: DialogState{
		Title: "Sampling request",
		Body:  body.String(),
		Input: &input,
		Buttons: []DialogButton{
			{Label: "Decline", Run: func() { a.closeDialog(); declined <- struct{}{} }},
			{Label: "Send reply", Primary: true, Run: func() {
				a.closeDialog()
				answer <- reqport.SamplingResult{Text: input.Text, Model: "user"}
			}},
		},
	}}
	return a.dialog
}

// openToolForm shows the form of tool; a tool with no properties is
// composed at once.
func (a *App) openToolForm(t *reqTab, tool model.MCPTool) {
	form := mcpform.New(tool)
	if form.Empty() {
		a.composeCall(t, tool)
		return
	}
	a.dialog = &dialog{
		DialogState: DialogState{
			Title: "Call " + tool.Name,
			Buttons: []DialogButton{
				{Label: "Cancel", Run: a.closeDialog},
				{Label: "Fill composer", Primary: true, Run: func() { a.fillFromForm(t, form) }},
			},
		},
		form: form,
	}
}

// fillFromForm puts the call the form describes into the composer, or
// keeps the form open and says what is wrong.
func (a *App) fillFromForm(t *reqTab, form *mcpform.Form) {
	command, err := form.Command()
	if err != nil {
		form.Problem = err.Error()
		return
	}
	a.closeDialog()
	t.msg = NewTextField(command)
	t.msg.Focused = true
}

// Submit makes the App the host of the tool form: Enter in a field of the
// dialog fills the composer, in the explorer's form it calls.
func (a *App) Submit() {
	if a.dialog != nil && a.dialog.form != nil {
		a.fillFromForm(a.cur(), a.dialog.form)
		return
	}
	a.exploreCall(a.cur()) // Enter in a field of the explorer's form
}
