package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func TestMCPToolsFillTheComposerFromTheirSchemas(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "mcp"
	list := `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"search","inputSchema":{"type":"object","properties":{"q":{"type":"string"},"limit":{"type":"integer","default":5}},"required":["q"]}},{"name":"ping"}]}}`
	a.applyWS(wsEvent{tab: tab, ev: proto.Event{Dir: proto.Received, Kind: "response", Text: list}, hasEv: true})
	if len(tab.tools) != 2 {
		t.Fatalf("tools %+v", tab.tools)
	}
	items := a.toolMenu(tab)
	if len(items) != 2 || items[0].label != "search" {
		t.Fatalf("menu %+v", items)
	}
	items[0].run()
	if got := tab.msg.Text; got != `call search {"limit":5,"q":""}` {
		t.Fatalf("composer %q", got)
	}
	items[1].run()
	if got := tab.msg.Text; got != "call ping {}" {
		t.Fatalf("composer %q", got)
	}
	a.applyWS(wsEvent{tab: tab, closed: true})
	if tab.tools != nil {
		t.Fatal("tools must go when the session ends")
	}
}

func TestMCPResourcesAndPromptsFillTheComposer(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "mcp"
	a.applyWS(wsEvent{tab: tab, hasEv: true, ev: proto.Event{Dir: proto.Received, Kind: "response", Text: `{"result":{"resources":[{"uri":"file:///a.txt","name":"A"}]}}`}})
	a.applyWS(wsEvent{tab: tab, hasEv: true, ev: proto.Event{Dir: proto.Received, Kind: "response", Text: `{"result":{"prompts":[{"name":"review","arguments":[{"name":"code","required":true}]}]}}`}})
	a.resourceMenu(tab)[0].run()
	if tab.msg.Text != "read file:///a.txt" {
		t.Fatalf("composer %q", tab.msg.Text)
	}
	a.promptMenu(tab)[0].run()
	if tab.msg.Text != `prompt review {"code":""}` {
		t.Fatalf("composer %q", tab.msg.Text)
	}
}

func TestSSETabListensAndShowsNoComposer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: ping\ndata: hello\n\n")
	}))
	defer srv.Close()
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol = "sse"
	tab.req.URL = srv.URL
	a.send(tab)
	until(t, a, "event", func() bool { return strings.Contains(logText(tab), "hello") })
	until(t, a, "end of stream", func() bool { return tab.sess == nil })
	if !strings.Contains(vml.Dump(a.messagesPane(wsPanes[0], tab, nil)), "Not connected") {
		t.Fatal("the pane did not build")
	}
	if strings.Contains(vml.Dump(a.messagesPane(wsPanes[0], tab, nil)), "Connect to send messages") {
		t.Fatal("an SSE tab must not show the composer")
	}
}

func TestSSEReconnectResumesFromTheLastEventID(t *testing.T) {
	retryDelay = 10 * time.Millisecond
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Last-Event-ID"))
		n := len(seen)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		if n == 1 {
			io.WriteString(w, "id: 41\ndata: first\n\n") // then the stream ends: a drop
			return
		}
		io.WriteString(w, "id: 42\ndata: second\n\n")
		w.(http.Flusher).Flush() // the client waits for the headers of a stream that stays open
		<-r.Context().Done()
	}))
	defer func() { srv.CloseClientConnections(); srv.Close() }()
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Protocol, tab.req.URL, tab.reconnect = "sse", srv.URL, true
	defer func() {
		if t.Failed() {
			t.Logf("log:\n%s\nwantOpen=%v connecting=%v sess=%v gen=%d", logText(tab), tab.wantOpen, tab.connecting, tab.sess != nil, tab.wsGen)
		}
	}()
	a.send(tab)
	until(t, a, "the second stream's event", func() bool { return strings.Contains(logText(tab), "second") || t.Failed() })
	if !strings.Contains(logText(tab), "second") {
		t.Fatalf("log:\n%s", logText(tab))
	}
	mu.Lock()
	got := append([]string(nil), seen...)
	mu.Unlock()
	if len(got) < 2 || got[0] != "" || got[1] != "41" {
		t.Fatalf("Last-Event-ID per connection: %q", got)
	}
	a.send(tab) // disconnect
	if tab.lastEventID != "" {
		t.Fatal("a deliberate disconnect must forget the id")
	}
}
