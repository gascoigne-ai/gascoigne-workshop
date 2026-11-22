package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
)

// SchemaType is the JSON Schema "type" of a value; "" means none was given
// or it was not one of the known names.
type SchemaType string

// The JSON Schema types.
const (
	SchemaObject  SchemaType = "object"
	SchemaArray   SchemaType = "array"
	SchemaString  SchemaType = "string"
	SchemaNumber  SchemaType = "number"
	SchemaInteger SchemaType = "integer"
	SchemaBoolean SchemaType = "boolean"
	SchemaNull    SchemaType = "null"
)

// JSONValue is one JSON value kept as the exact bytes the server sent; nil
// means the value is absent.
type JSONValue json.RawMessage

// UnmarshalJSON keeps the bytes of the value as they are.
func (value *JSONValue) UnmarshalJSON(data []byte) error {
	*value = append((*value)[:0], data...)
	return nil
}

// Text is the value as a form field shows it: a string unquoted, anything
// else as compact JSON.
func (value JSONValue) Text() string {
	var text string
	if bytes.HasPrefix(value, []byte(`"`)) && json.Unmarshal(value, &text) == nil {
		return text
	}
	var compact bytes.Buffer
	if json.Compact(&compact, value) != nil {
		return string(value)
	}
	return compact.String()
}

// JSONSchema is the part of a JSON Schema the workshop reads: enough to build
// a form and a skeleton. It is decoded once, at the edge, by
// DecodeJSONSchema.
type JSONSchema struct {
	Type        SchemaType
	Description string
	Properties  []SchemaProperty // in document order, names unique
	Items       *JSONSchema
	Enum        []JSONValue
	Default     JSONValue
	Const       JSONValue
	Ref         string // the "$ref" of an OpenAPI schema; MCP ignores it
}

// SchemaProperty is one named property of an object schema.
type SchemaProperty struct {
	Name     string
	Required bool // listed in the object's "required"
	Schema   JSONSchema
}

// Bounds of a decoded schema, against a hostile server.
const (
	maxSchemaDepth      = 32
	maxSchemaProperties = 1000
	maxSchemaEnum       = 1000
)

// DecodeJSONSchema reads a JSON Schema. Only invalid JSON is an error; a
// value that is not an object (true, a string) is the zero schema. Parts
// beyond the bounds above are left out and logged at debug. Keywords without
// a field here (anyOf, $defs, pattern, format...) are ignored. A list of types
// takes the first that is not null, an unknown or non-string type is none, properties
// without a type make it an object, a required name without a property is
// dropped and a duplicate property keeps the later value.
func DecodeJSONSchema(data []byte) (JSONSchema, error) {
	if !json.Valid(data) {
		return JSONSchema{}, errors.New("decode JSON schema: invalid JSON")
	}
	return decodeSchema(data, 1)
}

// UnmarshalJSON decodes a schema inside a larger document; anything that is
// no schema object is the zero schema.
func (schema *JSONSchema) UnmarshalJSON(data []byte) error {
	decoded, err := DecodeJSONSchema(data)
	if err != nil {
		return err
	}
	*schema = decoded
	return nil
}

// rawField is one member of a JSON object whose value is not read yet.
type rawField struct {
	name  string
	value json.RawMessage
}

// readObject lists the members of a JSON object in document order; a later
// duplicate replaces the earlier value in place. It keeps at most limit
// members and reports whether it cut any. A value that is not an object has
// no members.
func readObject(data []byte, limit int) (fields []rawField, cut bool, err error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil, false, err
	}
	index := map[string]int{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, false, err
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, false, err
		}
		name := token.(string) // an object key is always a string
		if at, seen := index[name]; seen {
			fields[at].value = value
			continue
		}
		if len(fields) >= limit {
			cut = true
			continue
		}
		index[name] = len(fields)
		fields = append(fields, rawField{name, value})
	}
	return fields, cut, nil
}

// readArray lists the first limit elements of a JSON array and reports
// whether it cut any. A value that is not an array has no elements.
func readArray(data []byte, limit int) (elements []json.RawMessage, cut bool, err error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if token, err := decoder.Token(); err != nil || token != json.Delim('[') {
		return nil, false, err
	}
	for decoder.More() {
		var element json.RawMessage
		if err := decoder.Decode(&element); err != nil {
			return nil, false, err
		}
		if len(elements) >= limit {
			cut = true
			continue
		}
		elements = append(elements, element)
	}
	return elements, cut, nil
}

