// Package postman reads a Postman collection (v2.0 and v2.1, JSON) into a
// folder of requests: folders stay folders, headers, query, body modes and
// the common auth types carry over, collection variables are returned.
package postman

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
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
