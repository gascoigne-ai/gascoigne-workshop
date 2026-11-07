package ui

import (
	"context"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// toolFormMaxH is the tallest the form's fields grow before they scroll.
const toolFormMaxH = 360

// toolForm is the dialog content of a tool call: one input per property.
type toolForm struct {
	tool    proto.MCPTool
	fields  []proto.ToolField
	inputs  map[string]*TextField // text, number and JSON fields
	choices map[string]string     // boolean and enum fields: the chosen value, "" for unset
	problem string                // why the last attempt was refused
	scroll  ScrollState
}

// openToolForm shows the form of tool; a tool with no properties is
// composed at once.
func (a *App) openToolForm(t *reqTab, tool proto.MCPTool) {
	fields := proto.ToolFields(tool.Schema)
	if len(fields) == 0 {
		a.composeCall(t, tool)
		return
	}
	form := &toolForm{tool: tool, fields: fields, inputs: map[string]*TextField{}, choices: map[string]string{}}
	for _, field := range fields {
		switch field.Kind {
		case proto.FieldBoolean, proto.FieldEnum:
			form.choices[field.Name] = initialChoice(field)
		default:
			input := NewTextField(field.Initial())
			form.inputs[field.Name] = &input
		}
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

// initialChoice is where a boolean or enum starts: its default, else the
// first option when the argument is required, else unset (so it is left
// out of the call).
func initialChoice(field proto.ToolField) string {
	switch {
	case field.Default != nil:
		return field.Initial()
	case field.Required && field.Kind == proto.FieldBoolean:
		return "false"
	case field.Required && len(field.Enum) > 0:
		return proto.ToolField{Kind: proto.FieldEnum, Default: field.Enum[0]}.Initial()
	}
	return ""
}

// values are the form's entries by field name, as typed or chosen.
func (form *toolForm) values() map[string]string {
	out := map[string]string{}
	for name, input := range form.inputs {
		out[name] = input.Text
	}
	for name, choice := range form.choices {
		out[name] = choice
	}
	return out
}

// fillFromForm puts the call the form describes into the composer, or
// keeps the form open and says what is wrong.
func (a *App) fillFromForm(t *reqTab, form *toolForm) {
	args, err := proto.Arguments(form.fields, form.values())
	if err != nil {
		form.problem = err.Error()
		return
	}
	a.closeDialog()
	t.msg = NewTextField("call " + form.tool.Name + " " + args)
	t.msg.Focused = true
}

// toolFormView is the fields of the form, scrolling when there are many.
func (a *App) toolFormView(form *toolForm) Node {
	return Col(Gap(8),
		If(form.tool.Description != "", Lines(form.tool.Description, w.Muted)),
		Scroll(&form.scroll, MaxH(toolFormMaxH), Gap(8),
			For(form.fields, func(_ int, field proto.ToolField) Node { return a.toolFieldView(form, field) }),
		),
		If(form.problem != "", Text(form.problem, Small(), Color(w.Error))),
	)
}

// toolFieldView is one field: its name (starred when required), the
// description, and the input of its kind.
func (a *App) toolFieldView(form *toolForm, field proto.ToolField) Node {
	name := field.Name
	if field.Required {
		name += " *"
	}
	return Col(Gap(2),
		Row(H(w.LabelH), Gap(8), Text(name, Small(), Bold()), If(field.Description != "", Text(field.Description, Small(), Muted(), Ellipsis(), Grow(1)))),
		a.toolInput(form, field),
	)
}

func (a *App) toolInput(form *toolForm, field proto.ToolField) Node {
	switch field.Kind {
	case proto.FieldBoolean:
		on := form.choices[field.Name] == "true"
		return Row(H(w.RowH), Gap(8), OnClick(func() { form.choices[field.Name] = boolText(!on) }),
			Checkbox(on),
			Text(form.boolHint(field), Small(), Muted()),
		)
	case proto.FieldEnum:
		return a.enumChoice(form, field)
	}
	input := form.inputs[field.Name]
	return Field(input, H(w.RowH), Bg(w.Sunken), OnEnter(func() { a.fillFromForm(a.cur(), form) }), a.textMenuOpt(input, nil))
}

func boolText(on bool) string {
	if on {
		return "true"
	}
	return "false"
}

// boolHint says whether an untouched optional boolean is sent.
func (form *toolForm) boolHint(field proto.ToolField) string {
	if form.choices[field.Name] == "" {
		return "not sent"
	}
	return "sent as " + form.choices[field.Name]
}

// enumChoice is a row of buttons, one per option; an optional field also
// has a "none" one that leaves it out.
func (a *App) enumChoice(form *toolForm, field proto.ToolField) Node {
	options := []string{}
	if !field.Required {
		options = append(options, "")
	}
	for _, option := range field.Enum {
		options = append(options, proto.ToolField{Kind: proto.FieldEnum, Default: option}.Initial())
	}
	return Row(H(w.RowH), Gap(6),
		For(options, func(_ int, option string) Node {
			label := option
			if option == "" {
				label = "none"
			}
			pick := OnClick(func() { form.choices[field.Name] = option })
			if form.choices[field.Name] == option {
				return PrimaryButton(label, H(w.CompactH), pick)
			}
			return Button(label, H(w.CompactH), pick)
		}),
	)
}

// samplingRetry is how often a sampling request checks whether the
// dialog it wants to show is free.
const samplingRetry = 100 * time.Millisecond

// sampleViaDialog is the app's proto.Sampler: it shows the server's
// request and the person types the reply, or declines. It runs on the
// session's goroutine and blocks; the dialog itself is opened on the UI
// goroutine. A request that arrives while another dialog is open waits
// for it to close.
func (a *App) sampleViaDialog(ctx context.Context, request proto.SamplingRequest) (proto.SamplingResult, bool) {
	answer := make(chan proto.SamplingResult, 1)
	declined := make(chan struct{}, 1)
	var shown *dialog
	for shown == nil {
		opened := make(chan *dialog, 1)
		select {
		case a.later <- func() { opened <- a.openSamplingDialog(request, answer, declined) }:
		case <-ctx.Done():
			return proto.SamplingResult{}, false
		}
		select {
		case shown = <-opened:
		case <-ctx.Done():
			return proto.SamplingResult{}, false
		}
		if shown != nil {
			break
		}
		select {
		case <-time.After(samplingRetry):
		case <-ctx.Done():
			return proto.SamplingResult{}, false
		}
	}
	select {
	case result := <-answer:
		return result, true
	case <-declined:
		return proto.SamplingResult{}, false
	case <-ctx.Done():
		select { // the session ended: take the question off the screen
		case a.later <- func() {
			if a.dialog == shown {
				a.closeDialog()
			}
		}:
		default:
		}
		return proto.SamplingResult{}, false
	}
}

// openSamplingDialog shows the request when no other dialog is open and
// returns it, nil when the screen is busy.
func (a *App) openSamplingDialog(request proto.SamplingRequest, answer chan<- proto.SamplingResult, declined chan<- struct{}) *dialog {
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
				answer <- proto.SamplingResult{Text: input.Text, Model: "user"}
			}},
		},
	}
	return a.dialog
}
