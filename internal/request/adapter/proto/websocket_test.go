package proto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func wsServer(t *testing.T, handle func(c *ws.Conn, r *http.Request)) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		c, err := ws.Upgrade(writer, request)
		if err != nil {
			return
		}
		defer c.Close(ws.CloseNormal, "")
		handle(c, request)
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func echoHandler(conn *ws.Conn, _ *http.Request) {
	for {
		m, err := conn.Read()
		if err != nil || m.Type == ws.Close {
			return
		}
		if m.Type == ws.Text || m.Type == ws.Binary {
			conn.Write(m.Type, m.Data)
		}
	}
}

// next waits for the next event.
func next(t *testing.T, session Session) Event {
	t.Helper()
	select {
	case e, ok := <-session.Events():
		if !ok {
			t.Fatal("events closed")
		}
		return e
	case <-time.After(3 * time.Second):
		t.Fatal("no event")
	}
	return Event{}
}

func TestWebSocketSessionLogsMessagesInBothDirections(t *testing.T) {
	url := wsServer(t, echoHandler)
	s, err := WebSocket{}.Connect(context.Background(), model.Request{Protocol: "websocket", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if e := next(t, s); e.Dir != Info || !strings.Contains(e.Text, "Connected") {
		t.Fatalf("first event %+v", e)
	}
	if err := s.Send("hi there"); err != nil {
		t.Fatal(err)
	}
	if e := next(t, s); e.Dir != Sent || e.Text != "hi there" || e.Size != 8 || e.Time.IsZero() {
		t.Fatalf("sent %+v", e)
	}
	if e := next(t, s); e.Dir != Received || e.Text != "hi there" || e.Kind != "text" {
		t.Fatalf("received %+v", e)
	}
}

// A send was logged only after its write, so an echo that came back in
// between was logged first, and the test above failed now and then (about
// 1 run in 100 under load). The server here echoes only once the send is
// being logged, which makes that gap certain: the send must be logged
// before the echo.
func TestWebSocketSessionLogsASendBeforeItsEcho(t *testing.T) {
	cases := []struct {
		name string
		send func(session *wsSession) error
		kind string
	}{
		{"text", func(session *wsSession) error { return session.Send("hi there") }, "text"},
		{"binary", func(session *wsSession) error { return session.SendBinary([]byte{0xde, 0xad}) }, "binary"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			release := make(chan struct{})
			url := wsServer(t, func(conn *ws.Conn, _ *http.Request) {
				message, err := conn.Read()
				if err != nil {
					return
				}
				select {
				case <-release:
				case <-time.After(3 * time.Second):
				}
				conn.Write(message.Type, message.Data)
				conn.Read() // until the client closes
			})
			conn, err := ws.Dial(context.Background(), url, ws.Options{})
			if err != nil {
				t.Fatal(err)
			}
			session := &wsSession{conn: conn, events: make(chan Event, eventBuffer), done: make(chan struct{})}
			var logging atomic.Bool
			session.now = func() time.Time {
				if logging.CompareAndSwap(false, true) { // the first event logged is the send
					close(release)
					deadline := time.Now().Add(300 * time.Millisecond)
					for len(session.events) == 0 && time.Now().Before(deadline) {
						time.Sleep(time.Millisecond)
					}
				}
				return time.Now()
			}
			go session.readLoop()
			defer session.Close()
			if err := tc.send(session); err != nil {
				t.Fatal(err)
			}
			if e := next(t, session); e.Dir != Sent || e.Kind != tc.kind {
				t.Fatalf("first event %+v, want the %s send", e, tc.kind)
			}
			if e := next(t, session); e.Dir != Received || e.Kind != tc.kind {
				t.Fatalf("second event %+v, want the %s echo", e, tc.kind)
			}
		})
	}
}

func TestWebSocketSessionReportsServerCloseAndThenEnds(t *testing.T) {
	url := wsServer(t, func(conn *ws.Conn, _ *http.Request) { conn.Close(ws.CloseGoingAway, "restart") })
	s, err := WebSocket{}.Connect(context.Background(), model.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	next(t, s) // connected
	e := next(t, s)
	if e.Dir != Info || e.Code != ws.CloseGoingAway || !strings.Contains(e.Text, "restart") {
		t.Fatalf("close event %+v", e)
	}
	select {
	case _, ok := <-s.Events():
		if ok {
			t.Fatal("more events after close")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("events not closed")
	}
	if err := s.Send("late"); err == nil {
		t.Fatal("sending on an ended session must fail")
	}
}

func TestWebSocketSendsHeadersAuthAndSubprotocols(t *testing.T) {
	got := make(chan http.Header, 1)
	url := wsServer(t, func(conn *ws.Conn, request *http.Request) { got <- request.Header })
	req := model.Request{
		URL: url,
		Headers: []model.KV{
			{Name: "X-Trace", Value: "1", Enabled: true},
			{Name: "X-Off", Value: "2", Enabled: false},
			{Name: "Sec-WebSocket-Protocol", Value: "chat, v2", Enabled: true},
		},
		Auth: model.Auth{Type: model.AuthBearer, Token: "tok"},
	}
	s, err := WebSocket{}.Connect(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	h := <-got
	if h.Get("X-Trace") != "1" || h.Get("X-Off") != "" || h.Get("Authorization") != "Bearer tok" || h.Get("Sec-WebSocket-Protocol") != "chat, v2" {
		t.Fatalf("headers %v", h)
	}
}

func TestWebSocketConnectFailureIsAnError(t *testing.T) {
	if _, err := (WebSocket{}).Connect(context.Background(), model.Request{URL: "ws://127.0.0.1:1"}); err == nil {
		t.Fatal("want an error")
	}
	if _, err := (WebSocket{}).Connect(context.Background(), model.Request{URL: "http://x"}); err == nil {
		t.Fatal("want an error for a non-ws scheme")
	}
}

func TestWebSocketSlowConsumerBlocksTheReaderInsteadOfDroppingMessages(t *testing.T) {
	const n = 600 // more than the event buffer
	url := wsServer(t, func(conn *ws.Conn, _ *http.Request) {
		for i := 0; i < n; i++ {
			conn.Write(ws.Text, []byte("m"))
		}
		conn.Read()
	})
	s, err := WebSocket{}.Connect(context.Background(), model.Request{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	time.Sleep(100 * time.Millisecond) // let the buffer fill
	got := 0
	for got < n {
		if e := next(t, s); e.Dir == Received {
			got++
		}
	}
}

func TestWebSocketSendOnceReturnsTheFirstReply(t *testing.T) {
	url := wsServer(t, echoHandler)
	resp, err := WebSocket{}.Send(context.Background(), model.Request{URL: url, Body: `{"a":1}`, BodyType: model.BodyJSON})
	if err != nil || resp.Err != "" || string(resp.Body) != `{"a":1}` || resp.Size != 7 {
		t.Fatalf("got %+v, %v", resp, err)
	}
	if resp.Status != 101 {
		t.Fatalf("status %d", resp.Status)
	}
}

func TestWebSocketSendOnceReportsConnectFailureInTheResponse(t *testing.T) {
	resp, err := WebSocket{}.Send(context.Background(), model.Request{URL: "ws://127.0.0.1:1"})
	if err != nil || resp.Err == "" {
		t.Fatalf("got %+v, %v", resp, err)
	}
}

func TestRegistryConnectUsesTheStreamingProtocol(t *testing.T) {
	url := wsServer(t, echoHandler)
	r := NewRegistry()
	s, err := r.Connect(context.Background(), model.Request{Protocol: "websocket", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	if _, err := r.Connect(context.Background(), model.Request{Protocol: "rest"}); err == nil {
		t.Fatal("rest does not stream")
	}
	if _, err := r.Connect(context.Background(), model.Request{Protocol: "nope"}); err == nil {
		t.Fatal("unknown protocol")
	}
}
