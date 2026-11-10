//go:build !release

package exampleserver

import (
	"net"
	"net/url"
	"strconv"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
)

// The debug sample collection hard-codes the example server's default
// ports because a model package cannot import this adapter; this keeps
// the two from drifting apart.
func TestDebugSampleRequestsUseTheDefaultPorts(t *testing.T) {
	defaults := DefaultConfig()
	ports := map[string]int{"rest": defaults.REST, "websocket": defaults.WebSocket, "sse": defaults.SSE, "mcp": defaults.MCP, "grpc": defaults.GRPC}
	sample := collection.Sample()
	example := sample.Folders[len(sample.Folders)-1]
	if example.Name != collection.ExampleServerFolder {
		t.Fatalf("last folder = %q, want %q", example.Name, collection.ExampleServerFolder)
	}
	for _, request := range example.Requests {
		parsed, err := url.Parse(request.URL)
		if err != nil {
			t.Fatalf("%s: %v", request.Name, err)
		}
		_, port, err := net.SplitHostPort(parsed.Host)
		if err != nil || port != strconv.Itoa(ports[request.Protocol]) {
			t.Errorf("%s (%s) uses %q, want port %d", request.Name, request.Protocol, parsed.Host, ports[request.Protocol])
		}
	}
}
