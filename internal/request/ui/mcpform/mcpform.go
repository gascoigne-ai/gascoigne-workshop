// Package mcpform is the dialog form of an MCP tool call: one input per
// property of the tool's schema, composing the call for the message box.
package mcpform

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// maxH is the tallest the form's fields grow before they scroll.
const maxH = 360

// Form is the dialog content of a tool call: one input per property.
type Form struct {
	Tool    model.MCPTool
	Inputs  map[string]*TextField // text, number and JSON fields
	Choices map[string]string     // boolean and enum fields: the chosen value, "" for unset
	Problem string                // why the last attempt was refused
	fields  []model.ToolField
	scroll  ScrollState
}

// Host is what the form asks of the application.
type Host interface {
	// TextMenu makes an element open the text menu of field on right click.
	TextMenu(field *TextField, afterEdit func()) Option
	// Submit is called when Enter is pressed in a field.
	Submit()
}

// New is the form of tool, with each field at its default.
func New(tool model.MCPTool) *Form {
	form := &Form{Tool: tool, fields: model.ToolFields(tool.Schema), Inputs: map[string]*TextField{}, Choices: map[string]string{}}
	for _, field := range form.fields {
		switch field.Kind {
		case model.FieldBoolean, model.FieldEnum:
			form.Choices[field.Name] = initialChoice(field)
		default:
			input := NewTextField(field.Initial())
			form.Inputs[field.Name] = &input
		}
	}
	return form
}

// Empty reports a tool with no properties: it needs no form.
func (form *Form) Empty() bool { return len(form.fields) == 0 }

// initialChoice is where a boolean or enum starts: its default, else the
// first option when the argument is required, else unset (so it is left
// out of the call).
func initialChoice(field model.ToolField) string {
	switch {
	case field.Default != nil:
		return field.Initial()
	case field.Required && field.Kind == model.FieldBoolean:
		return "false"
	case field.Required && len(field.Enum) > 0:
		return model.ToolField{Kind: model.FieldEnum, Default: field.Enum[0]}.Initial()
	}
	return ""
}

// values are the form's entries by field name, as typed or chosen.
func (form *Form) values() map[string]string {
	out := map[string]string{}
	for name, input := range form.Inputs {
		out[name] = input.Text
	}
	for name, choice := range form.Choices {
		out[name] = choice
	}
	return out
}

// Arguments is the JSON object of the entries, or the error saying which
// value is wrong.
func (form *Form) Arguments() (string, error) {
	return model.Arguments(form.fields, form.values())
}

// Command is the composer text of the call the form describes, or the
// error saying which value is wrong.
func (form *Form) Command() (string, error) {
	args, err := form.Arguments()
	if err != nil {
		return "", err
	}
	return "call " + form.Tool.Name + " " + args, nil
}

// View is the fields of the form, scrolling when there are many.
func (form *Form) View(host Host) Node {
	return Col(Gap(8),
		If(form.Tool.Description != "", Lines(form.Tool.Description, w.Muted)),
		Scroll(&form.scroll, MaxH(maxH), Gap(8),
			For(form.fields, func(_ int, field model.ToolField) Node { return form.fieldView(host, field) }),
		),
		If(form.Problem != "", Text(form.Problem, Small(), Color(w.Error))),
	)
}

// fieldView is one field: its name (starred when required), the
// description, and the input of its kind.
func (form *Form) fieldView(host Host, field model.ToolField) Node {
	name := field.Name
	if field.Required {
		name += " *"
	}
	return Col(Gap(2),
		Row(H(w.LabelH), Gap(8), Text(name, Small(), Bold()), If(field.Description != "", Text(field.Description, Small(), Muted(), Ellipsis(), Grow(1)))),
		form.input(host, field),
	)
}

func (form *Form) input(host Host, field model.ToolField) Node {
	switch field.Kind {
	case model.FieldBoolean:
		on := form.Choices[field.Name] == "true"
		return Row(H(w.RowH), Gap(8), OnClick(func() { form.Choices[field.Name] = boolText(!on) }),
			Checkbox(on),
			Text(form.boolHint(field), Small(), Muted()),
		)
	case model.FieldEnum:
		return form.enumChoice(field)
	}
	input := form.Inputs[field.Name]
	return Field(input, H(w.RowH), Bg(w.Sunken), OnEnter(host.Submit), host.TextMenu(input, nil))
}

func boolText(on bool) string {
	if on {
		return "true"
	}
	return "false"
}

// boolHint says whether an untouched optional boolean is sent.
func (form *Form) boolHint(field model.ToolField) string {
	if form.Choices[field.Name] == "" {
		return "not sent"
	}
	return "sent as " + form.Choices[field.Name]
}

// enumChoice is a row of buttons, one per option; an optional field also
// has a "none" one that leaves it out.
func (form *Form) enumChoice(field model.ToolField) Node {
	options := []string{}
	if !field.Required {
		options = append(options, "")
	}
	for _, option := range field.Enum {
		options = append(options, model.ToolField{Kind: model.FieldEnum, Default: option}.Initial())
	}
	return Row(H(w.RowH), Gap(6),
		For(options, func(_ int, option string) Node {
			label := option
			if option == "" {
				label = "none"
			}
			pick := OnClick(func() { form.Choices[field.Name] = option })
			if form.Choices[field.Name] == option {
				return PrimaryButton(label, H(w.RowH), pick)
			}
			return Button(label, H(w.RowH), pick)
		}),
	)
}
