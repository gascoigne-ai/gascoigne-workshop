package proto

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func TestMCPAnswersRequestsTheServerSends(t *testing.T) {
	var mu sync.Mutex
	replies := map[string]map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		body, _ := io.ReadAll(call.Body)
		var m map[string]any
		json.Unmarshal(body, &m)
		if m["method"] == nil && m["id"] != nil { // an answer to one of the server's requests
			mu.Lock()
			replies[string(body)[:0]+toKey(m["id"])] = m
			mu.Unlock()
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		out := mcpReply(m)
		if out == nil {
			writer.WriteHeader(http.StatusAccepted)
			return
		}
		b, _ := json.Marshal(out)
		writer.Header().Set("Content-Type", "text/event-stream")
		// the answer, then requests of the server's own on the same stream
		io.WriteString(writer, "data: "+string(b)+"\n\n")
		if m["method"] == "tools/list" {
			io.WriteString(writer, `data: {"jsonrpc":"2.0","id":"s1","method":"roots/list"}`+"\n\n")
			io.WriteString(writer, `data: {"jsonrpc":"2.0","id":"s2","method":"ping"}`+"\n\n")
			io.WriteString(writer, `data: {"jsonrpc":"2.0","id":"s3","method":"sampling/createMessage"}`+"\n\n")
		}
	}))
	defer srv.Close()
	s, err := MCP{}.Connect(context.Background(), request.Request{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background())
	waitFor(t, s, `"sampling/createMessage"`)
	waitFor(t, s, `method not found`) // our refusal shows in the log as a sent response
	for i := 0; i < 200; i++ {        // the three answers go out independently
		mu.Lock()
		n := len(replies)
		mu.Unlock()
		if n == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(replies) != 3 {
		t.Fatalf("replies %v", replies)
	}
	if r := replies["s1"]["result"].(map[string]any); r["roots"] == nil {
		t.Errorf("roots/list answer %v", replies["s1"])
	}
	if replies["s2"]["result"] == nil {
		t.Errorf("ping answer %v", replies["s2"])
	}
	if e, _ := replies["s3"]["error"].(map[string]any); !strings.Contains(e["message"].(string), "sampling/createMessage") {
		t.Errorf("unknown method answer %v", replies["s3"])
	}
}

func toKey(value any) string {
	s, _ := value.(string)
	return s
}