func isObject(data []byte) bool {
	return bytes.HasPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte("{"))
}

// decodeSchema reads the schema at the given depth (the root is 1).
func decodeSchema(data []byte, depth int) (JSONSchema, error) {
	var schema JSONSchema
	fields, _, err := readObject(data, maxSchemaProperties)
	if err != nil {
		return JSONSchema{}, fmt.Errorf("decode JSON schema: %w", err)
	}
	var required []string
	for _, field := range fields {
		switch field.name {
		case "type":
			schema.Type = decodeSchemaType(field.value)
		case "description":
			_ = json.Unmarshal(field.value, &schema.Description) // not a string: none
		case "$ref":
			_ = json.Unmarshal(field.value, &schema.Ref)
		case "required":
			_ = json.Unmarshal(field.value, &required)
		case "default":
			schema.Default = JSONValue(field.value)
		case "const":
			schema.Const = JSONValue(field.value)
		case "enum":
			schema.Enum = decodeSchemaEnum(field.value)
		case "properties":
			schema.Properties, err = decodeSchemaProperties(field.value, depth)
		case "items":
			schema.Items, err = decodeSchemaItems(field.value, depth)
		}
		if err != nil {
			return JSONSchema{}, err
		}
	}
	markRequired(schema.Properties, required)
	if schema.Type == "" && len(schema.Properties) > 0 {
		schema.Type = SchemaObject
	}
	return schema, nil
}

var knownSchemaTypes = map[SchemaType]bool{
	SchemaObject: true, SchemaArray: true, SchemaString: true, SchemaNumber: true,
	SchemaInteger: true, SchemaBoolean: true, SchemaNull: true,
}

// decodeSchemaType reads "type": a name, or a list of names of which the
// first that is not null counts (a nullable type).
func decodeSchemaType(data []byte) SchemaType {
	var names []string
	if json.Unmarshal(data, &names) != nil {
		var name string
		if json.Unmarshal(data, &name) != nil {
			return ""
		}
		names = []string{name}
	}
	for _, name := range names {
		if name != string(SchemaNull) {
			return knownType(name)
		}
	}
	if len(names) > 0 {
		return SchemaNull
	}
	return ""
}

func knownType(name string) SchemaType {
	if kind := SchemaType(name); knownSchemaTypes[kind] {
		return kind
	}
	return ""
}

func decodeSchemaEnum(data []byte) []JSONValue {
	elements, cut, err := readArray(data, maxSchemaEnum)
	if err != nil {
		return nil
	}
	if cut {
		slog.Debug("JSON schema enum cut", "limit", maxSchemaEnum)
	}
	values := make([]JSONValue, len(elements))
	for i, element := range elements {
		values[i] = JSONValue(element)
	}
	return values
}

func decodeSchemaProperties(data []byte, depth int) ([]SchemaProperty, error) {
	if depth >= maxSchemaDepth {
		slog.Debug("JSON schema properties cut", "reason", "too deep", "limit", maxSchemaDepth)
		return nil, nil
	}
	fields, cut, err := readObject(data, maxSchemaProperties)
	if err != nil {
		return nil, fmt.Errorf("decode JSON schema properties: %w", err)
	}
	if cut {
		slog.Debug("JSON schema properties cut", "reason", "too many", "limit", maxSchemaProperties)
	}
	var properties []SchemaProperty
	for _, field := range fields {
		schema, err := decodeSchema(field.value, depth+1)
		if err != nil {
			return nil, err
		}
		properties = append(properties, SchemaProperty{Name: field.name, Schema: schema})
	}
	return properties, nil
}

func decodeSchemaItems(data []byte, depth int) (*JSONSchema, error) {
	if !isObject(data) {
		return nil, nil // true or a tuple of schemas: no single item schema
	}
	if depth >= maxSchemaDepth {
		slog.Debug("JSON schema items cut", "reason", "too deep", "limit", maxSchemaDepth)
		return nil, nil
	}
	items, err := decodeSchema(data, depth+1)
	if err != nil {
		return nil, err
	}
	return &items, nil
}

// markRequired sets Required on the properties named; a name without a
// property is dropped.
func markRequired(properties []SchemaProperty, names []string) {
	for _, name := range names {
		for i := range properties {
			if properties[i].Name == name {
				properties[i].Required = true
			}
		}
	}
}
