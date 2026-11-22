package proto

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// Whatever bytes a server sends, deliver returns: it logs the message, never
// panics and never waits for anything.
func FuzzMCPDeliver(fuzz *testing.F) {
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":1,"result":` + toolsResult + `}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":2,"result":{"resources":[{"uri":"file:///a","name":"a"}]}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":3,"result":{"prompts":[{"name":"p","arguments":[{"name":"x","required":true}]}]}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":4,"method":"sampling/createMessage","params":{}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":5,"error":{"code":-32601,"message":"method not found"}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"t","inputSchema":true}]}}`))
	fuzz.Add([]byte(`not json`))
	fuzz.Fuzz(func(t *testing.T, message []byte) {
		session := &mcpSession{eventLog: newTestLog(64), tr: closedTransport{}, waiters: map[int64]chan rpcAnswer{}}
		session.deliver(message)
	})
}

// A batch from the server is split into messages that together are no
// larger than the body, whatever the body holds.
func FuzzDeliverJSON(fuzz *testing.F) {
	fuzz.Add([]byte(`[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}]`))
	fuzz.Add([]byte(`  {"jsonrpc":"2.0","id":1,"result":{}}`))
	fuzz.Add([]byte(`[]`))
	fuzz.Add([]byte(`[1,"a",null,[]]`))
	fuzz.Add([]byte(`[{"id":`))
	fuzz.Fuzz(func(t *testing.T, body []byte) {
		total := 0
		deliverJSON(body, func(message []byte) { total += len(message) })
		if total > len(body) {
			t.Fatalf("delivered %d bytes from a body of %d", total, len(body))
		}
	})
}

// countingTransport counts the replies a session writes.
type countingTransport struct{ sent int }

func (counting *countingTransport) send(context.Context, []byte) error {
	counting.sent++
	return nil
}
func (counting *countingTransport) setVersion(string)           {}
func (counting *countingTransport) close(context.Context) error { return nil }

func serverRequestSeeds(fuzz *testing.F) {
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":2,"method":"roots/list"}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":"a","method":"nope","params":[1]}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":3,"method":"sampling/createMessage","params":{"messages":[{"role":"user","content":{"type":"text","text":"hi"}}],"systemPrompt":"s","maxTokens":10}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":4,"method":"sampling/createMessage","params":{"messages":[{"content":{"type":"image"}}]}}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":5,"method":"sampling/createMessage","params":"x"}`))
	fuzz.Add([]byte(`{"jsonrpc":"2.0","id":6,"method":"sampling/createMessage"}`))
}

func answeringSession(counting *countingTransport) *mcpSession {
	session := &mcpSession{eventLog: newTestLog(64), tr: counting, waiters: map[int64]chan rpcAnswer{}}
	session.sampler = func(context.Context, request.SamplingRequest) (request.SamplingResult, bool) {
		return request.SamplingResult{}, false
	}
	return session
}

// A request of the server gets exactly one reply, whatever its method and
// params, and answering it never panics or waits.
func FuzzAnswerServer(fuzz *testing.F) {
	serverRequestSeeds(fuzz)
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		var message rpcMessage
		if json.Unmarshal(data, &message) != nil || message.ID == nil {
			t.Skip()
		}
		counting := &countingTransport{}
		answeringSession(counting).answerServer(message)
		if counting.sent != 1 {
			t.Fatalf("%d replies to one request", counting.sent)
		}
	})
}

// Sampling params of any shape give a reply and never panic.
func FuzzSample(fuzz *testing.F) {
	serverRequestSeeds(fuzz)
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		var message rpcMessage
		if json.Unmarshal(data, &message) != nil || message.ID == nil {
			t.Skip()
		}
		reply := answeringSession(&countingTransport{}).sample(message)
		if reply.Result == nil && reply.Error == nil {
			t.Fatal("a reply with neither result nor error")
		}
	})
}
