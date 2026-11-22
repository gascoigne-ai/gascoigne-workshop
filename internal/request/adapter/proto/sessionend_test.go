package proto

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
)

// floodedSessions opens one session of every streaming protocol whose
// event queue is full and unread, so an emit is waiting for room.
func floodedSessions(t *testing.T) map[string]request.Session {
	t.Helper()
	const flood = 3 * eventBuffer
	sessions := map[string]request.Session{}

	socketURL := wsServer(t, func(conn *ws.Conn, _ *http.Request) {
		for range flood {
			conn.Write(ws.Text, []byte("m"))
		}
		conn.Read()
	})
	socket, err := WebSocket{}.Connect(context.Background(), request.Request{URL: socketURL})
	if err != nil {
		t.Fatal(err)
	}
	sessions["websocket"] = socket

	stream, err := SSE{}.Connect(context.Background(), request.Request{URL: sseServer(t, flood, true)})
	if err != nil {
		t.Fatal(err)
	}
	sessions["sse"] = stream

	target, _ := grpcStub(t)
	call, err := GRPC{}.Connect(grpcContext(t), grpcRequest(target, "grpc.health.v1.Health/Watch", `{"service": "known"}`))
	if err != nil {
		t.Fatal(err)
	}
	sessions["grpc"] = call

	mcpURL, _ := mcpHTTPServer(t)
	rpc, err := MCP{}.Connect(context.Background(), request.Request{Protocol: "mcp", URL: mcpURL})
	if err != nil {
		t.Fatal(err)
	}
	go func() { // every ping logs a sent event, which fills the queue
		for range flood {
			if rpc.Ping(context.Background()) != nil {
				return
			}
		}
	}()
	sessions["mcp"] = rpc

	time.Sleep(100 * time.Millisecond) // let the queues fill
	t.Cleanup(func() {
		for _, session := range sessions {
			session.Close(context.Background())
		}
	})
	return sessions
}

// Close frees a reader that waits for room in a full queue, ends the event
// stream and adds no note about a lost connection: the user closed it.
func TestClosingASessionWithAFullQueueEndsItsEventsWithoutALossNote(t *testing.T) {
	for name, session := range floodedSessions(t) {
		t.Run(name, func(t *testing.T) {
			closesWithinASecond(t, func() { session.Close(context.Background()) })
			deadline := time.After(5 * time.Second)
			for {
				select {
				case event, ok := <-session.Events():
					if !ok {
						return
					}
					if event.Kind == request.KindClose && strings.Contains(event.Text, "lost") {
						t.Fatalf("a closed session reported a loss: %+v", event)
					}
				case <-deadline:
					t.Fatal("the event channel stayed open after Close")
				}
			}
		})
	}
}

func TestClosingASessionTwiceIsHarmless(t *testing.T) {
	for name, session := range floodedSessions(t) {
		t.Run(name, func(t *testing.T) {
			session.Close(context.Background())
			closesWithinASecond(t, func() { session.Close(context.Background()) })
		})
	}
}
