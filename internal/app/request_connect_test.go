package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	wire "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// echoServer is a WebSocket server that sends every message back.
func echoServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		c, err := wire.Upgrade(writer, call)
		if err != nil {
			return
		}
		defer c.Close(wire.CloseNormal, "")
		for {
			m, err := c.Read()
			if err != nil || m.Type == wire.Close {
				return
			}
			if m.Type == wire.Text {
				c.Write(m.Type, m.Data)
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func logText(a *App, tab *reqTab) string {
	var b strings.Builder
	for _, l := range a.ws.Lines(tab.Tab) {
		for _, s := range l.Spans {
			b.WriteString(s.Text)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestWebSocketTabConnectsSendsAndDisconnects(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	*a.urlbar.Field(tab.Tab) = widget.NewTextField(tab.Req.URL)

	a.send(tab) // the Send button of a socket connects
	until(t, a, "connection", func() bool { return strings.Contains(logText(a, tab), "Connected") })
	if tab.Session() == nil {
		t.Fatal("no session after connecting")
	}

	*a.ws.Composer(tab.Tab) = widget.NewTextField("hello")
	a.ws.SendMessage(tab.Tab)
	until(t, a, "echo", func() bool { return strings.Contains(logText(a, tab), "↓ hello") })
	if !strings.Contains(logText(a, tab), "↑ hello") {
		t.Fatalf("sent message missing:\n%s", logText(a, tab))
	}
	if a.ws.Composer(tab.Tab).Text != "" {
		t.Fatalf("composer keeps %q after sending", a.ws.Composer(tab.Tab).Text)
	}

	a.send(tab) // connected: the same button disconnects
	until(t, a, "close", func() bool { return tab.Session() == nil })
}

func TestWebSocketConnectFailureShowsInTheLog(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws://127.0.0.1:1/"
	a.send(tab)
	until(t, a, "failure", func() bool {
		return tab.Session() == nil && tab.State() != tabs.ConnConnecting && len(a.ws.Lines(tab.Tab)) > 0
	})
	if tab.Session() != nil {
		t.Fatal("session after a failed connect")
	}
	if tab.Active() {
		t.Fatal("a failed connect leaves the button on Disconnect")
	}
}

func TestClearEmptiesTheMessageLog(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	a.send(tab)
	until(t, a, "connection", func() bool { return tab.Session() != nil && len(a.ws.Lines(tab.Tab)) > 0 })
	a.ws.Clear(tab.Tab)
	if len(a.ws.Lines(tab.Tab)) != 0 {
		t.Fatal("log not cleared")
	}
	a.disconnect(tab)
}

func TestWebSocketReconnectsAfterTheServerClosesWhenAsked(t *testing.T) {
	var conns atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		c, err := wire.Upgrade(writer, call)
		if err != nil {
			return
		}
		defer c.Close(wire.CloseNormal, "")
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

	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	tab.Reconnect = true
	a.send(tab)
	until(t, a, "second connection", func() bool { return conns.Load() >= 2 && tab.Session() != nil })
	if !strings.Contains(logText(a, tab), "Reconnecting") {
		t.Fatalf("log does not mention the retry:\n%s", logText(a, tab))
	}

	a.send(tab) // a deliberate disconnect must not reconnect
	until(t, a, "close", func() bool { return tab.Session() == nil })
	time.Sleep(100 * time.Millisecond)
	a.Step(core.Input{})
	if tab.Session() != nil || tab.State() == tabs.ConnConnecting || conns.Load() != 2 {
		t.Fatalf("reconnected after a disconnect: %d connections", conns.Load())
	}
}

func TestReconnectDelayBelongsToEachApp(t *testing.T) {
	for _, delay := range []time.Duration{10 * time.Millisecond, 20 * time.Millisecond} {
		t.Run(delay.String(), func(t *testing.T) {
			t.Parallel()
			a := &App{retryDelay: delay, bus: bus.New(), ws: ws.New(ws.Deps{})}
			tab := newTab(&request.Request{})
			tab.Reconnect = true
			tab.Begin()
			tab.Failed()                      // a drop that is to be retried
			time.Sleep(20 * time.Millisecond) // let both cases overlap
			a.scheduleRetry(tab)
			if want := "Reconnecting in " + delay.String(); !strings.Contains(logText(a, tab), want) {
				t.Fatalf("log lacks %q:\n%s", want, logText(a, tab))
			}
		})
	}
}

func TestWebSocketWithoutReconnectStaysClosed(t *testing.T) {
	var conns atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		if c, err := wire.Upgrade(writer, call); err == nil {
			conns.Add(1)
			c.Close(wire.CloseNormal, "")
		}
	}))
	t.Cleanup(srv.Close)
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	a.send(tab)
	until(t, a, "close", func() bool { return strings.Contains(logText(a, tab), "Closed by server") && tab.Session() == nil })
	time.Sleep(100 * time.Millisecond)
	a.Step(core.Input{})
	if conns.Load() != 1 {
		t.Fatalf("%d connections, want 1", conns.Load())
	}
}

func TestComposerSendsHexAsABinaryMessage(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	a.send(tab)
	until(t, a, "connection", func() bool { return tab.Session() != nil })
	*a.ws.Composer(tab.Tab) = widget.NewTextField("hex: de ad BE ef")
	a.ws.SendMessage(tab.Tab)
	until(t, a, "sent bytes", func() bool { return strings.Contains(logText(a, tab), "↑ deadbeef") })
	*a.ws.Composer(tab.Tab) = widget.NewTextField("hex: xyz")
	a.ws.SendMessage(tab.Tab)
	if !strings.Contains(logText(a, tab), "hex:") {
		t.Fatalf("a bad hex message must be explained:\n%s", logText(a, tab))
	}
	a.disconnect(tab)
}

func TestMCPToolsFillTheComposerFromTheirSchemas(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	list := `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"search","inputSchema":{"type":"object","properties":{"q":{"type":"string"},"limit":{"type":"integer","default":5}},"required":["q"]}},{"name":"ping"}]}}`
	a.applyWS(wsReceived{wsTarget{tab: tab}, listEvent(list)})
	if len(tab.Tools) != 2 {
		t.Fatalf("tools %+v", tab.Tools)
	}
	a.mcp.OpenToolForm(tab.Tab, tab.Tools[0]) // a tool with properties opens its form; its defaults make the call
	if a.over.Dialog() == nil || !strings.Contains(dialogDump(a), `text="limit"`) {
		t.Fatal("picking a tool with arguments opens the form")
	}
	for _, button := range a.over.Dialog().Buttons {
		if button.Label == "Fill composer" {
			button.Run()
		}
	}
	if got := a.ws.Composer(tab.Tab).Text; got != `call search {"q":"","limit":5}` {
		t.Fatalf("composer %q", got)
	}
	a.mcp.OpenToolForm(tab.Tab, tab.Tools[1])
	if got := a.ws.Composer(tab.Tab).Text; got != "call ping {}" {
		t.Fatalf("composer %q", got)
	}
	openSession(tab, &fakeSession{})
	a.applyWS(wsClosed{wsTarget{tab: tab}})
	if tab.Tools != nil {
		t.Fatal("tools must go when the session ends")
	}
}

func TestSSETabListensAndShowsNoComposer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(writer, "event: ping\ndata: hello\n\n")
	}))
	defer srv.Close()
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "sse"
	tab.Req.URL = srv.URL
	a.send(tab)
	until(t, a, "event", func() bool { return strings.Contains(logText(a, tab), "hello") })
	until(t, a, "end of stream", func() bool { return tab.Session() == nil })
	if !strings.Contains(view.Dump(a.messagesPane(wsPanes[0], tab, nil)), "Not connected") {
		t.Fatal("the pane did not build")
	}
	if strings.Contains(view.Dump(a.messagesPane(wsPanes[0], tab, nil)), "Connect to send messages") {
		t.Fatal("an SSE tab must not show the composer")
	}
}

