package proto

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

// samplingServer is an MCP server that asks the client for a sampling
// completion once tools are listed. It records what the client said in
// initialize and how it answered.
type samplingServer struct {
	url          string
	mu           sync.Mutex
	capabilities map[string]any
	answer       map[string]any
}

const samplingParams = `{"messages":[{"role":"user","content":{"type":"text","text":"Say hi"}},{"role":"assistant","content":{"type":"text","text":"Hi"}}],"systemPrompt":"Be brief","maxTokens":50}`

func newSamplingServer(t *testing.T) *samplingServer {
	t.Helper()
	server := &samplingServer{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		switch {
		case m["method"] == nil && m["id"] == "s1": // the client's answer to the sampling request
			server.mu.Lock()
			server.answer = m
			server.mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
			return
		case m["method"] == "initialize":
			params, _ := m["params"].(map[string]any)
			server.mu.Lock()
			server.capabilities, _ = params["capabilities"].(map[string]any)
			server.mu.Unlock()
		}
		out := mcpReply(m)
		if out == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		b, _ := json.Marshal(out)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: "+string(b)+"\n\n")
		if m["method"] == "tools/list" {
			io.WriteString(w, `data: {"jsonrpc":"2.0","id":"s1","method":"sampling/createMessage","params":`+samplingParams+`}`+"\n\n")
		}
	}))
	t.Cleanup(srv.Close)
	server.url = srv.URL
	return server
}

func (server *samplingServer) waitForAnswer(t *testing.T) map[string]any {
	t.Helper()
	for range 300 {
		server.mu.Lock()
		answer := server.answer
		server.mu.Unlock()
		if answer != nil {
			return answer
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the client never answered the sampling request")
	return nil
}

func TestMCPSamplingPutsTheServersRequestToTheSamplerAndSendsItsText(t *testing.T) {
	server := newSamplingServer(t)
	var asked SamplingRequest
	sampler := func(_ context.Context, request SamplingRequest) (SamplingResult, bool) {
		asked = request
		return SamplingResult{Text: "Hello!", Model: "the user"}, true
	}
	session, err := MCP{Sampler: sampler}.Connect(context.Background(), model.Request{URL: server.url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	answer := server.waitForAnswer(t)
	result, _ := answer["result"].(map[string]any)
	content, _ := result["content"].(map[string]any)
	if result["role"] != "assistant" || content["type"] != "text" || content["text"] != "Hello!" || result["model"] != "the user" {
		t.Errorf("answer = %v", answer)
	}
	if result["stopReason"] != "endTurn" {
		t.Errorf("stopReason = %v", result["stopReason"])
	}
	if asked.SystemPrompt != "Be brief" || asked.MaxTokens != 50 || len(asked.Messages) != 2 ||
		asked.Messages[0] != (SamplingMessage{Role: "user", Text: "Say hi"}) || asked.Messages[1].Role != "assistant" {
		t.Errorf("the sampler was given %+v", asked)
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if _, ok := server.capabilities["sampling"]; !ok {
		t.Errorf("initialize must offer sampling, capabilities = %v", server.capabilities)
	}
}

func TestMCPDecliningASamplingRequestAnswersWithAnError(t *testing.T) {
	server := newSamplingServer(t)
	sampler := func(context.Context, SamplingRequest) (SamplingResult, bool) { return SamplingResult{}, false }
	session, err := MCP{Sampler: sampler}.Connect(context.Background(), model.Request{URL: server.url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	answer := server.waitForAnswer(t)
	failure, _ := answer["error"].(map[string]any)
	if failure == nil || failure["message"] != "User rejected sampling request" {
		t.Fatalf("answer = %v", answer)
	}
}

func TestMCPWithoutASamplerDoesNotOfferSampling(t *testing.T) {
	server := newSamplingServer(t)
	session, err := MCP{}.Connect(context.Background(), model.Request{URL: server.url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	server.waitForAnswer(t) // refused as an unknown method, as before
	server.mu.Lock()
	defer server.mu.Unlock()
	if _, ok := server.capabilities["sampling"]; ok {
		t.Errorf("capabilities = %v", server.capabilities)
	}
}
