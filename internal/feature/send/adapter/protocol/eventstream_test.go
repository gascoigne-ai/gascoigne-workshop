package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// endlessEvent is a stream of data lines just over maxEventBytes with no
// blank line, as a server sends that never ends its event.
func endlessEvent() string {
	const lineData = 1 << 10
	line := "data: " + strings.Repeat("x", lineData) + "\n"
	return strings.Repeat(line, maxEventBytes/lineData+1)
}

// A server that sent data lines and never a blank line grew one event
// without bound: the reader kept every line in memory.
func TestReadEventStreamCutsOffAnEventOverTheLimit(t *testing.T) {
	var emitted int
	err := readEventStream(strings.NewReader("data: first\n\n"+endlessEvent()), func(event sseEvent) {
		if len(event.data) > maxEventBytes {
			t.Errorf("an event of %d bytes was emitted", len(event.data))
		}
		emitted++
	})
	if !errors.Is(err, errEventTooLarge) {
		t.Fatalf("err = %v, want errEventTooLarge", err)
	}
	if emitted != 1 {
		t.Fatalf("emitted %d events, want only the one before the oversized one", emitted)
	}
}

// An oversized line silently ended the stream: the scanner's error was
// dropped and the stream looked as if the server had closed it.
func TestReadEventStreamReportsALineOverTheLimit(t *testing.T) {
	stream := "data: " + strings.Repeat("x", maxEventBytes) + "\n\n"
	err := readEventStream(strings.NewReader(stream), func(sseEvent) {})
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Fatalf("err = %v, want bufio.ErrTooLong", err)
	}
}

func TestReadEventStreamReportsAReadError(t *testing.T) {
	broken := io.MultiReader(strings.NewReader("data: a\n\n"), iotest.ErrReader(errBrokenStream))
	err := readEventStream(broken, func(sseEvent) {})
	if !errors.Is(err, errBrokenStream) {
		t.Fatalf("err = %v, want the read error", err)
	}
}

var errBrokenStream = errors.New("broken stream")

// An SSE server that never ended its event kept the session reading,
// and growing, until the user closed it.
func TestSSESessionReportsAnEventOverTheLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(writer, endlessEvent())
		writer.(http.Flusher).Flush()
		<-call.Context().Done()
	}))
	defer srv.Close()
	session, err := SSE{}.Connect(context.Background(), request.Request{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	next(t, session) // connected
	if event := next(t, session); event.Kind != request.KindClose || !strings.Contains(event.Text, "Connection lost") {
		t.Fatalf("got %s %q, want the stream to end with the reason", event.Kind, event.Text[:min(len(event.Text), 80)])
	}
}

// mcpStreamServer answers MCP over HTTP with plain JSON and opens its
// optional GET stream with stream.
func mcpStreamServer(t *testing.T, stream http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		if call.Method == http.MethodGet {
			stream(writer, call)
			return
		}
		body, _ := io.ReadAll(call.Body)
		var message map[string]any
		_ = json.Unmarshal(body, &message)
		out := mcpReply(message)
		if out == nil {
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		answer, _ := json.Marshal(out)
		writer.Header().Set("Content-Type", "application/json")
		writer.Write(answer)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// An MCP server stream that never ended its event grew memory without
// end.
func TestMCPSessionEndsWhenItsStreamSendsAnEventOverTheLimit(t *testing.T) {
	url := mcpStreamServer(t, func(writer http.ResponseWriter, call *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(writer, endlessEvent())
		writer.(http.Flusher).Flush()
		<-call.Context().Done()
	})
	session, err := MCP{}.Connect(context.Background(), request.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	waitFor(t, session, "Connection lost")
}

// The notification stream is optional: when it breaks, only notifications
// are lost, so the session goes on and says why the stream stopped.
func TestMCPSessionGoesOnWhenItsNotificationStreamBreaks(t *testing.T) {
	url := mcpStreamServer(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.Header().Set("Content-Length", "1000") // more than is sent, so the client reads an unexpected EOF
		io.WriteString(writer, "data: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/message\"}\n\n")
	})
	session, err := MCP{}.Connect(context.Background(), request.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	if event := waitFor(t, session, "read the server's stream"); event.Dir != request.Info || event.Kind == request.KindClose {
		t.Fatalf("a broken notification stream ended the session: %+v", event)
	}
	if err := session.Send(context.Background(), `call echo {"x":1}`); err != nil {
		t.Fatal(err)
	}
	waitFor(t, session, "echo {")
}
