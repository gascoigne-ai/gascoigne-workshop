package protocol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// debugRecorder is a debugFunc that keeps what it is given.
type debugRecorder struct {
	mu    sync.Mutex
	lines []string
	seen  chan struct{}
}

func newDebugRecorder() *debugRecorder { return &debugRecorder{seen: make(chan struct{}, 16)} }

func (rec *debugRecorder) log(message string, args ...any) {
	rec.mu.Lock()
	line := message
	for _, arg := range args {
		line += " " + toText(arg)
	}
	rec.lines = append(rec.lines, line)
	rec.mu.Unlock()
	rec.seen <- struct{}{}
}

func toText(value any) string {
	if err, ok := value.(error); ok {
		return err.Error()
	}
	if text, ok := value.(string); ok {
		return text
	}
	return "?"
}

// waitLine waits for a debug line containing part.
func (rec *debugRecorder) waitLine(t *testing.T, part string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		rec.mu.Lock()
		for _, line := range rec.lines {
			if strings.Contains(line, part) {
				rec.mu.Unlock()
				return
			}
		}
		rec.mu.Unlock()
		select {
		case <-rec.seen:
		case <-deadline:
			t.Fatalf("no debug line with %q, got %q", part, rec.lines)
		}
	}
}

func debugHTTP(t *testing.T, url string, rec *debugRecorder, notes chan error) *mcpHTTP {
	t.Helper()
	h, err := newMCPHTTP(request.Request{URL: url}, Network{}, func([]byte) {}, func(error) {}, func(err error) { notes <- err })
	if err != nil {
		t.Fatal(err)
	}
	h.debug = rec.log
	t.Cleanup(h.cancel)
	return h
}

func TestListenLogsWhyThereIsNoServerStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusMethodNotAllowed) // the normal answer of a server with no stream
	}))
	defer srv.Close()
	rec, notes := newDebugRecorder(), make(chan error, 1)
	debugHTTP(t, srv.URL, rec, notes).listen()
	rec.waitLine(t, "405")
	select {
	case err := <-notes:
		t.Fatalf("a missing stream is normal, but it was noted: %v", err)
	default:
	}
}

func TestListenLogsAFailedConnection(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens any more
	rec := newDebugRecorder()
	debugHTTP(t, url, rec, make(chan error, 1)).listen()
	rec.waitLine(t, "listen")
}

func TestCloseLogsARefusedGoodbye(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		conn, _, err := writer.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close() // drops the DELETE without an answer
		}
	}))
	defer srv.Close()
	rec := newDebugRecorder()
	h := debugHTTP(t, srv.URL, rec, make(chan error, 1))
	h.setSession("sess-1")
	h.close(context.Background())
	rec.waitLine(t, "end the session")
}

func TestStdioCloseLogsAnInputThatCannotBeClosed(t *testing.T) {
	m, ended := stdioEnds(t, "stderr-lines")
	<-ended // the process is gone and its input pipe closed
	rec := newDebugRecorder()
	m.debug = rec.log
	m.close(context.Background())
	rec.waitLine(t, "close the server's input")
}
