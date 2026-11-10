package ui

import (
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
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
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	tab.url = view.NewTextField(tab.Req.URL)

	a.send(tab) // the Send button of a socket connects
	until(t, a, "connection", func() bool { return strings.Contains(logText(tab), "Connected") })
	if tab.Session == nil {
		t.Fatal("no session after connecting")
	}

	tab.msg = view.NewTextField("hello")
	a.sendMessage(tab)
	until(t, a, "echo", func() bool { return strings.Contains(logText(tab), "↓ hello") })
	if !strings.Contains(logText(tab), "↑ hello") {
		t.Fatalf("sent message missing:\n%s", logText(tab))
	}
	if tab.msg.Text != "" {
		t.Fatalf("composer keeps %q after sending", tab.msg.Text)
	}

	a.send(tab) // connected: the same button disconnects
	until(t, a, "close", func() bool { return tab.Session == nil })
}

func TestWebSocketConnectFailureShowsInTheLog(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws://127.0.0.1:1/"
	a.send(tab)
	until(t, a, "failure", func() bool { return tab.Session == nil && !tab.Connecting && len(tab.wsView.Lines) > 0 })
	if tab.Session != nil {
		t.Fatal("session after a failed connect")
	}
	if tab.Active() {
		t.Fatal("a failed connect leaves the button on Disconnect")
	}
}

