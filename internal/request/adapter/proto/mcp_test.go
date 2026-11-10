package proto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// mcpReply answers one JSON-RPC request the way a small server would.
func mcpReply(m map[string]any) map[string]any {
	reply := func(result any) map[string]any {
		return map[string]any{"jsonrpc": "2.0", "id": m["id"], "result": result}
	}
	switch m["method"] {
	case "initialize":
		return reply(map[string]any{"protocolVersion": mcpVersion, "serverInfo": map[string]any{"name": "mock", "version": "1"},
			"capabilities": map[string]any{"tools": map[string]any{}, "prompts": map[string]any{}}})
	case "tools/list":
		return reply(map[string]any{"tools": []any{map[string]any{"name": "echo", "inputSchema": map[string]any{"type": "object"}}}})
	case "prompts/list":
		return reply(map[string]any{"prompts": []any{}})
	case "tools/call":
		p, _ := m["params"].(map[string]any)
		if p["name"] == "boom" {
			return map[string]any{"jsonrpc": "2.0", "id": m["id"], "error": map[string]any{"code": -32602, "message": "no such tool"}}
		}
		args, _ := json.Marshal(p["arguments"])
		return reply(map[string]any{"content": []any{map[string]any{"type": "text", "text": "echo " + string(args)}}})
	case "ping":
		return reply(map[string]any{})
	}
	return nil
}

// mcpHTTPServer is a streamable HTTP MCP server. tools/call answers as
// an event stream, everything else as plain JSON.
func mcpHTTPServer(t *testing.T) (url string, log func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.Header.Get("Mcp-Session-Id")+" "+r.Header.Get("MCP-Protocol-Version")+" "+r.Header.Get("Authorization"))
		mu.Unlock()
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		json.Unmarshal(body, &m)
		out := mcpReply(m)
		if out == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		b, _ := json.Marshal(out)
		if m["method"] == "initialize" {
			w.Header().Set("Mcp-Session-Id", "sess-1")
		}
		if m["method"] == "tools/call" {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", b)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), seen...) }
}

// waitFor reads events until one's text contains want.
func waitFor(t *testing.T, s Session, want string) Event {
	t.Helper()
	for {
		select {
		case e, ok := <-s.Events():
			if !ok {
				t.Fatalf("events ended while waiting for %q", want)
			}
			if strings.Contains(e.Text, want) {
				return e
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("no event containing %q", want)
		}
	}
}

func TestMCPHTTPSessionInitializesListsAndCalls(t *testing.T) {
	url, seen := mcpHTTPServer(t)
	s, err := MCP{}.Connect(context.Background(), model.Request{Protocol: "mcp", URL: url, Auth: model.Auth{Type: model.AuthBearer, Token: "tok"}})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, "Connected to mock")
	e := waitFor(t, s, `"tools":[`)
	if e.Dir != Received || e.Kind != "response" {
		t.Fatalf("%+v", e)
	}
	waitFor(t, s, `"prompts":[]`)

	if err := s.Send(`call echo {"x":1}`); err != nil {
		t.Fatal(err)
	}
	sent := waitFor(t, s, `"tools/call"`)
	if sent.Dir != Sent {
		t.Fatalf("sent event %+v", sent)
	}
	if got := waitFor(t, s, "echo {"); got.Dir != Received || !strings.Contains(got.Text, `{\"x\":1}`) {
		t.Fatalf("%+v", got)
	}

	s.Send(`call boom`)
	if got := waitFor(t, s, "no such tool"); got.Kind != "error" {
		t.Fatalf("error kind = %q", got.Kind)
	}
	if err := s.Ping(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, `"ping"`)
	s.Close()
	for range s.Events() {
	}
	log := strings.Join(seen(), "\n")
	if !strings.Contains(log, "POST sess-1 "+mcpVersion+" Bearer tok") {
		t.Fatalf("session id, version or auth missing:\n%s", log)
	}
	if !strings.Contains(log, "DELETE sess-1") {
		t.Fatalf("no DELETE on close:\n%s", log)
	}
}

