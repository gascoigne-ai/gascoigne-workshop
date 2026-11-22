//go:build !release

package collection

import (
	"path/filepath"
	"runtime"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// ExampleServerFolder is the name of the folder of requests that reach
// cmd/example-server.
const ExampleServerFolder = "Example Server"

// Sample is the collection a debug build shows until a workspace is
// opened: one request per endpoint of cmd/example-server on its default
// ports, and nothing that leaves the machine.
func Sample() request.Collection {
	return request.Collection{Folder: *exampleServerFolder()}
}

// The ports are the defaults of exampleserver.DefaultConfig; a test there
// keeps the two in step.
func exampleServerFolder() *request.Folder {
	return &request.Folder{Name: ExampleServerFolder, Requests: []*request.Request{
		{Name: "Index", Protocol: request.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/"},
		{Name: "Echo", Protocol: request.ProtocolREST, Method: "POST", URL: "http://127.0.0.1:8081/echo",
			BodyType: request.BodyJSON, Body: "{\n  \"hello\": \"world\"\n}"},
		{Name: "Status 418", Protocol: request.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/status/418"},
		{Name: "Delay 500ms", Protocol: request.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/delay/500"},
		{Name: "Auth", Protocol: request.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/auth",
			Auth: request.Auth{Type: request.AuthBearer, Token: "example-token"}},
		{Name: "Auth without credentials", Protocol: request.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/auth"},
		{Name: "WebSocket Echo", Protocol: request.ProtocolWebSocket, URL: "ws://127.0.0.1:8082"},
		{Name: "SSE Ticks", Protocol: request.ProtocolSSE, URL: "http://127.0.0.1:8083/events?count=5"},
		{Name: "MCP Tools", Protocol: request.ProtocolMCP, URL: "http://127.0.0.1:8084/mcp"},
		{Name: "gRPC Health", Protocol: request.ProtocolGRPC, URL: "grpc://127.0.0.1:8085",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"\"\n}"},
		{Name: "gRPC Health of example", Protocol: request.ProtocolGRPC, URL: "grpc://127.0.0.1:8085",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"example\"\n}"},
		{Name: "gRPC Health from .proto", Protocol: request.ProtocolGRPC, URL: "grpc://127.0.0.1:8086",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"example\"\n}",
			Proto: request.ProtoSource{Files: []string{exampleHealthProto()}}},
	}}
}

// exampleHealthProto is the example server's own copy of health.proto, for
// its port without reflection. The sample lives in no folder, so a relative
// path would be read from wherever the program was started (the repository
// for make run, cmd/workshop for its tests, anything from an IDE); a debug
// build runs from its source tree, so the file is found from this source
// file's place in it. A build with -trimpath records no such place and gets
// the path from the repository's root.
func exampleHealthProto() string {
	relative := filepath.FromSlash("internal/exampleserver/protos/grpc/health/v1/health.proto")
	_, source, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(source) {
		return relative
	}
	root := filepath.Join(filepath.Dir(source), "..", "..")
	return filepath.Join(root, relative)
}
