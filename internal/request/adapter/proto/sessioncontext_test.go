package proto

import (
	"context"
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"net/http"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
)

// openSessions are the sessions that write, each over a peer that takes
// whatever it is sent.
func openSessions(t *testing.T) map[string]port.Session {
	t.Helper()
	url := wsServer(t, func(conn *ws.Conn, _ *http.Request) {
		for {
			if _, err := conn.Read(); err != nil {
				return
			}
		}
	})
	conn, err := ws.Dial(context.Background(), url, ws.Options{})
	if err != nil {
		t.Fatal(err)
	}
	socket := &wsSession{eventLog: newEventLog(), conn: conn}
	go socket.readLoop()
	mcp := &mcpSession{eventLog: newEventLog(), tr: closedTransport{}, waiters: map[int64]chan rpcAnswer{}}
	sessions := map[string]port.Session{"websocket": socket, "mcp": mcp}
	t.Cleanup(func() {
		for _, session := range sessions {
			session.Close(context.Background())
		}
	})
	return sessions
}

func TestAWriteWithAnEndedContextSendsNothing(t *testing.T) {
	writes := map[string]func(ctx context.Context, session port.Session) error{
		"send": func(ctx context.Context, session port.Session) error { return session.Send(ctx, "ping") },
		"ping": func(ctx context.Context, session port.Session) error { return session.Ping(ctx) },
	}
	for sessionName, session := range openSessions(t) {
		for writeName, write := range writes {
			t.Run(sessionName+"/"+writeName, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if err := write(ctx, session); !errors.Is(err, context.Canceled) {
					t.Fatalf("err = %v, want the context's", err)
				}
				select {
				case e := <-session.Events():
					t.Fatalf("logged %+v for a write that should not happen", e)
				case <-time.After(50 * time.Millisecond):
				}
			})
		}
	}
}
