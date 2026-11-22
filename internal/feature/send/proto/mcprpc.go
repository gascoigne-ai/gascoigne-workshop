package proto

import "encoding/json"

// JSON-RPC error codes this client answers with.
const (
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
	rpcRejected       = -1     // the user declined a sampling request
	rpcBusy           = -32000 // the client has no free worker for the request
)

const rpcVersion = "2.0"

// The fields of the structs below are in alphabetical order, which is the
// key order the messages have always had on the wire (they were maps).

// rpcRequest is a request, or with a nil ID a notification, the client sends.
type rpcRequest struct {
	ID      *int64 `json:"id,omitempty"`
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// rpcReply is the client's answer to a request of the server: a Result or
// an Error.
type rpcReply struct {
	Error   *rpcError       `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  any             `json:"result,omitempty"`
}

// rpcError is the error member of a reply, sent and received.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// resultReply answers the server's request id with a result.
func resultReply(id json.RawMessage, result any) rpcReply {
	return rpcReply{ID: id, JSONRPC: rpcVersion, Result: result}
}

// errorReply answers the server's request id with an error.
func errorReply(id json.RawMessage, code int, message string) rpcReply {
	return rpcReply{ID: id, JSONRPC: rpcVersion, Error: &rpcError{Code: code, Message: message}}
}

// methodNotFound refuses a method this client does not implement.
func methodNotFound(id json.RawMessage, method string) rpcReply {
	return errorReply(id, rpcMethodNotFound, "method not found: "+method)
}

// initializeParams is the params of the initialize request.
type initializeParams struct {
	Capabilities    clientCapabilities `json:"capabilities"`
	ClientInfo      clientInfo         `json:"clientInfo"`
	ProtocolVersion string             `json:"protocolVersion"`
}

// clientCapabilities lists what the client offers; sampling only when it
// has a Sampler.
type clientCapabilities struct {
	Sampling *struct{} `json:"sampling,omitempty"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// rootsResult is the answer to roots/list: this client offers no roots.
type rootsResult struct {
	Roots []struct{} `json:"roots"`
}

// samplingResult is the answer to a sampling request that the user accepted.
type samplingResult struct {
	Content    samplingContent `json:"content"`
	Model      string          `json:"model"`
	Role       string          `json:"role"`
	StopReason string          `json:"stopReason"`
}

type samplingContent struct {
	Text string `json:"text"`
	Type string `json:"type"`
}
