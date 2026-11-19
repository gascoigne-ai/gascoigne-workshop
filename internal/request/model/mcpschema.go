package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// MCPTool is a tool an MCP server announced in its tools/list answer.
type MCPTool struct {
	Name        string
	Description string
	Schema      map[string]any // the JSON schema of the arguments
}

// Skeleton is a JSON value that fits schema, with the required properties
// first, to start editing from: "" for strings, 0 for numbers, false for
// booleans, the first value of an enum, [] and {} for the rest. Optional
// properties are left out unless they have a default.
func Skeleton(schema map[string]any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(skeletonValue(schema))
	return string(bytes.TrimSpace(buf.Bytes()))
}

func skeletonValue(other map[string]any) any {
	if d, ok := other["default"]; ok {
		return d
	}
	if e, ok := other["enum"].([]any); ok && len(e) > 0 {
		return e[0]
	}
	if c, ok := other["const"]; ok {
		return c
	}
	switch kind(other) {
	case "string":
		return ""
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "array":
		return []any{}
	case "object":
		props, _ := other["properties"].(map[string]any)
		required := map[string]bool{}
		if r, ok := other["required"].([]any); ok {
			for _, n := range r {
				if name, ok := n.(string); ok {
					required[name] = true
				}
			}
		}
		names := make([]string, 0, len(props))
		for n := range props {
			sub, _ := props[n].(map[string]any)
			if _, hasDefault := sub["default"]; required[n] || hasDefault {
				names = append(names, n)
			}
		}
		sort.Strings(names)
		return orderedObject{names: names, values: props}
	}
	return nil
}

// kind is the schema's type; a type list takes its first non-null entry.
func kind(other map[string]any) string {
	switch t := other["type"].(type) {
	case string:
		return t
	case []any:
		for _, x := range t {
			if name, ok := x.(string); ok && name != "null" {
				return name
			}
		}
	}
	if _, ok := other["properties"]; ok {
		return "object"
	}
	return ""
}

// orderedObject marshals the chosen properties in the given order.
type orderedObject struct {
	names  []string
	values map[string]any
}

func (o orderedObject) MarshalJSON() ([]byte, error) {
	return writeObject(o.names, func(name string) ([]byte, error) {
		sub, _ := o.values[name].(map[string]any)
		return json.Marshal(skeletonValue(sub))
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
func (p MCPPrompt) Schema() map[string]any {
	props := map[string]any{}
	for _, a := range p.Args {
		props[a] = map[string]any{"type": "string"}
	}
	return map[string]any{"type": "object", "properties": props, "required": toAny(p.Args)}
}

// ArgsJSON is {"name":"",...} for a prompt's required arguments.
func (p MCPPrompt) ArgsJSON() string { return Skeleton(p.Schema()) }

func toAny(names []string) []any {
	out := make([]any, len(names))
	for i, v := range names {
		out[i] = v
	}
	return out
}

// DecodeTools reads the tools of a tools/list result, in server order. An
// entry that is not a tool, or has no name, is skipped; a tool whose
// inputSchema is not an object keeps a nil Schema. The error is for a
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
		var schema map[string]any
		_ = json.Unmarshal(raw.InputSchema, &schema) // true or a string is no schema: the tool keeps none
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