func TestMCPSendShorthandsAndErrors(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	s, err := MCP{}.Connect(context.Background(), model.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for text, want := range map[string]string{
		"":                   "nothing to send",
		"call":               "a name is needed",
		"call x {nope":       "must be JSON",
		"read":               "needs a resource URI",
		"tools/list {nope":   "must be JSON",
		`{"jsonrpc": broken`: "not valid JSON",
	} {
		if err := s.Send(text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Send(%q) = %v, want %q", text, err, want)
		}
	}
	if err := s.Send(`{"jsonrpc":"2.0","id":99,"method":"tools/list"}`); err != nil { // raw message
		t.Fatal(err)
	}
	waitFor(t, s, `"id":99`)
}

func TestMCPSendIsAOneShotToolsList(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	resp, err := MCP{}.Send(context.Background(), model.Request{URL: url})
	if err != nil || resp.Err != "" || resp.Status != 200 || !strings.Contains(string(resp.Body), `"echo"`) {
		t.Fatalf("%+v %v", resp, err)
	}
}

func TestMCPRejectsBadURLsAndFailedServers(t *testing.T) {
	if _, err := (MCP{}).Connect(context.Background(), model.Request{URL: "ftp://x"}); err == nil {
		t.Fatal("ftp URL accepted")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := MCP{}.Connect(context.Background(), model.Request{URL: srv.URL})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
	resp, _ := MCP{}.Send(context.Background(), model.Request{URL: srv.URL})
	if !strings.Contains(resp.Err, "401") {
		t.Fatalf("one-shot error %q", resp.Err)
	}
}

func TestMCPConnectCanBeCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body) // reading the body lets the server notice a disconnect
		<-r.Context().Done()        // never answers
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := (MCP{}).Connect(ctx, model.Request{URL: srv.URL}); err == nil {
		t.Fatal("connect to a silent server succeeded")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("cancellation took too long")
	}
}

// The stdio server is this test binary run again with MCP_HELPER set.
func TestMain(m *testing.M) {
	if os.Getenv("MCP_HELPER") == "1" {
		dec := json.NewDecoder(os.Stdin)
		for {
			var msg map[string]any
			if err := dec.Decode(&msg); err != nil {
				return
			}
			if out := mcpReply(msg); out != nil {
				b, _ := json.Marshal(out)
				fmt.Println(string(b))
			}
		}
	}
	os.Exit(m.Run())
}

func TestMCPStdioSession(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCP_HELPER", "1")
	s, err := MCP{}.Connect(context.Background(), model.Request{URL: StdioPrefix + shellQuote(exe)})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, "Connected to mock")
	waitFor(t, s, `"echo"`)
	s.Send(`call echo {"a":"b"}`)
	waitFor(t, s, "echo {")
	s.Close()
	for e := range s.Events() {
		if strings.Contains(e.Text, "Connection lost") {
			t.Fatalf("a deliberate close reported as a loss: %+v", e)
		}
	}
}

func TestMCPStdioMissingCommand(t *testing.T) {
	if _, err := (MCP{}).Connect(context.Background(), model.Request{URL: "stdio:/no/such/binary"}); err == nil {
		t.Fatal("started a missing command")
	}
	if _, err := (MCP{}).Connect(context.Background(), model.Request{URL: "stdio:"}); err == nil {
		t.Fatal("started an empty command")
	}
}

func TestMCPConnectAnnouncesTheServerInfoAsAnEvent(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	s, err := MCP{}.Connect(context.Background(), model.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var e Event
	for e.Kind != "initialize" {
		e = waitFor(t, s, `"serverInfo"`) // the response is logged too, as traffic
	}
	if e.Dir != Info {
		t.Fatalf("%+v", e)
	}
	server, ok := model.ParseServerInfo(e.Text)
	if !ok || server.Name != "mock" || server.Version != "1" {
		t.Fatalf("%+v %v", server, ok)
	}
}
