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
	Enum        []any
	Default     any
}

// ToolFields lists the properties of a tool's input schema as form
// fields: the required ones first, then the rest, each group by name.
func ToolFields(schema map[string]any) []ToolField {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return nil
	}
	required := map[string]bool{}
	if list, ok := schema["required"].([]any); ok {
		for _, entry := range list {
			if name, ok := entry.(string); ok {
				required[name] = true
			}
		}
	}
	fields := make([]ToolField, 0, len(properties))
	for name, raw := range properties {
		property, _ := raw.(map[string]any)
		description, _ := property["description"].(string)
		field := ToolField{Name: name, Description: description, Required: required[name], Default: property["default"]}
		field.Enum, _ = property["enum"].([]any)
		field.Kind = fieldKind(property)
		fields = append(fields, field)
	}
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].Required != fields[j].Required {
			return fields[i].Required
		}
		return fields[i].Name < fields[j].Name
	})
	return fields
}

// fieldKind is the form input a property gets.
func fieldKind(property map[string]any) FieldKind {
	if enum, ok := property["enum"].([]any); ok && len(enum) > 0 {
		return FieldEnum
	}
	switch kind(property) {
	case "string":
		return FieldString
	case "number":
		return FieldNumber
	case "integer":
		return FieldInteger
	case "boolean":
		return FieldBoolean
	}
	return FieldJSON
}

// Initial is the text a form starts the field with: its default.
func (field ToolField) Initial() string {
	if field.Default == nil {
		if field.Kind == FieldBoolean {
			return "false"
		}
		return ""
	}
	return valueText(field.Default)
}

// valueText is a JSON value as the text a user would type for it.
func valueText(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

// Arguments builds the JSON object of a tool call from the form's values
// (by field name, as typed). Empty optional values are left out; a value
// that does not fit its field is an error naming the field.
func Arguments(fields []ToolField, values map[string]string) (string, error) {
	names := make([]string, 0, len(fields))
	parsed := map[string]any{}
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
	var out bytes.Buffer
	out.WriteByte('{')
	for i, name := range names {
		if i > 0 {
			out.WriteByte(',')
		}
		key, _ := json.Marshal(name)
		value, err := marshalNoEscape(parsed[name])
		if err != nil {
			return "", fmt.Errorf("%s: %w", name, err)
		}
		out.Write(key)
		out.WriteByte(':')
		out.Write(value)
	}
	out.WriteByte('}')
	return out.String(), nil
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
func (field ToolField) parse(text string) (value any, present bool, err error) {
	if field.Kind == FieldString { // a string keeps its spaces; only an empty optional one goes
		return text, text != "" || field.Required, nil
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
			return whole, true, nil
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
		return number, true, nil
	case FieldBoolean:
		switch text {
		case "true":
			return true, true, nil
		case "false":
			return false, true, nil
		}
		return nil, false, fmt.Errorf("%s must be true or false", field.Name)
	case FieldEnum:
		for _, option := range field.Enum {
			if valueText(option) == text {
				return option, true, nil
			}
		}
		return nil, false, fmt.Errorf("%s must be one of %s", field.Name, field.enumText())
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, false, fmt.Errorf("%s is not valid JSON: %v", field.Name, err)
	}
	return raw, true, nil
}

func (field ToolField) enumText() string {
	options := make([]string, len(field.Enum))
	for i, option := range field.Enum {
		options[i] = valueText(option)
	}
	return strings.Join(options, ", ")
}
