//go:build !release

package collection

import (
	"strings"
	"testing"
)

func TestDebugSampleIsOnlyTheExampleServer(t *testing.T) {
	sample := Sample()
	if sample.Name != ExampleServerFolder || len(sample.Folders) != 0 {
		t.Fatalf("collection %q with %d folders, want %q with the requests at the top", sample.Name, len(sample.Folders), ExampleServerFolder)
	}
	protocols := map[string]bool{}
	for _, request := range sample.Requests {
		protocols[request.Protocol] = true
		if !strings.Contains(request.URL, "127.0.0.1") {
			t.Errorf("%s points at %q, not at the local example server", request.Name, request.URL)
		}
	}
	for _, want := range []string{"rest", "websocket", "sse", "mcp", "grpc"} {
		if !protocols[want] {
			t.Errorf("no %s request in the example server collection", want)
		}
	}
}
