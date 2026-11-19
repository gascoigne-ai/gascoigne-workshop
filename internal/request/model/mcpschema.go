package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
)

// MCPTool is a tool an MCP server announced in its tools/list answer.
type MCPTool struct {
	Name        string
	Description string
	Schema      JSONSchema // the arguments; the zero schema when the server gave none
}

// Skeleton is a JSON value that fits schema, with the required properties
// first, to start editing from: "" for strings, 0 for numbers, false for
// booleans, the first value of an enum, [] and {} for the rest. Optional
// properties are left out unless they have a default.
func Skeleton(schema JSONSchema) string {
	encoded, err := marshalNoEscape(skeletonValue(schema))
	if err != nil {
		return "null"
	}
	return string(encoded)
}

func skeletonValue(schema JSONSchema) any {
	switch {
	case len(schema.Default) > 0:
		return json.RawMessage(schema.Default)
	case len(schema.Enum) > 0:
		return json.RawMessage(schema.Enum[0])
	case len(schema.Const) > 0:
		return json.RawMessage(schema.Const)
	}
	switch schema.Type {
	case SchemaString:
		return ""
	case SchemaInteger, SchemaNumber:
		return 0
	case SchemaBoolean:
		return false
	case SchemaArray:
		return []any{}
	case SchemaObject:
		return newSkeletonObject(schema.Properties)
	}
	return nil
}

// skeletonObject keeps the properties a skeleton shows, by name.
type skeletonObject struct {
	names   []string
	schemas map[string]JSONSchema
}

func newSkeletonObject(properties []SchemaProperty) skeletonObject {
	object := skeletonObject{schemas: map[string]JSONSchema{}}
	for _, property := range properties {
		if property.Required || len(property.Schema.Default) > 0 {
			object.names = append(object.names, property.Name)
			object.schemas[property.Name] = property.Schema
		}
	}
	sort.Strings(object.names)
	return object
}

func (object skeletonObject) MarshalJSON() ([]byte, error) {
	return writeObject(object.names, func(name string) ([]byte, error) {
		return json.Marshal(skeletonValue(object.schemas[name])) // members keep encoding/json's escapes
	})
}

// writeObject writes a JSON object with the members in the order of names,
// each value as encode gives it. A map would sort the keys.
func writeObject(names []string, encode func(name string) ([]byte, error)) ([]byte, error) {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, name := range names {
		if i > 0 {
			out.WriteByte(',')
		}
		key, _ := json.Marshal(name)
		value, err := encode(name)
		if err != nil {
			return nil, err
		}
		out.Write(key)
		out.WriteByte(':')
		out.Write(value)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

// MCPResource is a resource an MCP server announced in resources/list.
type MCPResource struct{ URI, Name string }

// MCPPrompt is a prompt an MCP server announced in prompts/list; Args are
// the names of its required arguments.
type MCPPrompt struct {
	Name string
	Args []string
}

// Schema is the JSON schema of the prompt's arguments: its required ones,
// all strings.
func (p MCPPrompt) Schema() JSONSchema {
	schema := JSONSchema{Type: SchemaObject}
	for _, name := range p.Args {
		schema.Properties = append(schema.Properties, SchemaProperty{Name: name, Required: true, Schema: JSONSchema{Type: SchemaString}})
	}
	return schema
}

// ArgsJSON is {"name":"",...} for a prompt's required arguments.
func (p MCPPrompt) ArgsJSON() string { return Skeleton(p.Schema()) }

// DecodeTools reads the tools of a tools/list result, in server order. An
// entry that is not a tool, or has no name, is skipped; a tool whose
// inputSchema is not an object keeps the zero Schema. The error is for a
// result that is not an object.
func DecodeTools(result []byte) ([]MCPTool, error) {
	entries, err := listEntries(result, "tools")
	if err != nil {
		return nil, err
	}
	var tools []MCPTool
	for _, entry := range entries {
		var raw struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		}
		if json.Unmarshal(entry, &raw) != nil || raw.Name == "" {
			continue
		}
		schema, err := DecodeJSONSchema(raw.InputSchema)
		if err != nil && len(raw.InputSchema) > 0 { // a broken schema leaves this tool without arguments, not the list without tools
			slog.Debug("tool schema skipped", "tool", raw.Name, "error", err)
		}
		tools = append(tools, MCPTool{Name: raw.Name, Description: raw.Description, Schema: schema})
	}
	return tools, nil
}

// DecodeResources reads the resources of a resources/list result; an
// entry without a uri is skipped.
func DecodeResources(result []byte) ([]MCPResource, error) {
	entries, err := listEntries(result, "resources")
	if err != nil {
		return nil, err
	}
	var resources []MCPResource
	for _, entry := range entries {
		var raw struct{ URI, Name string }
		if json.Unmarshal(entry, &raw) != nil || raw.URI == "" {
			continue
		}
		resources = append(resources, MCPResource{URI: raw.URI, Name: raw.Name})
	}
	return resources, nil
}

// DecodePrompts reads the prompts of a prompts/list result; an entry
// without a name is skipped.
func DecodePrompts(result []byte) ([]MCPPrompt, error) {
	entries, err := listEntries(result, "prompts")
	if err != nil {
		return nil, err
	}
	var prompts []MCPPrompt
	for _, entry := range entries {
		var raw struct {
			Name      string `json:"name"`
			Arguments []struct {
				Name     string `json:"name"`
				Required bool   `json:"required"`
			} `json:"arguments"`
		}
		if json.Unmarshal(entry, &raw) != nil || raw.Name == "" {
			continue
		}
		prompt := MCPPrompt{Name: raw.Name}
		for _, argument := range raw.Arguments {
			if argument.Required {
				prompt.Args = append(prompt.Args, argument.Name)
			}
		}
		prompts = append(prompts, prompt)
	}
	return prompts, nil
}

// listEntries splits a list result into its entries under key, each
// left undecoded so one bad entry cannot drop the others.
func listEntries(result []byte, key string) ([]json.RawMessage, error) {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(result, &members); err != nil {
		return nil, fmt.Errorf("the %s result is not an object: %w", key, err)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(members[key], &entries); err != nil && members[key] != nil {
		return nil, fmt.Errorf("%s is not a list: %w", key, err)
	}
	return entries, nil
}
