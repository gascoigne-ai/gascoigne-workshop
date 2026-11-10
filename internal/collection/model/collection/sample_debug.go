//go:build !release

package collection

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

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
		{Name: "Index", Protocol: "rest", Method: "GET", URL: "http://127.0.0.1:8081/"},
		{Name: "Echo", Protocol: "rest", Method: "POST", URL: "http://127.0.0.1:8081/echo",
			BodyType: model.BodyJSON, Body: "{\n  \"hello\": \"world\"\n}"},
		{Name: "Status 418", Protocol: "rest", Method: "GET", URL: "http://127.0.0.1:8081/status/418"},
		{Name: "Delay 500ms", Protocol: "rest", Method: "GET", URL: "http://127.0.0.1:8081/delay/500"},
		{Name: "Auth", Protocol: "rest", Method: "GET", URL: "http://127.0.0.1:8081/auth",
			Auth: model.Auth{Type: model.AuthBearer, Token: "example-token"}},
		{Name: "Auth without credentials", Protocol: "rest", Method: "GET", URL: "http://127.0.0.1:8081/auth"},
		{Name: "WebSocket Echo", Protocol: "websocket", URL: "ws://127.0.0.1:8082"},
		{Name: "SSE Ticks", Protocol: "sse", URL: "http://127.0.0.1:8083/events?count=5"},
		{Name: "MCP Tools", Protocol: "mcp", URL: "http://127.0.0.1:8084/mcp"},
		{Name: "gRPC Health", Protocol: "grpc", URL: "grpc://127.0.0.1:8085",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"\"\n}"},
		{Name: "gRPC Health of example", Protocol: "grpc", URL: "grpc://127.0.0.1:8085",
			Method: "grpc.health.v1.Health/Check", Body: "{\n  \"service\": \"example\"\n}"},
	}}
}
