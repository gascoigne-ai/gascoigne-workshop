package model

import (
	"bytes"
	"encoding/json"
	"sort"
)

// MCPTool is a tool an MCP server announced in its tools/list answer.
type MCPTool struct {
	Name        string
	Description string
	Schema      map[string]any // the JSON schema of the arguments
}

// ParseTools reads the tools of a tools/list response (the whole JSON-RPC
// message, as the event log has it). It returns nil for anything else.
func ParseTools(message string) []MCPTool {
	var m struct {
		Result struct {
			Tools []struct {
				Name        string         `json:"name"`
				Description string         `json:"description"`
				InputSchema map[string]any `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(message), &m) != nil {
		return nil
	}
	var out []MCPTool
	for _, t := range m.Result.Tools {
		if t.Name != "" {
			out = append(out, MCPTool{Name: t.Name, Description: t.Description, Schema: t.InputSchema})
		}
	}
	return out
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

func skeletonValue(s map[string]any) any {
	if d, ok := s["default"]; ok {
		return d
	}
	if e, ok := s["enum"].([]any); ok && len(e) > 0 {
		return e[0]
	}
	if c, ok := s["const"]; ok {
		return c
	}
	switch kind(s) {
	case "string":
		return ""
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "array":
		return []any{}
	case "object":
		props, _ := s["properties"].(map[string]any)
		required := map[string]bool{}
		if r, ok := s["required"].([]any); ok {
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
func kind(s map[string]any) string {
	switch t := s["type"].(type) {
	case string:
		return t
	case []any:
		for _, x := range t {
			if name, ok := x.(string); ok && name != "null" {
				return name
			}
		}
	}
	if _, ok := s["properties"]; ok {
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
	var b bytes.Buffer
	b.WriteByte('{')
	for i, n := range o.names {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(n)
		sub, _ := o.values[n].(map[string]any)
		val, err := json.Marshal(skeletonValue(sub))
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// MCPResource is a resource an MCP server announced in resources/list.
type MCPResource struct{ URI, Name string }

// MCPPrompt is a prompt an MCP server announced in prompts/list; Args are
// the names of its required arguments.
type MCPPrompt struct {
	Name string
	Args []string
}

// ParseResources reads the resources of a resources/list response.
func ParseResources(message string) []MCPResource {
	var m struct {
		Result struct {
			Resources []struct{ URI, Name string } `json:"resources"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(message), &m) != nil {
		return nil
	}
	var out []MCPResource
	for _, r := range m.Result.Resources {
		if r.URI != "" {
			out = append(out, MCPResource{URI: r.URI, Name: r.Name})
		}
	}
	return out
}

// ParsePrompts reads the prompts of a prompts/list response.
func ParsePrompts(message string) []MCPPrompt {
	var m struct {
		Result struct {
			Prompts []struct {
				Name      string `json:"name"`
				Arguments []struct {
					Name     string `json:"name"`
					Required bool   `json:"required"`
				} `json:"arguments"`
			} `json:"prompts"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(message), &m) != nil {
		return nil
	}
	var out []MCPPrompt
	for _, p := range m.Result.Prompts {
		if p.Name == "" {
			continue
		}
		pr := MCPPrompt{Name: p.Name}
		for _, a := range p.Arguments {
			if a.Required {
				pr.Args = append(pr.Args, a.Name)
			}
		}
		out = append(out, pr)
	}
	return out
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

func toAny(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}
