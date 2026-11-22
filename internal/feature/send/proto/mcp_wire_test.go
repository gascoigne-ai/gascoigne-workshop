package proto

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// wireServer is an MCP server over HTTP that records the raw bytes of
// every message the client posts. After tools/list it sends the client
// an unknown request and a sampling request.
type wireServer struct {
	url  string
	mu   sync.Mutex
	body []string
}

func newWireServer(t *testing.T) *wireServer {
	t.Helper()
	server := &wireServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		if call.Method != http.MethodPost {
			return
		}
		raw, _ := io.ReadAll(call.Body)
		server.mu.Lock()
		server.body = append(server.body, string(raw))
		server.mu.Unlock()
		var message map[string]any
		_ = json.Unmarshal(raw, &message)
		out := mcpReply(message)
		if out == nil {
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		b, _ := json.Marshal(out)
		writer.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(writer, "data: "+string(b)+"\n\n")
		if message["method"] == "tools/list" {
			io.WriteString(writer, `data: {"jsonrpc":"2.0","id":"s1","method":"foo/bar"}`+"\n\n")
			io.WriteString(writer, `data: {"jsonrpc":"2.0","id":"s2","method":"sampling/createMessage","params":`+samplingParams+`}`+"\n\n")
		}
	}))
	t.Cleanup(srv.Close)
	server.url = srv.URL
	return server
}

// waitForBody waits until the client has posted exactly want.
func (server *wireServer) waitForBody(t *testing.T, want string) {
	t.Helper()
	for range 300 {
		server.mu.Lock()
		found := slices.Contains(server.body, want)
		server.mu.Unlock()
		if found {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	t.Fatalf("the client never posted %s; posted %q", want, server.body)
}

// The messages the client writes are pinned byte for byte: servers and
// recorded logs depend on them.
func TestMCPWireBytesOfTheClientsMessages(t *testing.T) {
	decline := func(context.Context, request.SamplingRequest) (request.SamplingResult, bool) {
		return request.SamplingResult{}, false
	}
	answer := func(context.Context, request.SamplingRequest) (request.SamplingResult, bool) {
		return request.SamplingResult{Text: "Hello!", Model: "the user"}, true
	}
	tests := []struct {
		name    string
		sampler request.Sampler
		want    []string
	}{
		{"no sampler", nil, []string{
			`{"id":1,"jsonrpc":"2.0","method":"initialize","params":{"capabilities":{},"clientInfo":{"name":"gascoigne-workshop","version":"0"},"protocolVersion":"2025-06-18"}}`,
			`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
			`{"id":2,"jsonrpc":"2.0","method":"tools/list"}`,
			`{"error":{"code":-32601,"message":"method not found: foo/bar"},"id":"s1","jsonrpc":"2.0"}`,
			`{"error":{"code":-32601,"message":"method not found: sampling/createMessage"},"id":"s2","jsonrpc":"2.0"}`,
		}},
		{"sampler declines", decline, []string{
			`{"id":1,"jsonrpc":"2.0","method":"initialize","params":{"capabilities":{"sampling":{}},"clientInfo":{"name":"gascoigne-workshop","version":"0"},"protocolVersion":"2025-06-18"}}`,
			`{"error":{"code":-1,"message":"User rejected sampling request"},"id":"s2","jsonrpc":"2.0"}`,
		}},
		{"sampler answers", answer, []string{
			`{"id":"s2","jsonrpc":"2.0","result":{"content":{"text":"Hello!","type":"text"},"model":"the user","role":"assistant","stopReason":"endTurn"}}`,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newWireServer(t)
			session, err := MCP{Sampler: tt.sampler}.Connect(context.Background(), request.Request{URL: server.url})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close(context.Background())
			for _, want := range tt.want {
				server.waitForBody(t, want)
			}
		})
	}
}

// Send is the one-shot form: it connects and returns the tools/list result.
func TestMCPSendOneShotReturnsTheToolsListResult(t *testing.T) {
	server := newWireServer(t)
	response, err := MCP{}.Send(context.Background(), request.Request{URL: server.url})
	if err != nil || response.Err != "" {
		t.Fatalf("err = %v, response.Err = %q", err, response.Err)
	}
	if response.Status != http.StatusOK || string(response.Body) != `{"tools":[{"inputSchema":{"type":"object"},"name":"echo"}]}` {
		t.Errorf("response = %d %s", response.Status, response.Body)
	}
}