func TestSSEReconnectResumesFromTheLastEventID(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		mu.Lock()
		seen = append(seen, call.Header.Get("Last-Event-ID"))
		n := len(seen)
		mu.Unlock()
		writer.Header().Set("Content-Type", "text/event-stream")
		if n == 1 {
			io.WriteString(writer, "id: 41\ndata: first\n\n") // then the stream ends: a drop
			return
		}
		io.WriteString(writer, "id: 42\ndata: second\n\n")
		writer.(http.Flusher).Flush() // the client waits for the headers of a stream that stays open
		<-call.Context().Done()
	}))
	defer func() { srv.CloseClientConnections(); srv.Close() }()
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol, tab.Req.URL, tab.Reconnect = "sse", srv.URL, true
	defer func() {
		if t.Failed() {
			t.Logf("log:\n%s\nstate=%d sess=%v gen=%d", logText(a, tab), tab.State(), tab.Session() != nil, tab.Gen)
		}
	}()
	a.send(tab)
	until(t, a, "the second stream's event", func() bool { return strings.Contains(logText(a, tab), "second") || t.Failed() })
	if !strings.Contains(logText(a, tab), "second") {
		t.Fatalf("log:\n%s", logText(a, tab))
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

// Regression: connect dialed with context.Background and checked the
// timeout only after the dial returned, so a server that accepted the
// connection but never answered the handshake held the tab on
// "Connecting" forever.
func TestWebSocketConnectGivesUpWhenTheServerNeverAnswersTheHandshake(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) }) // runs before srv.Close, which waits for the handler
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws" + strings.TrimPrefix(srv.URL, "http")
	tab.Req.TimeoutSec = 1

	a.send(tab)
	deadline := time.Now().Add(4 * time.Second)
	for tab.State() == tabs.ConnConnecting && time.Now().Before(deadline) {
		a.Step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}
	if tab.State() == tabs.ConnConnecting {
		t.Fatal("still connecting after the timeout")
	}
	if !strings.Contains(logText(a, tab), "deadline exceeded") {
		t.Fatalf("the log does not say the connect timed out:\n%s", logText(a, tab))
	}
}

