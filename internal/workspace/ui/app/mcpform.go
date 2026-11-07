package ui

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"strings"
	"time"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
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
		select {
		case a.later <- func() { opened <- a.openSamplingDialog(request, answer, declined) }:
		case <-ctx.Done():
			return reqport.SamplingResult{}, false
		}
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
		select { // the session ended: take the question off the screen
		case a.later <- func() {
			if a.dialog == shown {
				a.closeDialog()
			}
		}:
		default:
		}
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
	a.dialog = &dialog{
		title: "Sampling request",
		body:  body.String(),
		input: &input,
		buttons: []dialogButton{
			{label: "Decline", run: func() { a.closeDialog(); declined <- struct{}{} }},
			{label: "Send reply", primary: true, run: func() {
				a.closeDialog()
				answer <- reqport.SamplingResult{Text: input.Text, Model: "user"}
			}},
		},
	}
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
		title: "Call " + tool.Name,
		form:  form,
		buttons: []dialogButton{
			{label: "Cancel", run: a.closeDialog},
			{label: "Fill composer", primary: true, run: func() { a.fillFromForm(t, form) }},
		},
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

// Submit makes the App the host of the tool form: Enter in a field fills
// the composer.
func (a *App) Submit() { a.fillFromForm(a.cur(), a.dialog.form) }
