package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// echoServer is a WebSocket server that sends every message back.
func echoServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := ws.Upgrade(w, r)
		if err != nil {
			return
		}
		defer c.Close(ws.CloseNormal, "")
		for {
			m, err := c.Read()
			if err != nil || m.Type == ws.Close {
				return
			}
			if m.Type == ws.Text {
				c.Write(m.Type, m.Data)
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// until steps the app until cond holds.
func until(t *testing.T, a *App, what string, cond func() bool) {
	t.Helper()
	for range 400 {
		a.step(core.Input{})
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func logText(tab *reqTab) string {
	var b strings.Builder
	for _, l := range tab.wsView.Lines {
		for _, s := range l.Spans {
			b.WriteString(s.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestWebSocketTabConnectsSendsAndDisconnects(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "websocket"
	tab.req.URL = echoServer(t)
	tab.url = vml.NewTextField(tab.req.URL)

	a.send(tab) // the Send button of a socket connects
	until(t, a, "connection", func() bool { return strings.Contains(logText(tab), "Connected") })
	if tab.sess == nil {
		t.Fatal("no session after connecting")
	}

	tab.msg = vml.NewTextField("hello")
	a.sendMessage(tab)
	until(t, a, "echo", func() bool { return strings.Contains(logText(tab), "↓ hello") })
	if !strings.Contains(logText(tab), "↑ hello") {
		t.Fatalf("sent message missing:\n%s", logText(tab))
	}
	if tab.msg.Text != "" {
		t.Fatalf("composer keeps %q after sending", tab.msg.Text)
	}

	a.send(tab) // connected: the same button disconnects
	until(t, a, "close", func() bool { return tab.sess == nil })
}

func TestWebSocketConnectFailureShowsInTheLog(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "websocket"
	tab.req.URL = "ws://127.0.0.1:1/"
	a.send(tab)
	until(t, a, "failure", func() bool { return tab.sess == nil && !tab.connecting && len(tab.wsView.Lines) > 0 })
	if tab.sess != nil {
		t.Fatal("session after a failed connect")
	}
}

func TestClearEmptiesTheMessageLog(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "websocket"
	tab.req.URL = echoServer(t)
	a.send(tab)
	until(t, a, "connection", func() bool { return tab.sess != nil && len(tab.wsView.Lines) > 0 })
	a.clearLog(tab)
	if len(tab.wsView.Lines) != 0 {
		t.Fatal("log not cleared")
	}
	a.disconnect(tab)
}

func TestWebSocketReconnectsAfterTheServerClosesWhenAsked(t *testing.T) {
	retryDelay = 10 * time.Millisecond
	var conns atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := ws.Upgrade(w, r)
		if err != nil {
			return
		}
		defer c.Close(ws.CloseNormal, "")
		if conns.Add(1) == 1 {
			return // the first connection is dropped at once
		}
		for {
			if _, err := c.Read(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "websocket"
	tab.req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	tab.reconnect = true
	a.send(tab)
	until(t, a, "second connection", func() bool { return conns.Load() >= 2 && tab.sess != nil })
	if !strings.Contains(logText(tab), "Reconnecting") {
		t.Fatalf("log does not mention the retry:\n%s", logText(tab))
	}

	a.send(tab) // a deliberate disconnect must not reconnect
	until(t, a, "close", func() bool { return tab.sess == nil })
	time.Sleep(100 * time.Millisecond)
	a.step(core.Input{})
	if tab.sess != nil || tab.connecting || conns.Load() != 2 {
		t.Fatalf("reconnected after a disconnect: %d connections", conns.Load())
	}
}

func TestWebSocketWithoutReconnectStaysClosed(t *testing.T) {
	retryDelay = 10 * time.Millisecond
	var conns atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := ws.Upgrade(w, r); err == nil {
			conns.Add(1)
			c.Close(ws.CloseNormal, "")
		}
	}))
	t.Cleanup(srv.Close)
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "websocket"
	tab.req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	a.send(tab)
	until(t, a, "close", func() bool { return strings.Contains(logText(tab), "Closed by server") && tab.sess == nil })
	time.Sleep(100 * time.Millisecond)
	a.step(core.Input{})
	if conns.Load() != 1 {
		t.Fatalf("%d connections, want 1", conns.Load())
	}
}

func TestComposerSendsHexAsABinaryMessage(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "websocket"
	tab.req.URL = echoServer(t)
	a.send(tab)
	until(t, a, "connection", func() bool { return tab.sess != nil })
	tab.msg = vml.NewTextField("hex: de ad BE ef")
	a.sendMessage(tab)
	until(t, a, "sent bytes", func() bool { return strings.Contains(logText(tab), "↑ deadbeef") })
	tab.msg = vml.NewTextField("hex: xyz")
	a.sendMessage(tab)
	if !strings.Contains(logText(tab), "hex:") {
		t.Fatalf("a bad hex message must be explained:\n%s", logText(tab))
	}
	a.disconnect(tab)
}
