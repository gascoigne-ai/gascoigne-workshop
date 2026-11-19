package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// FieldKind is how a tool argument is entered in a form.
type FieldKind int

// The kinds of form field.
const (
	FieldString FieldKind = iota
	FieldNumber
	FieldInteger
	FieldBoolean
	FieldEnum
	FieldJSON // objects, arrays and anything else: typed as JSON
)

// ToolField is one argument of a tool, as a form input.
type ToolField struct {
	Name        string
	Description string
	Kind        FieldKind
	Required    bool
	Enum        []JSONValue
	Default     JSONValue // nil when the schema gives none
}

// ToolFields lists the properties of a tool's input schema as form
// fields: the required ones first, then the rest, each group by name.
func ToolFields(schema JSONSchema) []ToolField {
	fields := make([]ToolField, 0, len(schema.Properties))
	for _, property := range schema.Properties {
		field := ToolField{
			Name: property.Name, Description: property.Schema.Description, Required: property.Required,
			Kind: fieldKind(property.Schema), Enum: property.Schema.Enum, Default: property.Schema.Default,
		}
		if string(field.Default) == "null" { // a null default is no default
			field.Default = nil
		}
		fields = append(fields, field)
	}
	sort.Slice(fields, func(index, other int) bool {
		if fields[index].Required != fields[other].Required {
			return fields[index].Required
		}
		return fields[index].Name < fields[other].Name
	})
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// fieldKind is the form input a property gets.
func fieldKind(property JSONSchema) FieldKind {
	if len(property.Enum) > 0 {
		return FieldEnum
	}
	switch property.Type {
	case SchemaString:
		return FieldString
	case SchemaNumber:
		return FieldNumber
	case SchemaInteger:
		return FieldInteger
	case SchemaBoolean:
		return FieldBoolean
	}
	return FieldJSON
}

// Initial is the text a form starts the field with: its default.
func (field ToolField) Initial() string {
	if len(field.Default) == 0 {
		if field.Kind == FieldBoolean {
			return "false"
		}
		return ""
	}
	return field.Default.Text()
}

// Arguments builds the JSON object of a tool call from the form's values
// (by field name, as typed). Empty optional values are left out; a value
// that does not fit its field is an error naming the field.
func Arguments(fields []ToolField, values map[string]string) (string, error) {
	names := make([]string, 0, len(fields))
	parsed := map[string]json.RawMessage{}
	for _, field := range fields {
		value, present, err := field.parse(values[field.Name])
		if err != nil {
			return "", err
		}
		if present {
			names = append(names, field.Name)
			parsed[field.Name] = value
		}
	}
	out, err := writeObject(names, func(name string) ([]byte, error) {
		return parsed[name], nil
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// marshalNoEscape is json.Marshal without turning <, > and & into escapes.
func marshalNoEscape(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(out.Bytes()), nil
}

// parse turns the typed text into the JSON value of the field; present
// is false when an optional field was left empty.
func (field ToolField) parse(text string) (value json.RawMessage, present bool, err error) {
	if field.Kind == FieldString { // a string keeps its spaces; only an empty optional one goes
		return field.encode(text, text != "" || field.Required)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		if field.Required {
			return nil, false, fmt.Errorf("%s is required", field.Name)
		}
		return nil, false, nil
	}
	switch field.Kind {
	case FieldInteger:
		whole, err := strconv.ParseInt(text, 10, 64)
		if err == nil {
			return field.encode(whole, true)
		}
		if _, floatErr := strconv.ParseFloat(text, 64); floatErr == nil {
			return nil, false, fmt.Errorf("%s must be a whole number", field.Name)
		}
		return nil, false, fmt.Errorf("%s must be a number", field.Name)
	case FieldNumber:
		number, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, false, fmt.Errorf("%s must be a number", field.Name)
		}
		return field.encode(number, true)
	case FieldBoolean:
		switch text {
		case "true", "false":
			return json.RawMessage(text), true, nil
		}
		return nil, false, fmt.Errorf("%s must be true or false", field.Name)
	case FieldEnum:
		for _, option := range field.Enum {
			if option.Text() == text {
				return field.encode(json.RawMessage(option), true)
			}
		}
		return nil, false, fmt.Errorf("%s must be one of %s", field.Name, field.enumText())
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, false, fmt.Errorf("%s is not valid JSON: %v", field.Name, err)
	}
	return field.encode(raw, true)
}

// encode is the JSON of a typed value, or the error naming the field.
func (field ToolField) encode(value any, present bool) (json.RawMessage, bool, error) {
	encoded, err := marshalNoEscape(value)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", field.Name, err)
	}
	return encoded, present, nil
}

func (field ToolField) enumText() string {
	options := make([]string, len(field.Enum))
	for i, option := range field.Enum {
		options[i] = option.Text()
	}
	return strings.Join(options, ", ")
}
