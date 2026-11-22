// Package openapi turns an OpenAPI 3 document (JSON) into a folder of
// requests: one subfolder per tag, one request per operation, with its
// parameters and a JSON body to start from.
package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/importers/miniyaml"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// BaseURLVar is the collection variable the imported URLs start with.
const BaseURLVar = "baseUrl"

// Result is what an import produces.
type Result struct {
	Folder  *request.Folder
	BaseURL string // the first server of the document, "" when it names none
	// Warnings name path items left out because they are not objects; an
	// "x-" extension is not one. They never fail the import.
	Warnings []string
	Skipped  []string // what could not be read: operations as "METHOD /path: why", shared parameters as "parameters of /path: why"
}

type doc struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title string `json:"title"`
	} `json:"info"`
	Servers []struct {
		URL string `json:"url"`
	} `json:"servers"`
	Paths      map[string]json.RawMessage `json:"paths"`
	Components struct {
		Schemas    map[string]request.JSONSchema `json:"schemas"`
		Parameters map[string]param              `json:"parameters"`
		Bodies     map[string]requestBody        `json:"requestBodies"`
	} `json:"components"`
}

type param struct {
	Ref      string             `json:"$ref"`
	Name     string             `json:"name"`
	In       string             `json:"in"`
	Required bool               `json:"required"`
	Schema   request.JSONSchema `json:"schema"`
	Example  request.JSONValue  `json:"example"`
}

// requestBody is a body description; Ref points into
// components.requestBodies.
type requestBody struct {
	Ref     string `json:"$ref"`
	Content map[string]struct {
		Schema  request.JSONSchema `json:"schema"`
		Example request.JSONValue  `json:"example"`
	} `json:"content"`
}

type operation struct {
	Tags        []string     `json:"tags"`
	Summary     string       `json:"summary"`
	OperationID string       `json:"operationId"`
	Parameters  []param      `json:"parameters"`
	RequestBody *requestBody `json:"requestBody"`
}

var methods = []string{"get", "post", "put", "patch", "delete", "head", "options"}

// Parse reads an OpenAPI 3 document in JSON or YAML (the subset of YAML
// that internal/feature/collections/importers/miniyaml reads, which covers the documents API tools write).
func Parse(data []byte) (Result, error) {
	var d doc
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] != '{' {
		var err error
		if data, err = miniyaml.ToJSON(data); err != nil {
			return Result{}, fmt.Errorf("not JSON, and not readable as YAML: %w", err)
		}
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return Result{}, fmt.Errorf("not JSON: %w", err)
	}
	if !strings.HasPrefix(d.OpenAPI, "3.") {
		return Result{}, errors.New("not an OpenAPI 3 document: no \"openapi\": \"3.x\"")
	}
	title := d.Info.Title
	if title == "" {
		title = "API"
	}
	root := &request.Folder{Name: title}
	byTag := map[string]*request.Folder{}
	paths := slices.Sorted(maps.Keys(d.Paths))
	var skipped, warnings []string
	for _, path := range paths {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(d.Paths[path], &item); err != nil {
			if !strings.HasPrefix(path, "x-") { // extensions are not path items
				warnings = append(warnings, fmt.Sprintf("path %s dropped: %v", path, err))
			}
			continue
		}
		var shared []param
		if raw, ok := item["parameters"]; ok {
			if err := json.Unmarshal(raw, &shared); err != nil { // the operations still import, without them
				skipped = append(skipped, fmt.Sprintf("parameters of %s: %v", path, err))
			}
		}
		for _, method := range methods {
			raw, ok := item[method]
			if !ok {
				continue
			}
			var op operation
			if err := json.Unmarshal(raw, &op); err != nil {
				skipped = append(skipped, fmt.Sprintf("%s %s: %v", strings.ToUpper(method), path, err))
				continue
			}
			req := buildRequest(&d, method, path, op, shared)
			folder := root
			if len(op.Tags) > 0 {
				tag := op.Tags[0]
				if byTag[tag] == nil {
					byTag[tag] = root.AddFolder(tag)
				}
				folder = byTag[tag]
			}
			req.Name = folder.UniqueName(req.Name)
			folder.AddRequest(req)
		}
	}
	if len(root.Requests)+len(root.Folders) == 0 {
		if len(skipped) > 0 { // nothing was readable: say why
			return Result{}, errors.New(skipped[0])
		}
		return Result{}, errors.New("the document has no operations")
	}
	res := Result{Folder: root, Skipped: skipped, Warnings: warnings}
	if len(d.Servers) > 0 {
		res.BaseURL = d.Servers[0].URL
	}
	return res, nil
}

