// Package postman reads a Postman collection (v2.0 and v2.1, JSON) into a
// folder of requests: folders stay folders, headers, query, body modes and
// the common auth types carry over, collection variables are returned.
package postman

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Result is what an import produces.
type Result struct {
	Folder *model.Folder
	Vars   []model.KV
}

type kv struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
	Type     string `json:"type"`
	Src      any    `json:"src"`
}

type item struct {
	Name    string `json:"name"`
	Items   []item `json:"item"`
	Request *struct {
		Method string          `json:"method"`
		Header []kv            `json:"header"`
		URL    json.RawMessage `json:"url"`
		Body   *struct {
			Mode       string `json:"mode"`
			Raw        string `json:"raw"`
			URLEncoded []kv   `json:"urlencoded"`
			FormData   []kv   `json:"formdata"`
			Options    struct {
				Raw struct {
					Language string `json:"language"`
				} `json:"raw"`
			} `json:"options"`
		} `json:"body"`
		Auth *auth `json:"auth"`
	} `json:"request"`
}

type auth struct {
	Type   string `json:"type"`
	Bearer []kv   `json:"bearer"`
	Basic  []kv   `json:"basic"`
	APIKey []kv   `json:"apikey"`
}

type collection struct {
	Info struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"info"`
	Variable []kv   `json:"variable"`
	Items    []item `json:"item"`
	Auth     *auth  `json:"auth"`
}

// Parse reads a Postman collection.
func Parse(data []byte) (Result, error) {
	var c collection
	if err := json.Unmarshal(data, &c); err != nil {
		return Result{}, fmt.Errorf("not JSON: %w", err)
	}
	if !strings.Contains(c.Info.Schema, "getpostman.com/json/collection") || len(c.Items) == 0 && c.Info.Name == "" {
		return Result{}, errors.New("not a Postman collection (v2.0 or v2.1)")
	}
	name := c.Info.Name
	if name == "" {
		name = "Postman collection"
	}
	root := &model.Folder{Name: name}
	fill(root, c.Items, c.Auth)
	if len(root.Requests)+len(root.Folders) == 0 {
		return Result{}, errors.New("the collection has no requests")
	}
	res := Result{Folder: root}
	for _, v := range c.Variable {
		if v.Key != "" {
			res.Vars = append(res.Vars, model.KV{Name: v.Key, Value: v.Value, Enabled: !v.Disabled})
		}
	}
	return res, nil
}

func fill(dir *model.Folder, items []item, inherited *auth) {
	for _, it := range items {
		if it.Request == nil {
			sub := dir.AddFolder(it.Name)
			fill(sub, it.Items, inherited)
			continue
		}
		r := request(it, inherited)
		r.Name = dir.UniqueName(r.Name)
		dir.AddRequest(r)
	}
}

func request(it item, inherited *auth) model.Request {
	pr := it.Request
	req := model.Request{Name: it.Name, Protocol: "rest", Method: strings.ToUpper(pr.Method), BodyType: model.BodyNone}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Name == "" {
		req.Name = req.Method + " request"
	}
	req.URL, req.Params = parseURL(pr.URL)
	for _, h := range pr.Header {
		req.Headers = append(req.Headers, model.KV{Name: h.Key, Value: h.Value, Enabled: !h.Disabled})
	}
	if b := pr.Body; b != nil {
		switch b.Mode {
		case "raw":
			req.Body = b.Raw
			req.BodyType = model.BodyText
			if b.Options.Raw.Language == "json" || looksJSON(b.Raw) {
				req.BodyType = model.BodyJSON
			}
		case "urlencoded":
			req.BodyType, req.Form = model.BodyForm, rows(b.URLEncoded, false)
		case "formdata":
			req.BodyType, req.Form = model.BodyMultipart, rows(b.FormData, true)
		case "graphql":
			req.BodyType = model.BodyGraphQL
		}
	}
	a := pr.Auth
	if a == nil {
		a = inherited
	}
	req.Auth = convertAuth(a)
	return req
}

func looksJSON(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")) && json.Valid([]byte(s))
}

// parseURL reads a Postman url, a string or an object with raw and query.
func parseURL(raw json.RawMessage) (string, []model.KV) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, model.ParamsFromURL(s)
	}
	var o struct {
		Raw   string `json:"raw"`
		Query []kv   `json:"query"`
	}
	if json.Unmarshal(raw, &o) != nil {
		return "", nil
	}
	base := o.Raw
	var params []model.KV
	for _, q := range o.Query {
		params = append(params, model.KV{Name: q.Key, Value: q.Value, Enabled: !q.Disabled})
	}
	if len(params) > 0 {
		base = model.WithParams(base, params)
	} else {
		params = model.ParamsFromURL(base)
	}
	return base, params
}

func rows(in []kv, files bool) []model.KV {
	var out []model.KV
	for _, f := range in {
		v := f.Value
		if files && f.Type == "file" {
			if src, ok := f.Src.(string); ok && src != "" {
				v = "@" + src
			}
		}
		out = append(out, model.KV{Name: f.Key, Value: v, Enabled: !f.Disabled})
	}
	return out
}

func get(list []kv, key string) string {
	for _, e := range list {
		if e.Key == key {
			return e.Value
		}
	}
	return ""
}

func convertAuth(a *auth) model.Auth {
	if a == nil {
		return model.Auth{}
	}
	switch a.Type {
	case "bearer":
		return model.Auth{Type: model.AuthBearer, Token: get(a.Bearer, "token")}
	case "basic":
		return model.Auth{Type: model.AuthBasic, User: get(a.Basic, "username"), Password: get(a.Basic, "password")}
	case "apikey":
		in := model.KeyInHeader
		if get(a.APIKey, "in") == "query" {
			in = model.KeyInQuery
		}
		return model.Auth{Type: model.AuthAPIKey, KeyName: get(a.APIKey, "key"), KeyValue: get(a.APIKey, "value"), KeyIn: in}
	}
	return model.Auth{}
}

// ---- export ------------------------------------------------------------

type outKV struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
	Type     string `json:"type,omitempty"`
	Src      string `json:"src,omitempty"`
}

type outItem struct {
	Name    string     `json:"name"`
	Items   []*outItem `json:"item,omitempty"`
	Request *outReq    `json:"request,omitempty"`
}

type outReq struct {
	Method string   `json:"method"`
	Header []outKV  `json:"header"`
	URL    outURL   `json:"url"`
	Body   *outBody `json:"body,omitempty"`
	Auth   *outAuth `json:"auth,omitempty"`
}

type outURL struct {
	Raw   string  `json:"raw"`
	Query []outKV `json:"query,omitempty"`
}

type outBody struct {
	Mode       string  `json:"mode"`
	Raw        string  `json:"raw,omitempty"`
	URLEncoded []outKV `json:"urlencoded,omitempty"`
	FormData   []outKV `json:"formdata,omitempty"`
	Options    *struct {
		Raw struct {
			Language string `json:"language"`
		} `json:"raw"`
	} `json:"options,omitempty"`
}

type outAuth struct {
	Type   string  `json:"type"`
	Bearer []outKV `json:"bearer,omitempty"`
	Basic  []outKV `json:"basic,omitempty"`
	APIKey []outKV `json:"apikey,omitempty"`
}

// Export writes the folder and the collection variables as a Postman v2.1
// collection. Request kinds Postman has no place for (WebSocket, MCP, SSE)
// are left out; the caller can count them with Skipped.
func Export(root *model.Folder, vars []model.KV) ([]byte, error) {
	out := struct {
		Info struct {
			Name   string `json:"name"`
			Schema string `json:"schema"`
		} `json:"info"`
		Items    []*outItem `json:"item"`
		Variable []outKV    `json:"variable,omitempty"`
	}{}
	out.Info.Name = root.Name
	out.Info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	out.Items = exportFolder(root)
	for _, v := range vars {
		out.Variable = append(out.Variable, outKV{Key: v.Name, Value: v.Value, Disabled: !v.Enabled})
	}
	return json.MarshalIndent(out, "", "  ")
}

// Skipped counts the requests of root that Export leaves out.
func Skipped(root *model.Folder) int {
	n := 0
	for _, r := range root.Requests {
		if r.Protocol != "rest" {
			n++
		}
	}
	for _, f := range root.Folders {
		n += Skipped(f)
	}
	return n
}

func exportFolder(f *model.Folder) []*outItem {
	var items []*outItem
	for _, sub := range f.Folders {
		items = append(items, &outItem{Name: sub.Name, Items: exportFolder(sub)})
	}
	for _, r := range f.Requests {
		if r.Protocol != "rest" {
			continue
		}
		items = append(items, &outItem{Name: r.Name, Request: exportRequest(r)})
	}
	return items
}

func outRows(rows []model.KV, files bool) []outKV {
	var out []outKV
	for _, r := range rows {
		e := outKV{Key: r.Name, Value: r.Value, Disabled: !r.Enabled}
		if files {
			e.Type = "text"
			if path, ok := strings.CutPrefix(r.Value, "@"); ok && path != "" && !strings.HasPrefix(path, "@") {
				e.Type, e.Value, e.Src = "file", "", path
			}
		}
		out = append(out, e)
	}
	return out
}

func exportRequest(r *model.Request) *outReq {
	o := &outReq{Method: r.Method, Header: outRows(r.Headers, false), URL: outURL{Raw: r.URL}}
	if o.Header == nil {
		o.Header = []outKV{}
	}
	for _, p := range r.Params {
		if !model.IsPathParam(p.Name) {
			o.URL.Query = append(o.URL.Query, outKV{Key: p.Name, Value: p.Value, Disabled: !p.Enabled})
		}
	}
	switch r.BodyType {
	case model.BodyJSON, model.BodyText:
		b := &outBody{Mode: "raw", Raw: r.Body}
		if r.BodyType == model.BodyJSON {
			b.Options = &struct {
				Raw struct {
					Language string `json:"language"`
				} `json:"raw"`
			}{}
			b.Options.Raw.Language = "json"
		}
		o.Body = b
	case model.BodyForm:
		o.Body = &outBody{Mode: "urlencoded", URLEncoded: outRows(r.Form, false)}
	case model.BodyMultipart:
		o.Body = &outBody{Mode: "formdata", FormData: outRows(r.Form, true)}
	}
	switch r.Auth.Type {
	case model.AuthBearer:
		o.Auth = &outAuth{Type: "bearer", Bearer: []outKV{{Key: "token", Value: r.Auth.Token}}}
	case model.AuthBasic:
		o.Auth = &outAuth{Type: "basic", Basic: []outKV{{Key: "username", Value: r.Auth.User}, {Key: "password", Value: r.Auth.Password}}}
	case model.AuthAPIKey:
		in := r.Auth.KeyIn
		if in == "" {
			in = model.KeyInHeader
		}
		o.Auth = &outAuth{Type: "apikey", APIKey: []outKV{{Key: "key", Value: r.Auth.KeyName}, {Key: "value", Value: r.Auth.KeyValue}, {Key: "in", Value: in}}}
	}
	return o
}