func TestSendingAndPingingDoNotBlockTheFrameAndKeepTheirOrder(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	session := &fakeSession{release: make(chan struct{})}
	openSession(tab, session)

	done := make(chan struct{})
	go func() {
		*a.ws.Composer(tab.Tab) = widget.NewTextField("one")
		a.ws.SendMessage(tab.Tab)
		a.ws.Ping(tab.Tab)
		*a.ws.Composer(tab.Tab) = widget.NewTextField("two")
		a.ws.SendMessage(tab.Tab)
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

// Pins what differs between the composers of the protocols that connect:
// only MCP's hint lists its commands.
func TestOnlyTheMCPComposerShowsTheCommandHint(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	for _, protocol := range []request.Protocol{"websocket", "mcp", "sse", "grpc"} {
		tab.Req.Protocol = protocol
		hint := viewOf(protocol).composer.HintFor(true)
		if mcp := strings.HasPrefix(hint, "call TOOL"); mcp != (protocol == "mcp") {
			t.Errorf("%s composer hint %q", protocol, hint)
		}
	}
}

// Pins what differs between the composers of the protocols that connect:
// only WebSocket offers the { } button that makes the message one line of
// JSON.
func TestOnlyTheWebSocketComposerOffersFormat(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	for _, protocol := range []request.Protocol{"websocket", "mcp", "grpc"} {
		tab.Req.Protocol = protocol
		dump := view.Dump(a.messagesPane(wsPanes[0], tab, nil))
		if formats := strings.Contains(dump, "{ }"); formats != (protocol == "websocket") {
			t.Errorf("%s offers { }: %v", protocol, formats)
		}
	}
}

// A session that arrives for a tab that is not connecting, or for a connect
// that was replaced, is closed, not kept.
func TestASessionNobodyWaitsForIsClosed(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	stale, refused := &fakeSession{}, &fakeSession{}
	gen, _ := tab.Begin()
	tab.Begin() // supersedes the first connect
	a.handle(wsOpened{wsTarget{tab: tab, gen: gen}, stale})
	openSession(tab, &fakeSession{})
	a.handle(wsOpened{wsTarget{tab: tab, gen: tab.Gen}, refused})
	if !stale.closed || !refused.closed {
		t.Fatalf("stale closed %v, refused closed %v", stale.closed, refused.closed)
	}
}
