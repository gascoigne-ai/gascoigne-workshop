package proto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// notifyServer is an MCP server over HTTP with a push stream: a GET
// opens an event stream that announces a changed tool list; each
// tools/list answer names the tool by how many times it was asked.
type notifyServer struct {
	url       string
	listCalls atomic.Int32
	streamHdr chan http.Header // headers of the GET that opened the stream
}

// streamHeaders is what the GET that opened the stream carried, failing
// the test when none arrives.
func (server *notifyServer) streamHeaders(t *testing.T) http.Header {
	t.Helper()
	select {
	case header := <-server.streamHdr:
		return header
	case <-time.After(3 * time.Second):
		t.Fatal("the client never opened the notification stream")
		return nil
	}
}

func newNotifyServer(t *testing.T, streamStatus int) *notifyServer {
	t.Helper()
	server := &notifyServer{streamHdr: make(chan http.Header, 1)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			server.streamHdr <- r.Header.Clone()
			if streamStatus != http.StatusOK {
				w.WriteHeader(streamStatus)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/tools/list_changed\"}\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done() // the stream stays open until the client leaves
			return
		}
		body, _ := io.ReadAll(r.Body)
		var message map[string]any
		_ = json.Unmarshal(body, &message)
		if message["method"] == "tools/list" {
			n := server.listCalls.Add(1)
			b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": message["id"], "result": map[string]any{
				"tools": []any{map[string]any{"name": fmt.Sprintf("tool-%d", n)}}}})
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
			return
		}
		out := mcpReply(message)
		if out == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if message["method"] == "initialize" {
			w.Header().Set("Mcp-Session-Id", "sess-9")
		}
		b, _ := json.Marshal(out)
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	server.url = srv.URL
	return server
}

func TestMCPListensForServerNotificationsAndRefreshesAChangedList(t *testing.T) {
	server := newNotifyServer(t, http.StatusOK)
	session, err := MCP{}.Connect(context.Background(), model.Request{Protocol: "mcp", URL: server.url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	waitFor(t, session, "notifications/tools/list_changed") // pushed on the stream, shown in the log
	waitFor(t, session, `"tool-2"`)                         // the list was asked for again
	header := server.streamHeaders(t)
	if header.Get("Mcp-Session-Id") != "sess-9" || header.Get("MCP-Protocol-Version") == "" {
		t.Errorf("the stream must carry the session: %v", header)
	}
	if header.Get("Accept") != "text/event-stream" {
		t.Errorf("Accept = %q", header.Get("Accept"))
	}
}

func TestMCPWorksWhenTheServerHasNoNotificationStream(t *testing.T) {
	server := newNotifyServer(t, http.StatusMethodNotAllowed)
	session, err := MCP{}.Connect(context.Background(), model.Request{Protocol: "mcp", URL: server.url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	waitFor(t, session, "Connected to mock")
	server.streamHeaders(t) // the client tried
	waitFor(t, session, `"tool-1"`)
	if server.listCalls.Load() != 1 {
		t.Fatalf("the list was asked %d times without any notification", server.listCalls.Load())
	}
}
