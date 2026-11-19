package proto

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

type kindAt struct {
	dir  port.Direction
	kind port.EventKind
}

// collect reads events until every wanted (direction, kind) pair was seen.
func collect(t *testing.T, session port.Session, want ...kindAt) {
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
	session, err := WebSocket{}.Connect(context.Background(), model.Request{Protocol: "websocket", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	ctx := context.Background()
	if err := session.Send(ctx, "hi"); err != nil {
		t.Fatal(err)
	}
	if err := session.(port.BinarySender).SendBinary(ctx, []byte{3}); err != nil {
		t.Fatal(err)
	}
	if err := session.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	collect(t, session,
		kindAt{port.Received, port.KindText}, kindAt{port.Received, port.KindBinary},
		kindAt{port.Received, port.KindPing}, kindAt{port.Received, port.KindPong},
		kindAt{port.Sent, port.KindText}, kindAt{port.Sent, port.KindBinary}, kindAt{port.Sent, port.KindPing})
}

func TestMCPDeliverNamesTheKindOfEachMessage(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want port.EventKind
	}{
		{"response", `{"jsonrpc":"2.0","id":1,"result":{}}`, port.KindResponse},
		{"error response", `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"no"}}`, port.KindError},
		{"request", `{"jsonrpc":"2.0","id":1,"method":"nothing/known"}`, port.KindRequest},
		{"notification", `{"jsonrpc":"2.0","method":"notifications/message"}`, port.KindNotification},
		{"invalid", `not json`, port.KindInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &mcpSession{eventLog: newTestLog(8), tr: closedTransport{}, waiters: map[int64]chan rpcAnswer{}}
			s.deliver([]byte(tc.raw))
			event := <-s.events
			if event.Dir != port.Received || event.Kind != tc.want {
				t.Errorf("event %+v, want Received %s", event, tc.want)
			}
		})
	}
}

func TestMCPConnectLogsOpenAndInitializeKinds(t *testing.T) {
	url, _ := mcpHTTPServer(t)
	session, err := MCP{}.Connect(context.Background(), model.Request{Protocol: "mcp", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	collect(t, session, kindAt{port.Info, port.KindOpen}, kindAt{port.Info, port.KindInitialize})
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
		kindAt{port.Info, port.KindOpen}, kindAt{port.Sent, port.KindMessage},
		kindAt{port.Received, port.KindMessage}, kindAt{port.Info, port.KindEnd}, kindAt{port.Info, port.KindClose})
}

// An event stream let the server choose Event.Kind through its event name,
// so a stream could pose as an error or as the MCP initialize note.
func TestSSEEventNameIsNotTheKind(t *testing.T) {
	for _, name := range []string{"error", "initialize", "open", "close", "text", ""} {
		t.Run("name "+name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "text/event-stream")
				if name != "" {
					fmt.Fprintf(writer, "event: %s\n", name)
				}
				fmt.Fprint(writer, "data: payload\n\n")
				writer.(http.Flusher).Flush()
				<-request.Context().Done()
			}))
			defer server.Close()
			session, err := SSE{}.Connect(context.Background(), model.Request{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close(context.Background())
			var event port.Event
			for event.Dir != port.Received {
				event = next(t, session)
			}
			if event.Kind != port.KindMessage || event.Name != name {
				t.Errorf("kind %q name %q, want %q and %q", event.Kind, event.Name, port.KindMessage, name)
			}
		})
	}
}