// resolve follows a parameter's $ref into components.parameters.
func (d *doc) resolve(parameter param) param {
	if parameter.Ref == "" {
		return parameter
	}
	if name, ok := strings.CutPrefix(parameter.Ref, "#/components/parameters/"); ok {
		if r, ok := d.Components.Parameters[name]; ok {
			return r
		}
	}
	return parameter
}

// resolveBody follows a request body's $ref into components.requestBodies.
func (d *doc) resolveBody(body requestBody) requestBody {
	if name, ok := strings.CutPrefix(body.Ref, "#/components/requestBodies/"); ok {
		if target, found := d.Components.Bodies[name]; found {
			return target
		}
	}
	return body
}

// maxRefDepth is how many nested $ref levels resolveRefs follows before it
// stops; it ends a cycle.
const maxRefDepth = 6

// resolveRefs follows $ref into components.schemas, and the same inside
// properties. A reference outside the document, or to a missing schema, is
// the zero schema.
func (d *doc) resolveRefs(schema request.JSONSchema, depth int) request.JSONSchema {
	if depth > maxRefDepth {
		return schema
	}
	if schema.Ref != "" {
		name, local := strings.CutPrefix(schema.Ref, "#/components/schemas/")
		if !local {
			return request.JSONSchema{}
		}
		return d.resolveRefs(d.Components.Schemas[name], depth+1)
	}
	properties := make([]request.SchemaProperty, len(schema.Properties))
	for i, property := range schema.Properties {
		properties[i] = property
		properties[i].Schema = d.resolveRefs(property.Schema, depth+1)
	}
	schema.Properties = properties
	return schema
}

func buildRequest(spec *doc, method, path string, op operation, shared []param) request.Request {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = strings.ToUpper(method) + " " + path
	}
	req := request.Request{Name: name, Protocol: request.ProtocolREST, Method: strings.ToUpper(method), BodyType: request.BodyNone}
	url := "{{" + BaseURLVar + "}}" + pathVars(path)
	var query []request.KV
	for _, raw := range append(append([]param{}, shared...), op.Parameters...) {
		p := spec.resolve(raw)
		value := paramValue(p)
		switch p.In {
		case "query":
			query = append(query, request.KV{Name: p.Name, Value: value, Enabled: p.Required || value != ""})
		case "header":
			req.Headers = append(req.Headers, request.KV{Name: p.Name, Value: value, Enabled: p.Required})
		}
	}
	req.Params = query
	req.URL = request.WithParams(url, req.Params)
	if op.RequestBody != nil {
		body := spec.resolveBody(*op.RequestBody)
		if mt, ok := body.Content["application/json"]; ok {
			req.BodyType = request.BodyJSON
			switch {
			case hasValue(mt.Example):
				req.Body = indent(string(mt.Example))
			case mt.Schema.Type != "" || mt.Schema.Ref != "":
				req.Body = indent(request.Skeleton(spec.resolveRefs(mt.Schema, 0)))
			default:
				req.Body = "{}"
			}
		}
	}
	return req
}

// pathVars turns {id} into {{id}}, the form variables take here.
func pathVars(path string) string {
	var sb strings.Builder
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '{':
			sb.WriteString("{{")
		case '}':
			sb.WriteString("}}")
		default:
			sb.WriteByte(path[i])
		}
	}
	return sb.String()
}

// paramValue is a sample value for a parameter: its example, its default,
// or the first enum value; empty when the document gives none.
func paramValue(parameter param) string {
	switch {
	case hasValue(parameter.Example):
		return parameter.Example.Text()
	case hasValue(parameter.Schema.Default):
		return parameter.Schema.Default.Text()
	case len(parameter.Schema.Enum) > 0 && parameter.Required:
		return parameter.Schema.Enum[0].Text()
	}
	return ""
}

// hasValue is false for an absent value and for null.
func hasValue(value request.JSONValue) bool {
	return len(value) > 0 && string(value) != "null"
}

func indent(text string) string {
	var v any
	if json.Unmarshal([]byte(text), &v) != nil {
		return text
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return text
	}
	return string(b)
}
