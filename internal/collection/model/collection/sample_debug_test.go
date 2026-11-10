//go:build !release

package collection

import "testing"

func TestDebugSampleKeepsTheGitHubFoldersAndAddsTheExampleServer(t *testing.T) {
	sample := Sample()
	if len(sample.Folders) < 4 || sample.Folders[0].Name != "User" || sample.Folders[1].Name != "Repository" {
		t.Fatalf("folders = %d, want the GitHub folders first and one more", len(sample.Folders))
	}
	example := sample.Folders[len(sample.Folders)-1]
	if example.Name != ExampleServerFolder {
		t.Fatalf("last folder = %q, want %q", example.Name, ExampleServerFolder)
	}
	protocols := map[string]bool{}
	for _, request := range example.Requests {
		protocols[request.Protocol] = true
	}
	for _, want := range []string{"rest", "websocket", "sse", "mcp", "grpc"} {
		if !protocols[want] {
			t.Errorf("no %s request in the example server folder", want)
		}
	}
}
