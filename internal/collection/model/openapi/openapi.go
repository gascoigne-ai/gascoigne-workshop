// Package openapi turns an OpenAPI 3 document (JSON) into a folder of
// requests: one subfolder per tag, one request per operation, with its
// parameters and a JSON body to start from.
package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/miniyaml"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// BaseURLVar is the collection variable the imported URLs start with.
const BaseURLVar = "baseUrl"

// Result is what an import produces.
type Result struct {
	Folder  *model.Folder
	BaseURL string   // the first server of the document, "" when it names none
	Skipped []string // operations that could not be read, as "METHOD /path: why"
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
		Schemas    map[string]map[string]any `json:"schemas"`
		Parameters map[string]param          `json:"parameters"`
		Bodies     map[string]requestBody    `json:"requestBodies"`
	} `json:"components"`
}

type param struct {
	Ref      string         `json:"$ref"`
	Name     string         `json:"name"`
	In       string         `json:"in"`
	Required bool           `json:"required"`
	Schema   map[string]any `json:"schema"`
	Example  any            `json:"example"`
}

// requestBody is a body description; Ref points into
// components.requestBodies.
type requestBody struct {
	Ref     string `json:"$ref"`
	Content map[string]struct {
		Schema  map[string]any `json:"schema"`
		Example any            `json:"example"`
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
// that internal/miniyaml reads, which covers the documents API tools write).
func Parse(data []byte) (Result, error) {
	var d doc
	if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] != '{' {
		v, err := miniyaml.Parse(data)
		if err != nil {
			return Result{}, fmt.Errorf("not JSON, and not readable as YAML: %w", err)
		}
		if data, err = json.Marshal(v); err != nil {
			return Result{}, err
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
	root := &model.Folder{Name: title}
	byTag := map[string]*model.Folder{}
	paths := make([]string, 0, len(d.Paths))
	for p := range d.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var skipped []string
	for _, path := range paths {
		var item map[string]json.RawMessage
		if json.Unmarshal(d.Paths[path], &item) != nil { // "x-..." extensions are not path items
			continue
		}
		var shared []param
		if raw, ok := item["parameters"]; ok {
			_ = json.Unmarshal(raw, &shared)
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
	res := Result{Folder: root, Skipped: skipped}
	if len(d.Servers) > 0 {
		res.BaseURL = d.Servers[0].URL
	}
	return res, nil
}

// resolve follows a parameter's $ref into components.parameters.
func (d *doc) resolve(param param) param {
	if param.Ref == "" {
		return param
	}
	if name, ok := strings.CutPrefix(param.Ref, "#/components/parameters/"); ok {
		if r, ok := d.Components.Parameters[name]; ok {
			return r
		}
	}
	return param
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

// schema follows $ref chains into components.schemas, to a depth that
// stops a cycle, and does the same inside properties.
func (d *doc) schema(other map[string]any, depth int) map[string]any {
	if other == nil || depth > 6 {
		return other
	}
	if ref, ok := other["$ref"].(string); ok {
		if name, ok := strings.CutPrefix(ref, "#/components/schemas/"); ok {
			return d.schema(d.Components.Schemas[name], depth+1)
		}
		return nil
	}
	out := map[string]any{}
	for k, v := range other {
		out[k] = v
	}
	if props, ok := other["properties"].(map[string]any); ok {
		np := map[string]any{}
		for name, p := range props {
			if sub, ok := p.(map[string]any); ok {
				np[name] = d.schema(sub, depth+1)
			}
		}
		out["properties"] = np
	}
	return out
}

func buildRequest(docValue *doc, method, path string, op operation, shared []param) model.Request {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}
	if name == "" {
		name = strings.ToUpper(method) + " " + path
	}
	req := model.Request{Name: name, Protocol: "rest", Method: strings.ToUpper(method), BodyType: model.BodyNone}
	url := "{{" + BaseURLVar + "}}" + pathVars(path)
	var query []model.KV
	for _, raw := range append(append([]param{}, shared...), op.Parameters...) {
		p := docValue.resolve(raw)
		value := paramValue(p)
		switch p.In {
		case "query":
			query = append(query, model.KV{Name: p.Name, Value: value, Enabled: p.Required || value != ""})
		case "header":
			req.Headers = append(req.Headers, model.KV{Name: p.Name, Value: value, Enabled: p.Required})
		}
	}
	req.Params = query
	req.URL = model.WithParams(url, req.Params)
	if op.RequestBody != nil {
		body := docValue.resolveBody(*op.RequestBody)
		if mt, ok := body.Content["application/json"]; ok {
			req.BodyType = model.BodyJSON
			switch {
			case mt.Example != nil:
				b, _ := json.MarshalIndent(mt.Example, "", "  ")
				req.Body = string(b)
			case mt.Schema != nil:
				req.Body = indent(model.Skeleton(docValue.schema(mt.Schema, 0)))
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
func paramValue(param param) string {
	if param.Example != nil {
		return fmt.Sprint(param.Example)
	}
	if v, ok := param.Schema["default"]; ok {
		return fmt.Sprint(v)
	}
	if e, ok := param.Schema["enum"].([]any); ok && len(e) > 0 && param.Required {
		return fmt.Sprint(e[0])
	}
	return ""
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
