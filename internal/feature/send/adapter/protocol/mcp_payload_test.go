package protocol

import (
	"context"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

const toolsResult = `{"tools":[{"name":"echo","description":"Say it","inputSchema":{"type":"object","properties":{"text":{"type":"string"}}}}]}`

// newPayloadSession is a session on a transport that swallows sends.
func newPayloadSession() *mcpSession {
	return &mcpSession{eventLog: newTestLog(16), tr: closedTransport{}, waiters: map[int64]chan rpcAnswer{}}
}

// receivedEvent drains the sent events and returns the next received one.
func receivedEvent(t *testing.T, s *mcpSession) request.Event {
	t.Helper()
	for {
		select {
		case event := <-s.events:
			if event.Dir == request.Received || event.Kind == request.KindError {
				return event
			}
		default:
			t.Fatal("no received event")
		}
	}
}

func TestAResponseToToolsListCarriesAToolsListPayload(t *testing.T) {
	s := newPayloadSession()
	if err := s.Send(context.Background(), "tools/list"); err != nil {
		t.Fatal(err)
	}
	s.deliver([]byte(`{"jsonrpc":"2.0","id":1,"result":` + toolsResult + `}`))
	event := receivedEvent(t, s)
	list, ok := event.Payload.(request.ToolsList)
	if !ok || len(list.Tools) != 1 || list.Tools[0].Name != "echo" {
		t.Fatalf("payload %#v", event.Payload)
	}
}

func TestResourcesAndPromptsListsCarryTheirPayloads(t *testing.T) {
	cases := []struct {
		method, result string
		check          func(request.Payload) bool
	}{
		{"resources/list", `{"resources":[{"uri":"file:///a.txt","name":"A"}]}`, func(payload request.Payload) bool {
			list, ok := payload.(request.ResourcesList)
			return ok && len(list.Resources) == 1 && list.Resources[0].URI == "file:///a.txt"
		}},
		{"prompts/list", `{"prompts":[{"name":"review","arguments":[{"name":"code","required":true}]}]}`, func(payload request.Payload) bool {
			list, ok := payload.(request.PromptsList)
			return ok && len(list.Prompts) == 1 && list.Prompts[0].Name == "review"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			s := newPayloadSession()
			if err := s.Send(context.Background(), tc.method); err != nil {
				t.Fatal(err)
			}
			s.deliver([]byte(`{"jsonrpc":"2.0","id":1,"result":` + tc.result + `}`))
			if event := receivedEvent(t, s); !tc.check(event.Payload) {
				t.Fatalf("payload %#v", event.Payload)
			}
		})
	}
}

// A result shaped like a tools list is no list unless it answers tools/list.
func TestAToolsShapedResultOfAnotherCallOrANotificationCarriesNoPayload(t *testing.T) {
	cases := []struct {
		name, method, message string
	}{
		{"tools/call response", "call echo {}", `{"jsonrpc":"2.0","id":1,"result":` + toolsResult + `}`},
		{"notification", "", `{"jsonrpc":"2.0","method":"notifications/message","params":` + toolsResult + `}`},
		{"unrequested response", "", `{"jsonrpc":"2.0","id":7,"result":` + toolsResult + `}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newPayloadSession()
			if tc.method != "" {
				if err := s.Send(context.Background(), tc.method); err != nil {
					t.Fatal(err)
				}
			}
			s.deliver([]byte(tc.message))
			if event := receivedEvent(t, s); event.Payload != nil {
				t.Fatalf("payload %#v", event.Payload)
			}
		})
	}
}

func TestARawToolsListTypedInTheComposerStillFillsThePayload(t *testing.T) {
	s := newPayloadSession()
	raw := `{"jsonrpc":"2.0","id":41,"method":"tools/list"}`
	if err := s.Send(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	s.deliver([]byte(`{"jsonrpc":"2.0","id":41,"result":` + toolsResult + `}`))
	if _, ok := receivedEvent(t, s).Payload.(request.ToolsList); !ok {
		t.Fatal("no ToolsList payload")
	}
}

func TestAMalformedListResultGivesAnErrorNoteAndNoPayload(t *testing.T) {
	s := newPayloadSession()
	if err := s.Send(context.Background(), "tools/list"); err != nil {
		t.Fatal(err)
	}
	s.deliver([]byte(`{"jsonrpc":"2.0","id":1,"result":"nope"}`))
	var sawNote bool
	for len(s.events) > 0 {
		event := <-s.events
		if event.Payload != nil {
			t.Fatalf("payload %#v", event.Payload)
		}
		sawNote = sawNote || event.Dir == request.Info && event.Kind == request.KindError
	}
	if !sawNote {
		t.Fatal("no Info error event")
	}
}

func TestAToolWithAMalformedInputSchemaDoesNotDropTheList(t *testing.T) {
	s := newPayloadSession()
	if err := s.Send(context.Background(), "tools/list"); err != nil {
		t.Fatal(err)
	}
	s.deliver([]byte(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"bad","inputSchema":true},{"name":"odd","inputSchema":"x"},{"name":"good"}]}}`))
	list, ok := receivedEvent(t, s).Payload.(request.ToolsList)
	if !ok || len(list.Tools) != 3 {
		t.Fatalf("payload %#v", list)
	}
}
