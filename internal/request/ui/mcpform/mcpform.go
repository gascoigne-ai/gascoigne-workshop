// Package mcpform is the dialog form of an MCP tool call: one input per
// property of the tool's schema, composing the call for the message box.
package mcpform

import (
	"embed"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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

//go:embed form.vml
var templates embed.FS

var formTemplate = markup.MustLoad(templates, "form.vml")

// fieldRow is what the form shows of one field: exactly one of IsBool,
// IsEnum and Input says which input it has.
type fieldRow struct {
	Name        string // starred when required
	Description string
	IsBool      bool
	On          bool   // the box is ticked
	Hint        string // whether the boolean is sent, and as what
	Toggle      func()
	IsEnum      bool
	Options     []optionRow
	Input       *TextField // nil for a boolean or an enum
}

// optionRow is one button of an enum's row of choices.
type optionRow struct {
	Label    string
	Selected bool
	Pick     func()
}

// View is the fields of the form, scrolling when there are many.
func (form *Form) View(host Host) Node {
	fields := make([]fieldRow, len(form.fields))
	for i, field := range form.fields {
		fields[i] = form.fieldRow(field)
	}
	return formTemplate.MustBuild(markup.Scope{
		"description": form.Tool.Description,
		"scroll":      &form.scroll,
		"maxH":        maxH,
		"fields":      fields,
		"problem":     form.Problem,
		"submit":      host.Submit,
		"labelH":      w.LabelH,
		"rowH":        w.RowH,
	})
}

// fieldRow is one field: its name (starred when required), the
// description, and the input of its kind.
func (form *Form) fieldRow(field model.ToolField) fieldRow {
	row := fieldRow{Name: field.Name, Description: field.Description}
	if field.Required {
		row.Name += " *"
	}
	switch field.Kind {
	case model.FieldBoolean:
		on := form.Choices[field.Name] == "true"
		row.IsBool, row.On, row.Hint = true, on, form.boolHint(field)
		row.Toggle = func() { form.Choices[field.Name] = boolText(!on) }
	case model.FieldEnum:
		row.IsEnum, row.Options = true, form.enumOptions(field)
	default:
		row.Input = form.Inputs[field.Name]
	}
	return row
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

// enumOptions are the buttons of an enum, one per option; an optional
// field also has a "none" one that leaves it out.
func (form *Form) enumOptions(field model.ToolField) []optionRow {
	options := []string{}
	if !field.Required {
		options = append(options, "")
	}
	for _, option := range field.Enum {
		options = append(options, model.ToolField{Kind: model.FieldEnum, Default: option}.Initial())
	}
	rows := make([]optionRow, len(options))
	for i, option := range options {
		label := option
		if option == "" {
			label = "none"
		}
		rows[i] = optionRow{Label: label, Selected: form.Choices[field.Name] == option, Pick: func() { form.Choices[field.Name] = option }}
	}
	return rows
}
