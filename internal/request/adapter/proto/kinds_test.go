package proto

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
)

type kindAt struct {
	dir  request.Direction
	kind request.EventKind
}

// collect reads events until every wanted (direction, kind) pair was seen.
func collect(t *testing.T, session request.Session, want ...kindAt) {
	t.Helper()
	missing := map[kindAt]bool{}
	for _, pair := range want {
		missing[pair] = true
	}
	for len(missing) > 0 {
		event := next(t, session)
		delete(missing, kindAt{event.Dir, event.Kind})
	}
}

func TestWebSocketEventsCarryTheirFrameKinds(t *testing.T) {
	url := wsServer(t, func(conn *ws.Conn, _ *http.Request) {
		conn.Write(ws.Text, []byte("t"))
		conn.Write(ws.Binary, []byte{1, 2})
		conn.Write(ws.Ping, nil)
		conn.Write(ws.Pong, nil)
		for {
			if message, err := conn.Read(); err != nil || message.Type == ws.Close {
				return
			}
		}
	})
	session, err := WebSocket{}.Connect(context.Background(), request.Request{Protocol: "websocket", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	ctx := context.Background()
	if err := session.Send(ctx, "hi"); err != nil {
		t.Fatal(err)
	}
	if err := session.(request.BinarySender).SendBinary(ctx, []byte{3}); err != nil {
		t.Fatal(err)
	}
	if err := session.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	collect(t, session,
		kindAt{request.Received, request.KindText}, kindAt{request.Received, request.KindBinary},
		kindAt{request.Received, request.KindPing}, kindAt{request.Received, request.KindPong},
		kindAt{request.Sent, request.KindText}, kindAt{request.Sent, request.KindBinary}, kindAt{request.Sent, request.KindPing})
}

func TestMCPDeliverNamesTheKindOfEachMessage(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want request.EventKind
	}{
		{"response", `{"jsonrpc":"2.0","id":1,"result":{}}`, request.KindResponse},
		{"error response", `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"no"}}`, request.KindError},
		{"request", `{"jsonrpc":"2.0","id":1,"method":"nothing/known"}`, request.KindRequest},
		{"notification", `{"jsonrpc":"2.0","method":"notifications/message"}`, request.KindNotification},
		{"invalid", `not json`, request.KindInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &mcpSession{eventLog: newTestLog(8), tr: closedTransport{}, waiters: map[int64]chan rpcAnswer{}}
			s.deliver([]byte(tc.raw))
			event := <-s.events
			if event.Dir != request.Received || event.Kind != tc.want {
				t.Errorf("event %+v, want Received %s", event, tc.want)
			}
		})
	}
}

func TestMCPConnectLogsOpenAndInitializeKinds(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	session, err := MCP{}.Connect(context.Background(), request.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	collect(t, session, kindAt{request.Info, request.KindOpen}, kindAt{request.Info, request.KindInitialize})
}

func TestGRPCEventsCarryTheirKinds(t *testing.T) {
	target, _ := grpcStub(t)
	session, err := GRPC{}.Connect(grpcContext(t), grpcRequest(target, "grpc.reflection.v1.ServerReflection/ServerReflectionInfo", ""))
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	if err := session.Send(context.Background(), `{"listServices": ""}`); err != nil {
		t.Fatal(err)
	}
	if err := session.Send(context.Background(), "end"); err != nil {
		t.Fatal(err)
	}
	collect(t, session,
		kindAt{request.Info, request.KindOpen}, kindAt{request.Sent, request.KindMessage},
		kindAt{request.Received, request.KindMessage}, kindAt{request.Info, request.KindEnd}, kindAt{request.Info, request.KindClose})
}

// An event stream let the server choose Event.Kind through its event name,
// so a stream could pose as an error or as the MCP initialize note.
func TestSSEEventNameIsNotTheKind(t *testing.T) {
	for _, name := range []string{"error", "initialize", "open", "close", "text", ""} {
		t.Run("name "+name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
				writer.Header().Set("Content-Type", "text/event-stream")
				if name != "" {
					fmt.Fprintf(writer, "event: %s\n", name)
				}
				fmt.Fprint(writer, "data: payload\n\n")
				writer.(http.Flusher).Flush()
				<-call.Context().Done()
			}))
			defer server.Close()
			session, err := SSE{}.Connect(context.Background(), request.Request{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close(context.Background())
			var event request.Event
			for event.Dir != request.Received {
				event = next(t, session)
			}
			if event.Kind != request.KindMessage || event.Name != name {
				t.Errorf("kind %q name %q, want %q and %q", event.Kind, event.Name, request.KindMessage, name)
			}
		})
	}
}
