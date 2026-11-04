// Package model holds the plain data the workshop edits and sends:
// collections of requests and the responses they produce.
package model

import "time"

// KV is one row of a params, headers or vars table.
type KV struct {
	Name        string
	Value       string
	Description string
	Enabled     bool
	Secret      bool // a variable whose value is kept out of the collection files
}

// Request is one saved call. Protocol selects the proto implementation
// ("rest", "websocket", "mcp"); Method is only meaningful for REST.
type Request struct {
	Name     string
	Protocol string
	Method   string
	URL      string
	Params   []KV
	Headers  []KV
	Body     string // raw text of the json, text and legacy bodies
	BodyType string // BodyNone, BodyJSON, ...; "" sends Body as is, with no Content-Type
	Form     []KV   // fields of the form and multipart bodies

	Auth Auth // credentials added when the request is sent; never shown in Headers

	TimeoutSec  int  // seconds before the request is abandoned; 0 uses the default
	NoRedirects bool // stop at the first redirect instead of following it
	InsecureTLS bool // accept any server certificate (self-signed test servers)
}

// Body types a request can send.
const (
	BodyNone      = "none"
	BodyJSON      = "json"
	BodyText      = "text"
	BodyForm      = "form"
	BodyMultipart = "multipart"
)

// Folder groups requests and nested folders in a collection tree.
// Children are pointers, so a request or folder keeps its identity when
// siblings are added, removed or moved (open tabs point at requests).
type Folder struct {
	Name     string
	Vars     []KV // variables of the requests below, over the environment's and the collection's
	Folders  []*Folder
	Requests []*Request
}

// Collection is the root of a tree of folders plus the variables
// ({{name}} in URLs) shared by its requests.
type Collection struct {
	Folder
	Vars []KV
	Envs []Environment // named sets of variables that override Vars when active
}

// Environment is a named set of variables, such as "Dev" or "Prod".
type Environment struct {
	Name string
	Vars []KV
}

// Response is the outcome of sending a Request. Err is set instead of
// Status when no response was received.
type Response struct {
	Timing     Timing // where the time went, from net/http/httptrace
	Truncated  bool   // the body was cut at the size limit; Size is what was kept
	Status     int
	StatusText string
	Duration   time.Duration
	Size       int
	Headers    []KV
	Body       []byte
	Err        string
}

// Timing splits a request into its phases. A phase the connection did not
// go through (a reused connection has no DNS, connect or TLS) is zero.
type Timing struct {
	DNS, Connect, TLS time.Duration
	FirstByte         time.Duration // from sending the request to the first byte of the answer
	Transfer          time.Duration // reading the body
}

// Clone returns a deep copy of r: editing the copy's rows never changes
// r. Empty slices become nil, so a copy compares equal to a request
// whose rows were all deleted again.
func (r Request) Clone() Request {
	r.Params, r.Headers, r.Form = cloneKV(r.Params), cloneKV(r.Headers), cloneKV(r.Form)
	return r
}

func cloneKV(rows []KV) []KV {
	if len(rows) == 0 {
		return nil
	}
	return append([]KV(nil), rows...)
}

// Auth types. The zero value, "", is no authentication.
const (
	AuthNone   = "none"
	AuthBearer = "bearer"
	AuthBasic  = "basic"
	AuthAPIKey = "apikey"
)

// Where an API key goes.
const (
	KeyInHeader = "header"
	KeyInQuery  = "query"
)

// Auth is a request's credentials. Every type has its own fields, so
// switching type keeps what was typed for the others.
type Auth struct {
	Type     string
	Token    string // bearer
	User     string // basic
	Password string // basic
	KeyName  string // api key
	KeyValue string // api key
	KeyIn    string // KeyInHeader (default) or KeyInQuery
}

// Expanded returns a copy of r with {{name}} replaced from vars in the
// URL, header names and values, bodies, form fields and credentials.
func (r Request) Expanded(vars []KV) Request {
	r = r.Clone()
	for i := range r.Params { // values may use {{variables}}, and path parameters fill the URL
		if IsPathParam(r.Params[i].Name) {
			r.Params[i].Value = Expand(r.Params[i].Value, vars)
		}
	}
	r.URL, r.Body = Expand(ApplyPathParams(r.URL, r.Params), vars), Expand(r.Body, vars)
	for _, rows := range [][]KV{r.Headers, r.Form} {
		for i := range rows {
			rows[i].Name, rows[i].Value = Expand(rows[i].Name, vars), Expand(rows[i].Value, vars)
		}
	}
	a := &r.Auth
	for _, field := range []*string{&a.Token, &a.User, &a.Password, &a.KeyName, &a.KeyValue} {
		*field = Expand(*field, vars)
	}
	return r
}
