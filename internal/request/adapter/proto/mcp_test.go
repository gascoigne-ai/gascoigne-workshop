package proto

import (
	"bytes"
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

	kernelrequest "github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// mcpReply answers one JSON-RPC request the way a small server would.
func mcpReply(other map[string]any) map[string]any {
	reply := func(result any) map[string]any {
		return map[string]any{"jsonrpc": "2.0", "id": other["id"], "result": result}
	}
	switch other["method"] {
	case "initialize":
		return reply(map[string]any{"protocolVersion": mcpVersion, "serverInfo": map[string]any{"name": "mock", "version": "1"},
			"capabilities": map[string]any{"tools": map[string]any{}, "prompts": map[string]any{}}})
	case "tools/list":
		return reply(map[string]any{"tools": []any{map[string]any{"name": "echo", "inputSchema": map[string]any{"type": "object"}}}})
	case "prompts/list":
		return reply(map[string]any{"prompts": []any{}})
	case "tools/call":
		p, _ := other["params"].(map[string]any)
		if p["name"] == "boom" {
			return map[string]any{"jsonrpc": "2.0", "id": other["id"], "error": map[string]any{"code": -32602, "message": "no such tool"}}
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
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		mu.Lock()
		seen = append(seen, call.Method+" "+call.Header.Get("Mcp-Session-Id")+" "+call.Header.Get("MCP-Protocol-Version")+" "+call.Header.Get("Authorization"))
		mu.Unlock()
		if call.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(call.Body)
		var m map[string]any
		json.Unmarshal(body, &m)
		out := mcpReply(m)
		if out == nil {
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		b, _ := json.Marshal(out)
		if m["method"] == "initialize" {
			writer.Header().Set("Mcp-Session-Id", "sess-1")
		}
		if m["method"] == "tools/call" {
			writer.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(writer, "event: message\ndata: %s\n\n", b)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), seen...) }
}

// waitFor reads events until one's text contains want.
func waitFor(t *testing.T, session kernelrequest.Session, want string) kernelrequest.Event {
	t.Helper()
	for {
		select {
		case e, ok := <-session.Events():
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
	s, err := MCP{}.Connect(context.Background(), kernelrequest.Request{Protocol: "mcp", URL: url, Auth: kernelrequest.Auth{Type: kernelrequest.AuthBearer, Token: "tok"}})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, "Connected to mock")
	e := waitFor(t, s, `"tools":[`)
	if e.Dir != kernelrequest.Received || e.Kind != kernelrequest.KindResponse {
		t.Fatalf("%+v", e)
	}
	waitFor(t, s, `"prompts":[]`)

	if err := s.Send(context.Background(), `call echo {"x":1}`); err != nil {
		t.Fatal(err)
	}
	sent := waitFor(t, s, `"tools/call"`)
	if sent.Dir != kernelrequest.Sent {
		t.Fatalf("sent event %+v", sent)
	}
	if got := waitFor(t, s, "echo {"); got.Dir != kernelrequest.Received || !strings.Contains(got.Text, `{\"x\":1}`) {
		t.Fatalf("%+v", got)
	}

	s.Send(context.Background(), `call boom`)
	if got := waitFor(t, s, "no such tool"); got.Kind != kernelrequest.KindError {
		t.Fatalf("error kind = %q", got.Kind)
	}
	if err := s.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, `"ping"`)
	s.Close(context.Background())
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
	s, err := MCP{}.Connect(context.Background(), kernelrequest.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	for text, want := range map[string]string{
		"":                   "nothing to send",
		"call":               "a name is needed",
		"call x {nope":       "must be JSON",
		"read":               "needs a resource URI",
		"tools/list {nope":   "must be JSON",
		`{"jsonrpc": broken`: "not valid JSON",
	} {
		if err := s.Send(context.Background(), text); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Send(%q) = %v, want %q", text, err, want)
		}
	}
	if err := s.Send(context.Background(), `{"jsonrpc":"2.0","id":99,"method":"tools/list"}`); err != nil { // raw message
		t.Fatal(err)
	}
	waitFor(t, s, `"id":99`)
}

func TestMCPSendIsAOneShotToolsList(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	resp, err := MCP{}.Send(context.Background(), kernelrequest.Request{URL: url})
	if err != nil || resp.Err != "" || resp.Status != 200 || !strings.Contains(string(resp.Body), `"echo"`) {
		t.Fatalf("%+v %v", resp, err)
	}
}

func TestMCPRejectsBadURLsAndFailedServers(t *testing.T) {
	if _, err := (MCP{}).Connect(context.Background(), kernelrequest.Request{URL: "ftp://x"}); err == nil {
		t.Fatal("ftp URL accepted")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := MCP{}.Connect(context.Background(), kernelrequest.Request{URL: srv.URL})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
	resp, _ := MCP{}.Send(context.Background(), kernelrequest.Request{URL: srv.URL})
	if !strings.Contains(resp.Err, "401") {
		t.Fatalf("one-shot error %q", resp.Err)
	}
}

func TestMCPConnectCanBeCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		io.Copy(io.Discard, call.Body) // reading the body lets the server notice a disconnect
		<-call.Context().Done()        // never answers
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := (MCP{}).Connect(ctx, kernelrequest.Request{URL: srv.URL}); err == nil {
		t.Fatal("connect to a silent server succeeded")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("cancellation took too long")
	}
}

// The stdio server is this test binary run again with MCP_HELPER set.
func TestMain(runner *testing.M) {
	switch os.Getenv("MCP_HELPER") {
	case "deaf": // never reads stdin, so closing it does not end the process
		time.Sleep(time.Minute)
		return
	case "stderr-lines": // fifty numbered lines on stderr, then a failure
		for i := 1; i <= 50; i++ {
			fmt.Fprintf(os.Stderr, "line %d\n", i)
		}
		os.Exit(1)
	case "stderr-huge": // one megabyte on stderr without a newline, then a failure
		os.Stderr.Write(bytes.Repeat([]byte("x"), 1<<20))
		os.Exit(1)
	case "flood": // one line over the message limit, then silence
		os.Stdout.Write(bytes.Repeat([]byte("a"), mcpMaxMessage+1<<20))
		time.Sleep(time.Minute)
		return
	}
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
	os.Exit(runner.Run())
}

func TestMCPStdioSession(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCP_HELPER", "1")
	s, err := MCP{}.Connect(context.Background(), kernelrequest.Request{URL: kernelrequest.StdioPrefix + shellQuote(exe)})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, "Connected to mock")
	waitFor(t, s, `"echo"`)
	s.Send(context.Background(), `call echo {"a":"b"}`)
	waitFor(t, s, "echo {")
	s.Close(context.Background())
	for e := range s.Events() {
		if strings.Contains(e.Text, "Connection lost") {
			t.Fatalf("a deliberate close reported as a loss: %+v", e)
		}
	}
}

func TestMCPStdioMissingCommand(t *testing.T) {
	if _, err := (MCP{}).Connect(context.Background(), kernelrequest.Request{URL: "stdio:/no/such/binary"}); err == nil {
		t.Fatal("started a missing command")
	}
	if _, err := (MCP{}).Connect(context.Background(), kernelrequest.Request{URL: "stdio:"}); err == nil {
		t.Fatal("started an empty command")
	}
}

func TestMCPConnectAnnouncesTheServerInfoAsAnEvent(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	s, err := MCP{}.Connect(context.Background(), kernelrequest.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	var e kernelrequest.Event
	for e.Kind != kernelrequest.KindInitialize {
		e = waitFor(t, s, `"serverInfo"`) // the response is logged too, as traffic
	}
	if e.Dir != kernelrequest.Info {
		t.Fatalf("%+v", e)
	}
	server, ok := request.ParseServerInfo(e.Text)
	if !ok || server.Name != "mock" || server.Version != "1" {
		t.Fatalf("%+v %v", server, ok)
	}
}

func TestMCPRequestReturnsTheResultOrTheServersError(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	s, err := MCP{}.Connect(context.Background(), kernelrequest.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	requester, ok := s.(kernelrequest.Requester)
	if !ok {
		t.Fatal("an MCP session answers requests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := requester.Request(ctx, `call echo {"x":1}`)
	if err != nil || !strings.Contains(result, `echo {`) {
		t.Fatalf("result %q err %v", result, err)
	}
	if _, err := requester.Request(ctx, `call boom`); err == nil || !strings.Contains(err.Error(), "no such tool") {
		t.Fatalf("err = %v, want the server's message", err)
	}
	if _, err := requester.Request(ctx, `call`); err == nil {
		t.Fatal("a call without a name is refused")
	}
}

// stdioEnds starts the helper in mode and returns what finish reports.
func stdioEnds(t *testing.T, mode string) (*mcpStdio, <-chan error) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MCP_HELPER", mode)
	ended := make(chan error, 1)
	m, err := newMCPStdio(shellQuote(exe), func([]byte) {}, func(err error) { ended <- err })
	if err != nil {
		t.Fatal(err)
	}
	return m, ended
}

// close started a kill timer and stopped it with a defer in the same call,
// so a server that ignores the end of its input was never killed.
func TestMCPStdioCloseKillsAServerThatIgnoresItsInput(t *testing.T) {
	m, ended := stdioEnds(t, "deaf")
	m.close(context.Background())
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("the server is still running after close")
	}
}

// A line over the message limit stopped the reader, but the process kept
// running and cmd.Wait never returned, so the session never ended.
func TestMCPStdioEndsWhenAServerSendsAnOversizedLine(t *testing.T) {
	_, ended := stdioEnds(t, "flood")
	select {
	case err := <-ended:
		if err == nil || !strings.Contains(err.Error(), "too long") {
			t.Fatalf("ended with %v, want the oversized line named", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the session does not end after an oversized line")
	}
}

// An initialize result that was not an object was decoded with its error
// ignored, so the session went on with no server info and no word why.
func TestAnUnreadableInitializeResultIsLogged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		body, _ := io.ReadAll(call.Body)
		var m map[string]any
		json.Unmarshal(body, &m)
		out := mcpReply(m)
		if m["method"] == "initialize" {
			out["result"] = "not an object"
		}
		if out == nil {
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(out)
	}))
	defer srv.Close()
	s, err := MCP{}.Connect(context.Background(), kernelrequest.Request{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	e := waitFor(t, s, "initialize result")
	if e.Dir != kernelrequest.Info || e.Kind != kernelrequest.KindError {
		t.Fatalf("%+v", e)
	}
}

func TestMCPCloseWithAnEndedContextSkipsTheGoodbye(t *testing.T) {
	url, seen := mcpHTTPServer(t)
	s, err := MCP{}.Connect(context.Background(), kernelrequest.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, s, "Connected to mock")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.Close(ctx)
	for range s.Events() {
	}
	if log := strings.Join(seen(), "\n"); strings.Contains(log, "DELETE") {
		t.Fatalf("a DELETE went out after its context ended:\n%s", log)
	}
}
