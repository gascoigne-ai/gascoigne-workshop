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
type Folder struct {
	Name     string
	Folders  []Folder
	Requests []Request
}

// Collection is the root of a tree of folders plus the variables
// ({{name}} in URLs) shared by its requests.
type Collection struct {
	Folder
	Vars []KV
}

// Response is the outcome of sending a Request. Err is set instead of
// Status when no response was received.
type Response struct {
	Status     int
	StatusText string
	Duration   time.Duration
	Size       int
	Headers    []KV
	Body       []byte
	Err        string
}
