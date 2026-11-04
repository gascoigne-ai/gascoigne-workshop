package ws

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// echoServer upgrades every request and answers each data message with
// the same message until the client closes.
func echoServer(t *testing.T, handle func(c *Conn)) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Upgrade(w, r)
		if err != nil {
			return
		}
		defer c.Close(CloseNormal, "")
		handle(c)
	}))
	t.Cleanup(srv.Close)
	return srv, "ws" + strings.TrimPrefix(srv.URL, "http")
}

func echo(c *Conn) {
	for {
		m, err := c.Read()
		if err != nil || m.Type == Close {
			return
		}
		if err := c.Write(m.Type, m.Data); err != nil {
			return
		}
	}
}

func dial(t *testing.T, url string, o Options) *Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := Dial(ctx, url, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(CloseNormal, "") })
	return c
}

func TestEchoTextAndBinary(t *testing.T) {
	_, url := echoServer(t, echo)
	c := dial(t, url, Options{})
	for _, tt := range []struct {
		typ  Type
		data []byte
	}{
		{Text, []byte("hello")},
		{Binary, []byte{0, 1, 2, 255}},
		{Text, nil},
		{Text, bytes.Repeat([]byte("x"), 200)},     // 16-bit length
		{Binary, bytes.Repeat([]byte("y"), 70000)}, // 64-bit length
	} {
		if err := c.Write(tt.typ, tt.data); err != nil {
			t.Fatal(err)
		}
		m, err := c.Read()
		if err != nil || m.Type != tt.typ || !bytes.Equal(m.Data, tt.data) {
			t.Fatalf("len %d: got %v %d bytes, %v", len(tt.data), m.Type, len(m.Data), err)
		}
	}
}

func TestSubprotocolAndHeadersReachTheServer(t *testing.T) {
	got := make(chan http.Header, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.Header
		c, err := Upgrade(w, r)
		if err == nil {
			c.Close(CloseNormal, "")
		}
	}))
	defer srv.Close()
	c := dial(t, "ws"+strings.TrimPrefix(srv.URL, "http"), Options{
		Header:       http.Header{"Authorization": {"Bearer t"}},
		Subprotocols: []string{"chat", "v2"},
	})
	h := <-got
	if h.Get("Authorization") != "Bearer t" || h.Get("Sec-WebSocket-Protocol") != "chat, v2" {
		t.Fatalf("headers %v", h)
	}
	if c.Subprotocol() != "chat" {
		t.Fatalf("negotiated %q", c.Subprotocol())
	}
}

func TestServerCloseCarriesCodeAndReason(t *testing.T) {
	_, url := echoServer(t, func(c *Conn) { c.Close(CloseGoingAway, "bye") })
	c := dial(t, url, Options{})
	m, err := c.Read()
	if err != nil || m.Type != Close || m.Code != CloseGoingAway || string(m.Data) != "bye" {
		t.Fatalf("got %+v, %v", m, err)
	}
	if _, err := c.Read(); err == nil {
		t.Fatal("reading after close must fail")
	}
}

func TestPingIsAnsweredWithPong(t *testing.T) {
	pong := make(chan []byte, 1)
	_, url := echoServer(t, func(c *Conn) {
		c.Write(Ping, []byte("are you there"))
		m, err := c.Read()
		if err == nil && m.Type == Pong {
			pong <- m.Data
		}
	})
	c := dial(t, url, Options{})
	m, err := c.Read() // the client sees the ping and answers it
	if err != nil || m.Type != Ping {
		t.Fatalf("got %+v, %v", m, err)
	}
	select {
	case d := <-pong:
		if string(d) != "are you there" {
			t.Fatalf("pong %q", d)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no pong")
	}
}

func TestFragmentedMessagesAreJoined(t *testing.T) {
	_, url := echoServer(t, func(c *Conn) {
		c.writeFrame(false, Text, []byte("hel"))
		c.writeFrame(false, opContinuation, []byte("lo "))
		c.writeFrame(true, opContinuation, []byte("world"))
	})
	c := dial(t, url, Options{})
	m, err := c.Read()
	if err != nil || string(m.Data) != "hello world" {
		t.Fatalf("got %q, %v", m.Data, err)
	}
}

func TestHandshakeFailuresAreReported(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer plain.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Dial(ctx, "ws"+strings.TrimPrefix(plain.URL, "http"), Options{})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("got %v", err)
	}
	if _, err := Dial(ctx, "http://example.org", Options{}); err == nil {
		t.Fatal("a non-ws scheme must be refused")
	}
	if _, err := Dial(ctx, "ws://127.0.0.1:1", Options{}); err == nil {
		t.Fatal("a refused connection must fail")
	}
}

func TestOversizedMessageIsRefused(t *testing.T) {
	_, url := echoServer(t, func(c *Conn) { c.Write(Binary, make([]byte, 5000)) })
	c := dial(t, url, Options{MaxMessage: 1000})
	if _, err := c.Read(); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("got %v", err)
	}
}

func TestClosingTheClientEndsTheServerLoop(t *testing.T) {
	done := make(chan struct{})
	_, url := echoServer(t, func(c *Conn) { echo(c); close(done) })
	c := dial(t, url, Options{})
	c.Close(CloseNormal, "done")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server never saw the close")
	}
	if err := c.Write(Text, []byte("x")); err == nil {
		t.Fatal("writing after close must fail")
	}
}

func TestReadStopsWhenTheContextOfDialIsNotInvolved(t *testing.T) {
	_, url := echoServer(t, func(c *Conn) { time.Sleep(200 * time.Millisecond) })
	c := dial(t, url, Options{})
	c.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
	if _, err := c.Read(); err == nil || err == io.EOF {
		t.Fatalf("want a timeout, got %v", err)
	}
}
