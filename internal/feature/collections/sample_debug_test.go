//go:build !release

package collection

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func TestDebugSampleIsOnlyTheExampleServer(t *testing.T) {
	sample := Sample()
	if sample.Name != ExampleServerFolder || len(sample.Folders) != 0 {
		t.Fatalf("collection %q with %d folders, want %q with the requests at the top", sample.Name, len(sample.Folders), ExampleServerFolder)
	}
	protocols := map[request.Protocol]bool{}
	for _, call := range sample.Requests {
		protocols[call.Protocol] = true
		if !strings.Contains(call.URL, "127.0.0.1") {
			t.Errorf("%s points at %q, not at the local example server", call.Name, call.URL)
		}
	}
	for _, want := range []request.Protocol{"rest", "websocket", "sse", "mcp", "grpc"} {
		if !protocols[want] {
			t.Errorf("no %s request in the example server collection", want)
		}
	}
}