func TestClearEmptiesTheMessageLog(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	a.send(tab)
	until(t, a, "connection", func() bool { return tab.Session != nil && len(tab.wsView.Lines) > 0 })
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
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	tab.Reconnect = true
	a.send(tab)
	until(t, a, "second connection", func() bool { return conns.Load() >= 2 && tab.Session != nil })
	if !strings.Contains(logText(tab), "Reconnecting") {
		t.Fatalf("log does not mention the retry:\n%s", logText(tab))
	}

	a.send(tab) // a deliberate disconnect must not reconnect
	until(t, a, "close", func() bool { return tab.Session == nil })
	time.Sleep(100 * time.Millisecond)
	a.step(core.Input{})
	if tab.Session != nil || tab.Connecting || conns.Load() != 2 {
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
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	a.send(tab)
	until(t, a, "close", func() bool { return strings.Contains(logText(tab), "Closed by server") && tab.Session == nil })
	time.Sleep(100 * time.Millisecond)
	a.step(core.Input{})
	if conns.Load() != 1 {
		t.Fatalf("%d connections, want 1", conns.Load())
	}
}

func TestComposerSendsHexAsABinaryMessage(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	a.send(tab)
	until(t, a, "connection", func() bool { return tab.Session != nil })
	tab.msg = view.NewTextField("hex: de ad BE ef")
	a.sendMessage(tab)
	until(t, a, "sent bytes", func() bool { return strings.Contains(logText(tab), "↑ deadbeef") })
	tab.msg = view.NewTextField("hex: xyz")
	a.sendMessage(tab)
	if !strings.Contains(logText(tab), "hex:") {
		t.Fatalf("a bad hex message must be explained:\n%s", logText(tab))
	}
	a.disconnect(tab)
}

func TestMCPToolsFillTheComposerFromTheirSchemas(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	list := `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"search","inputSchema":{"type":"object","properties":{"q":{"type":"string"},"limit":{"type":"integer","default":5}},"required":["q"]}},{"name":"ping"}]}}`
	a.applyWS(wsEvent{tab: tab, ev: reqport.Event{Dir: reqport.Received, Kind: "response", Text: list}, hasEv: true})
	if len(tab.Tools) != 2 {
		t.Fatalf("tools %+v", tab.Tools)
	}
	items := a.toolMenu(tab)
	if len(items) != 2 || items[0].label != "search" {
		t.Fatalf("menu %+v", items)
	}
	items[0].run() // a tool with properties opens its form; its defaults make the call
	if a.dialog == nil || a.dialog.form == nil {
		t.Fatal("picking a tool with arguments opens the form")
	}
	for _, button := range a.dialog.buttons {
		if button.label == "Fill composer" {
			button.run()
		}
	}
	if got := tab.msg.Text; got != `call search {"q":"","limit":5}` {
		t.Fatalf("composer %q", got)
	}
	items[1].run()
	if got := tab.msg.Text; got != "call ping {}" {
		t.Fatalf("composer %q", got)
	}
	a.applyWS(wsEvent{tab: tab, closed: true})
	if tab.Tools != nil {
		t.Fatal("tools must go when the session ends")
	}
}

func TestMCPResourcesAndPromptsFillTheComposer(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	a.applyWS(wsEvent{tab: tab, hasEv: true, ev: reqport.Event{Dir: reqport.Received, Kind: "response", Text: `{"result":{"resources":[{"uri":"file:///a.txt","name":"A"}]}}`}})
	a.applyWS(wsEvent{tab: tab, hasEv: true, ev: reqport.Event{Dir: reqport.Received, Kind: "response", Text: `{"result":{"prompts":[{"name":"review","arguments":[{"name":"code","required":true}]}]}}`}})
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
	tab.Req.Protocol = "sse"
	tab.Req.URL = srv.URL
	a.send(tab)
	until(t, a, "event", func() bool { return strings.Contains(logText(tab), "hello") })
	until(t, a, "end of stream", func() bool { return tab.Session == nil })
	if !strings.Contains(view.Dump(a.messagesPane(wsPanes[0], tab, nil)), "Not connected") {
		t.Fatal("the pane did not build")
	}
	if strings.Contains(view.Dump(a.messagesPane(wsPanes[0], tab, nil)), "Connect to send messages") {
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
	tab.Req.Protocol, tab.Req.URL, tab.Reconnect = "sse", srv.URL, true
	defer func() {
		if t.Failed() {
			t.Logf("log:\n%s\nwantOpen=%v connecting=%v sess=%v gen=%d", logText(tab), tab.WantOpen, tab.Connecting, tab.Session != nil, tab.Gen)
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
	if tab.LastEventID != "" {
		t.Fatal("a deliberate disconnect must forget the id")
	}
}

func TestFormatMessageMakesTheComposerOneLineOfJSON(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.msg = view.NewTextField("{\n  \"op\": \"ping\",\n  \"n\": 1\n}")
	a.formatMessage(tab)
	if tab.msg.Text != `{"op":"ping","n":1}` {
		t.Fatalf("composer = %q", tab.msg.Text)
	}
}

func TestFormatMessageLeavesBrokenTextAloneAndSaysWhy(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.msg = view.NewTextField(`{"op": }`)
	before := len(a.toasts)
	a.formatMessage(tab)
	if tab.msg.Text != `{"op": }` {
		t.Fatalf("composer = %q, broken text must stay as typed", tab.msg.Text)
	}
	if len(a.toasts) != before+1 || !strings.Contains(a.toasts[len(a.toasts)-1].Text, "not JSON") {
		t.Fatalf("toasts = %+v", a.toasts)
	}
}

// Regression: connect dialed with context.Background and checked the
// timeout only after the dial returned, so a server that accepted the
// connection but never answered the handshake held the tab on
// "Connecting" forever.
func TestWebSocketConnectGivesUpWhenTheServerNeverAnswersTheHandshake(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) }) // runs before srv.Close, which waits for the handler
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	tab.Req.TimeoutSec = 1

	a.send(tab)
	deadline := time.Now().Add(4 * time.Second)
	for tab.Connecting && time.Now().Before(deadline) {
		a.step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}
	if tab.Connecting {
		t.Fatal("still connecting after the timeout")
	}
	if !strings.Contains(logText(tab), "deadline exceeded") {
		t.Fatalf("the log does not say the connect timed out:\n%s", logText(tab))
	}
}

// slowSession is a session whose Send and Ping wait to be released and
// record the order they were asked in.
type slowSession struct {
	release chan struct{}
	mu      sync.Mutex
	sent    []string
}

func (s *slowSession) Events() <-chan reqport.Event { return nil }
func (s *slowSession) Close() error                 { return nil }
func (s *slowSession) Send(text string) error {
	<-s.release
	s.mu.Lock()
	s.sent = append(s.sent, text)
	s.mu.Unlock()
	return nil
}
func (s *slowSession) Ping() error { return s.Send("ping") }

func (s *slowSession) log() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.sent...)
}

func TestSendingAndPingingDoNotBlockTheFrameAndKeepTheirOrder(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	session := &slowSession{release: make(chan struct{})}
	tab.Session = session

	done := make(chan struct{})
	go func() {
		tab.msg = view.NewTextField("one")
		a.sendMessage(tab)
		a.pingSocket(tab)
		tab.msg = view.NewTextField("two")
		a.sendMessage(tab)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sending blocked the caller")
	}
	close(session.release)
	until(t, a, "all sent", func() bool { return len(session.log()) == 3 })
	if got := strings.Join(session.log(), ","); got != "one,ping,two" {
		t.Fatalf("order %s", got)
	}
}
