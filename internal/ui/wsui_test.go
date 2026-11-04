package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
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
		a.step(widget.Input{})
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
	tab.url = widget.NewTextField(tab.req.URL)

	a.send(tab) // the Send button of a socket connects
	until(t, a, "connection", func() bool { return strings.Contains(logText(tab), "Connected") })
	if tab.sess == nil {
		t.Fatal("no session after connecting")
	}

	tab.msg = widget.NewTextField("hello")
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
