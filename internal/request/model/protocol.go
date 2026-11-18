package model

import "fmt"

// Protocol names the way a Request is sent. Its text is what collection
// files, client configs and the sent log store; ParseProtocol turns that
// text into a Protocol once, at the edge.
type Protocol string

// The protocols the workshop speaks. What differs between them is a row of
// Protocols, never a comparison with one of these.
const (
	ProtocolREST      Protocol = "rest"
	ProtocolWebSocket Protocol = "websocket"
	ProtocolMCP       Protocol = "mcp"
	ProtocolSSE       Protocol = "sse"
	ProtocolGRPC      Protocol = "grpc"
)

// Connection says whether a protocol keeps a connection open.
type Connection uint8

// Connections.
const (
	OneShot  Connection = iota // each send is a request and its response
	Always                     // a tab connects, then sends and receives over the connection
	OnStream                   // one-shot, but connected while the tab streams (gRPC's Stream chip)
)

// MethodKind says what a request's Method holds for a protocol.
type MethodKind uint8

// Method kinds.
const (
	MethodNone MethodKind = iota // the protocol has no method
	MethodHTTP                   // an HTTP method: GET, POST, ...
	MethodRPC                    // a service method: package.Service/Method
)

// ProtocolInfo is what differs between protocols, as data. The zero value,
// the Info of no protocol, is one-shot with no method.
type ProtocolInfo struct {
	Name         Protocol
	Label        string // short name on the protocol chip and the tree badge
	Connection   Connection
	Method       MethodKind
	RunsCommand  bool   // its URL may be a command run on this machine (MCP's stdio:)
	ResumeHeader string // header that resumes a stream after a reconnect; "" when it cannot
}

// Protocols is every protocol, in the order the protocol chip cycles.
var Protocols = []ProtocolInfo{
	{Name: ProtocolREST, Label: "REST", Connection: OneShot, Method: MethodHTTP},
	{Name: ProtocolWebSocket, Label: "WS", Connection: Always},
	{Name: ProtocolMCP, Label: "MCP", Connection: Always, RunsCommand: true},
	{Name: ProtocolSSE, Label: "SSE", Connection: Always, ResumeHeader: "Last-Event-ID"},
	{Name: ProtocolGRPC, Label: "gRPC", Connection: OnStream, Method: MethodRPC},
}

// Info is the row of Protocols for p; the zero ProtocolInfo for an
// unknown protocol.
func (p Protocol) Info() ProtocolInfo {
	info, _ := lookupProtocol(string(p))
	return info
}

// lookupProtocol finds the row whose name is text.
func lookupProtocol(text string) (ProtocolInfo, bool) {
	for _, row := range Protocols {
		if string(row.Name) == text {
			return row, true
		}
	}
	return ProtocolInfo{}, false
}

// ParseProtocol returns the protocol named by text. An unknown or empty
// name is an error.
func ParseProtocol(text string) (Protocol, error) {
	row, ok := lookupProtocol(text)
	if !ok {
		return "", fmt.Errorf("unknown protocol %q", text)
	}
	return row.Name, nil
}

// MarshalText implements encoding.TextMarshaler: the protocol's name.
func (p Protocol) MarshalText() ([]byte, error) { return []byte(p), nil }

// UnmarshalText implements encoding.TextUnmarshaler with ParseProtocol, so
// a stored record with an unknown protocol fails to decode.
func (p *Protocol) UnmarshalText(text []byte) error {
	parsed, err := ParseProtocol(string(text))
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

// BodyType says how a request's body is encoded.
type BodyType string

// Body types a request can send.
const (
	BodyRaw       BodyType = ""     // legacy: Body is sent as is, with no Content-Type
	BodyNone      BodyType = "none" // no body
	BodyJSON      BodyType = "json"
	BodyText      BodyType = "text"
	BodyForm      BodyType = "form"
	BodyMultipart BodyType = "multipart"
	BodyGraphQL   BodyType = "graphql" // Body is the query, GraphQLVars the variables as JSON
	BodyFile      BodyType = "file"    // Body holds the path of the file to send as it is
)

// ParseBodyType returns the body type named by text; "" is BodyRaw.
func ParseBodyType(text string) (BodyType, error) {
	for _, known := range []BodyType{BodyRaw, BodyNone, BodyJSON, BodyText, BodyForm, BodyMultipart, BodyGraphQL, BodyFile} {
		if text == string(known) {
			return known, nil
		}
	}
	return "", fmt.Errorf("unknown body type %q", text)
}

// AuthType says which credentials a request carries.
type AuthType string

// Auth types. The zero value is no authentication.
const (
	AuthNone   AuthType = ""
	AuthBearer AuthType = "bearer"
	AuthBasic  AuthType = "basic"
	AuthAPIKey AuthType = "apikey"
)

// authNoneName is how files written before AuthNone was "" store no
// authentication.
const authNoneName = "none"

// ParseAuthType returns the auth type named by text; "" and "none" are
// AuthNone.
func ParseAuthType(text string) (AuthType, error) {
	if text == authNoneName {
		return AuthNone, nil
	}
	for _, known := range []AuthType{AuthNone, AuthBearer, AuthBasic, AuthAPIKey} {
		if text == string(known) {
			return known, nil
		}
	}
	return "", fmt.Errorf("unknown auth type %q", text)
}
