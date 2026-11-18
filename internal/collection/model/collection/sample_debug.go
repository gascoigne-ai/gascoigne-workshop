//go:build !release

package collection

import (
	"path/filepath"
	"runtime"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// ExampleServerFolder is the name of the folder of requests that reach
// cmd/example-server.
const ExampleServerFolder = "Example Server"

// Sample is the collection a debug build shows until a workspace is
// opened: one request per endpoint of cmd/example-server on its default
// ports, and nothing that leaves the machine.
func Sample() model.Collection {
	return model.Collection{Folder: *exampleServerFolder()}
}

// The ports are the defaults of exampleserver.DefaultConfig; a test there
// keeps the two in step.
func exampleServerFolder() *model.Folder {
	return &model.Folder{Name: ExampleServerFolder, Requests: []*model.Request{
		{Name: "Index", Protocol: model.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/"},
		{Name: "Echo", Protocol: model.ProtocolREST, Method: "POST", URL: "http://127.0.0.1:8081/echo",
			BodyType: model.BodyJSON, Body: "{\n  \"hello\": \"world\"\n}"},
		{Name: "Status 418", Protocol: model.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/status/418"},
		{Name: "Delay 500ms", Protocol: model.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/delay/500"},
		{Name: "Auth", Protocol: model.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/auth",
			Auth: model.Auth{Type: model.AuthBearer, Token: "example-token"}},
		{Name: "Auth without credentials", Protocol: model.ProtocolREST, Method: "GET", URL: "http://127.0.0.1:8081/auth"},
		{Name: "WebSocket Echo", Protocol: model.ProtocolWebSocket, URL: "ws://127.0.0.1:8082"},
		{Name: "SSE Ticks", Protocol: model.ProtocolSSE, URL: "http://127.0.0.1:8083/events?count=5"},
		{Name: "MCP Tools", Protocol: model.ProtocolMCP, URL: "http://127.0.0.1:8084/mcp"},
		{Name: "gRPC Health", Protocol: model.ProtocolGRPC, URL: "grpc://127.0.0.1:8085",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"\"\n}"},
		{Name: "gRPC Health of example", Protocol: model.ProtocolGRPC, URL: "grpc://127.0.0.1:8085",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"example\"\n}"},
		{Name: "gRPC Health from .proto", Protocol: model.ProtocolGRPC, URL: "grpc://127.0.0.1:8086",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"example\"\n}",
			Proto: model.ProtoSource{Files: []string{exampleHealthProto()}}},
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
	relative := filepath.FromSlash("internal/request/adapter/exampleserver/protos/grpc/health/v1/health.proto")
	_, source, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(source) {
		return relative
	}
	root := filepath.Join(filepath.Dir(source), "..", "..", "..", "..")
	return filepath.Join(root, relative)
}
