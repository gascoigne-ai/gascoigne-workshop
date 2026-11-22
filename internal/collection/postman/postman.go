// Package postman reads a Postman collection (v2.0 and v2.1, JSON) into a
// folder of requests: folders stay folders, headers, query, body modes and
// the common auth types carry over, collection variables are returned.
package postman

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// Result is what an import produces.
type Result struct {
	Folder  *request.Folder
	Vars    []request.KV
	Skipped []string // what could not be carried over, as "request: what"
}

// text is a JSON value read as text: a string as it is, a number or
// boolean as written, null as empty. Postman writes variable and header
// values of any of these types.
type text string

func (t *text) UnmarshalJSON(data []byte) error {
	var str string
	if json.Unmarshal(data, &str) == nil {
		*t = text(str)
		return nil
	}
	if string(data) == "null" {
		*t = ""
		return nil
	}
	*t = text(data)
	return nil
}

// kv is a header, variable, query or form row, read and written alike.
// Empty fields are left out of a written file.
type kv struct {
	Key      string `json:"key"`
	Value    text   `json:"value,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
	Type     string `json:"type,omitempty"`
	Src      any    `json:"src,omitempty"`
}

type item struct {
	Name    string       `json:"name"`
	Items   []item       `json:"item"`
	Request *requestSpec `json:"request"`
}

// requestSpec is a request; v2.0 also allows a bare URL string.
type requestSpec struct {
	Method string          `json:"method"`
	Header []kv            `json:"header"`
	URL    json.RawMessage `json:"url"`
	Body   *body           `json:"body"`
	Auth   *auth           `json:"auth"`
}

func (r *requestSpec) UnmarshalJSON(data []byte) error {
	var url string
	if json.Unmarshal(data, &url) == nil {
		quoted, _ := json.Marshal(url)
		r.URL = quoted
		return nil
	}
	type plain requestSpec // without the method, so it does not recurse
	return json.Unmarshal(data, (*plain)(r))
}

// body is a request body as Postman writes it: the mode names which of the
// other fields is in use.
type body struct {
	Mode       string       `json:"mode"`
	Raw        string       `json:"raw,omitempty"`
	URLEncoded []kv         `json:"urlencoded,omitempty"`
	FormData   []kv         `json:"formdata,omitempty"`
	Options    *bodyOptions `json:"options,omitempty"`
}

type bodyOptions struct {
	Raw struct {
		Language string `json:"language"`
	} `json:"raw"`
}

// bodyMode ties a Postman body mode to the body types it carries; read and
// write are nil where a mode is not carried that way. lost marks a mode
// the import reports as not (fully) imported.
type bodyMode struct {
	mode  string
	types []request.BodyType
	lost  bool
	read  func(from *body, to *request.Request)
	write func(from *request.Request) *body
}

var bodyModes = []bodyMode{
	{mode: "raw", types: []request.BodyType{request.BodyJSON, request.BodyText}, read: readRawBody, write: writeRawBody},
	{mode: "urlencoded", types: []request.BodyType{request.BodyForm},
		read: func(from *body, to *request.Request) {
			to.BodyType, to.Form = request.BodyForm, rows(from.URLEncoded, false)
		},
		write: func(from *request.Request) *body {
			return &body{Mode: "urlencoded", URLEncoded: outRows(from.Form, false)}
		}},
	{mode: "formdata", types: []request.BodyType{request.BodyMultipart},
		read: func(from *body, to *request.Request) {
			to.BodyType, to.Form = request.BodyMultipart, rows(from.FormData, true)
		},
		write: func(from *request.Request) *body {
			return &body{Mode: "formdata", FormData: outRows(from.Form, true)}
		}},
	{mode: "graphql", lost: true, read: func(_ *body, to *request.Request) { to.BodyType = request.BodyGraphQL }},
	{mode: "file", lost: true},
}

func readRawBody(from *body, to *request.Request) {
	to.Body = from.Raw
	to.BodyType = request.BodyText
	if from.Options != nil && from.Options.Raw.Language == "json" || isValidJSONObjectOrArray(from.Raw) {
		to.BodyType = request.BodyJSON
	}
}

func writeRawBody(from *request.Request) *body {
	out := &body{Mode: "raw", Raw: from.Body}
	if from.BodyType == request.BodyJSON {
		out.Options = &bodyOptions{}
		out.Options.Raw.Language = "json"
	}
	return out
}

func bodyModeNamed(mode string) *bodyMode {
	for i := range bodyModes {
		if bodyModes[i].mode == mode {
			return &bodyModes[i]
		}
	}
	return nil
}

func bodyModeFor(bodyType request.BodyType) *bodyMode {
	for i := range bodyModes {
		for _, candidate := range bodyModes[i].types {
			if candidate == bodyType {
				return &bodyModes[i]
			}
		}
	}
	return nil
}

// auth is an auth block: the type names which of the lists is in use.
type auth struct {
	Type   string `json:"type"`
	Bearer []kv   `json:"bearer,omitempty"`
	Basic  []kv   `json:"basic,omitempty"`
	APIKey []kv   `json:"apikey,omitempty"`
}

// authKind ties a Postman auth type to a model one.
type authKind struct {
	name  string
	model request.AuthType
	read  func(from *auth) request.Auth
	write func(from request.Auth) *auth
}

var authKinds = []authKind{
	{name: "bearer", model: request.AuthBearer,
		read: func(from *auth) request.Auth {
			return request.Auth{Type: request.AuthBearer, Token: get(from.Bearer, "token")}
		},
		write: func(from request.Auth) *auth {
			return &auth{Type: "bearer", Bearer: []kv{{Key: "token", Value: text(shareable(from.Token))}}}
		}},
	{name: "basic", model: request.AuthBasic,
		read: func(from *auth) request.Auth {
			return request.Auth{Type: request.AuthBasic, User: get(from.Basic, "username"), Password: get(from.Basic, "password")}
		},
		write: func(from request.Auth) *auth {
			return &auth{Type: "basic", Basic: []kv{{Key: "username", Value: text(from.User)}, {Key: "password", Value: text(shareable(from.Password))}}}
		}},
	{name: "apikey", model: request.AuthAPIKey,
		read: func(from *auth) request.Auth {
			in := request.KeyInHeader
			if get(from.APIKey, "in") == string(request.KeyInQuery) { // lenient: anything else is a header
				in = request.KeyInQuery
			}
			return request.Auth{Type: request.AuthAPIKey, KeyName: get(from.APIKey, "key"), KeyValue: get(from.APIKey, "value"), KeyIn: in}
		},
		write: func(from request.Auth) *auth {
			in := from.KeyIn
			if in == "" {
				in = request.KeyInHeader
			}
			return &auth{Type: "apikey", APIKey: []kv{{Key: "key", Value: text(from.KeyName)}, {Key: "value", Value: text(shareable(from.KeyValue))}, {Key: "in", Value: text(string(in))}}}
		}},
}

func authKindNamed(name string) *authKind {
	for i := range authKinds {
		if authKinds[i].name == name {
			return &authKinds[i]
		}
	}
	return nil
}

func authKindFor(authType request.AuthType) *authKind {
	for i := range authKinds {
		if authKinds[i].model == authType {
			return &authKinds[i]
		}
	}
	return nil
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
	root := &request.Folder{Name: name}
	var skipped []string
	fill(root, c.Items, c.Auth, &skipped)
	if len(root.Requests)+len(root.Folders) == 0 {
		return Result{}, errors.New("the collection has no requests")
	}
	res := Result{Folder: root, Skipped: skipped}
	for _, v := range c.Variable {
		if v.Key != "" {
			res.Vars = append(res.Vars, request.KV{Name: v.Key, Value: string(v.Value), Enabled: !v.Disabled})
		}
	}
	return res, nil
}

func fill(dir *request.Folder, items []item, inherited *auth, skipped *[]string) {
	for _, it := range items {
		if it.Request == nil {
			sub := dir.AddFolder(it.Name)
			fill(sub, it.Items, inherited, skipped)
			continue
		}
		effective := it.Request.Auth
		if effective == nil {
			effective = inherited
		}
		for _, what := range notCarriedOver(it, effective) {
			*skipped = append(*skipped, it.Name+": "+what)
		}
		r, urlErr := readRequest(it, effective)
		if urlErr != nil {
			*skipped = append(*skipped, it.Name+": url is not imported ("+urlErr.Error()+")")
		}
		r.Name = dir.UniqueName(r.Name)
		dir.AddRequest(r)
	}
}

// notCarriedOver lists what of a request has no equivalent here; effective
// is the auth block that applies to it, its own or the inherited one.
func notCarriedOver(it item, effective *auth) []string {
	var out []string
	switch {
	case effective == nil, effective.Type == "", effective.Type == "noauth", authKindNamed(effective.Type) != nil:
	default:
		out = append(out, "auth type "+effective.Type+" is not imported")
	}
	if b := it.Request.Body; b != nil {
		if mode := bodyModeNamed(b.Mode); mode != nil && mode.lost {
			out = append(out, "body mode "+b.Mode+" is not imported")
		}
	}
	return out
}

// readRequest converts one item. A url that cannot be read leaves the request
// with an empty one and comes back as the error.
func readRequest(it item, effective *auth) (request.Request, error) {
	pr := it.Request
	req := request.Request{Name: it.Name, Protocol: request.ProtocolREST, Method: strings.ToUpper(pr.Method), BodyType: request.BodyNone}
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Name == "" {
		req.Name = req.Method + " request"
	}
	var urlErr error
	req.URL, req.Params, urlErr = parseURL(pr.URL)
	for _, h := range pr.Header {
		req.Headers = append(req.Headers, request.KV{Name: h.Key, Value: string(h.Value), Enabled: !h.Disabled})
	}
	if pr.Body != nil {
		if mode := bodyModeNamed(pr.Body.Mode); mode != nil && mode.read != nil {
			mode.read(pr.Body, &req)
		}
	}
	req.Auth = convertAuth(effective)
	return req, urlErr
}

// isValidJSONObjectOrArray is true for text that is a complete JSON object
// or list; a body that merely starts with a brace is not.
func isValidJSONObjectOrArray(text string) bool {
	text = strings.TrimSpace(text)
	return (strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[")) && json.Valid([]byte(text))
}

// parseURL reads a Postman url, a string or an object with raw and query.
// A missing url is empty; one of another shape is an error.
func parseURL(raw json.RawMessage) (string, []request.KV, error) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, request.ParamsFromURL(s), nil
	}
	var o struct {
		Raw   string `json:"raw"`
		Query []kv   `json:"query"`
	}
	if err := json.Unmarshal(raw, &o); err != nil {
		return "", nil, err
	}
	base := o.Raw
	var params []request.KV
	for _, q := range o.Query {
		params = append(params, request.KV{Name: q.Key, Value: string(q.Value), Enabled: !q.Disabled})
	}
	if len(params) > 0 {
		base = request.WithParams(base, params)
	} else {
		params = request.ParamsFromURL(base)
	}
	return base, params, nil
}

func rows(in []kv, files bool) []request.KV {
	var out []request.KV
	for _, f := range in {
		v := string(f.Value)
		if files && f.Type == "file" {
			if src, ok := f.Src.(string); ok && src != "" {
				v = "@" + src
			}
		}
		out = append(out, request.KV{Name: f.Key, Value: v, Enabled: !f.Disabled})
	}
	return out
}

func get(list []kv, key string) string {
	for _, e := range list {
		if e.Key == key {
			return string(e.Value)
		}
	}
	return ""
}

func convertAuth(block *auth) request.Auth {
	if block == nil {
		return request.Auth{}
	}
	if kind := authKindNamed(block.Type); kind != nil {
		return kind.read(block)
	}
	return request.Auth{}
}

// ---- export ------------------------------------------------------------

type outItem struct {
	Name    string     `json:"name"`
	Items   []*outItem `json:"item,omitempty"`
	Request *outReq    `json:"request,omitempty"`
}

type outReq struct {
	Method string `json:"method"`
	Header []kv   `json:"header"`
	URL    outURL `json:"url"`
	Body   *body  `json:"body,omitempty"`
	Auth   *auth  `json:"auth,omitempty"`
}

type outURL struct {
	Raw   string `json:"raw"`
	Query []kv   `json:"query,omitempty"`
}

// Export writes the folder and the collection variables as a Postman v2.1
// collection. Request kinds Postman has no place for (WebSocket, MCP, SSE)
// are left out; the caller can count them with Skipped.
func Export(root *request.Folder, vars []request.KV) ([]byte, error) {
	out := struct {
		Info struct {
			Name   string `json:"name"`
			Schema string `json:"schema"`
		} `json:"info"`
		Items    []*outItem `json:"item"`
		Variable []kv       `json:"variable,omitempty"`
	}{}
	out.Info.Name = root.Name
	out.Info.Schema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	out.Items = exportFolder(root)
	for _, v := range vars {
		value := v.Value
		if v.Secret {
			value = shareable(value)
		}
		out.Variable = append(out.Variable, kv{Key: v.Name, Value: text(value), Disabled: !v.Enabled})
	}
	return json.MarshalIndent(out, "", "  ")
}

// shareable is what of a credential goes into an exported file, which is
// made to be shared: a {{variable}} reference stays, a literal value does
// not.
func shareable(value string) string {
	name, ok := strings.CutPrefix(value, "{{")
	if name, ok = strings.CutSuffix(name, "}}"); ok && name != "" && !strings.ContainsAny(name, "{}") {
		return value
	}
	return ""
}

// Skipped counts the requests of root that Export leaves out.
func Skipped(root *request.Folder) int {
	n := 0
	for _, r := range root.Requests {
		if !exportable(r) {
			n++
		}
	}
	for _, f := range root.Folders {
		n += Skipped(f)
	}
	return n
}

// exportable reports whether Postman can hold r: an HTTP request.
func exportable(call *request.Request) bool {
	return call.Protocol.Info().Method == request.MethodHTTP
}

func exportFolder(folder *request.Folder) []*outItem {
	var items []*outItem
	for _, sub := range folder.Folders {
		items = append(items, &outItem{Name: sub.Name, Items: exportFolder(sub)})
	}
	for _, r := range folder.Requests {
		if !exportable(r) {
			continue
		}
		items = append(items, &outItem{Name: r.Name, Request: exportRequest(r)})
	}
	return items
}

func outRows(rows []request.KV, files bool) []kv {
	var out []kv
	for _, r := range rows {
		e := kv{Key: r.Name, Value: text(r.Value), Disabled: !r.Enabled}
		if files {
			e.Type = "text"
			if path, ok := request.FormFile(r.Value); ok {
				e.Type, e.Value, e.Src = "file", "", path
			}
		}
		out = append(out, e)
	}
	return out
}

func exportRequest(call *request.Request) *outReq {
	o := &outReq{Method: call.Method, Header: outRows(call.Headers, false), URL: outURL{Raw: call.URL}}
	if o.Header == nil {
		o.Header = []kv{}
	}
	for _, p := range call.Params {
		if !request.IsPathParam(p.Name) {
			o.URL.Query = append(o.URL.Query, kv{Key: p.Name, Value: text(p.Value), Disabled: !p.Enabled})
		}
	}
	if mode := bodyModeFor(call.BodyType); mode != nil {
		o.Body = mode.write(call)
	}
	if kind := authKindFor(call.Auth.Type); kind != nil {
		o.Auth = kind.write(call.Auth)
	}
	return o
}
