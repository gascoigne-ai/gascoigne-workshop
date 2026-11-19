package ws

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// echoServer upgrades every request and answers each data message with
// the same message until the client closes.
func echoServer(t *testing.T, handle func(conn *Conn)) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		c, err := Upgrade(writer, request)
		if err != nil {
			return
		}
		defer c.Close(CloseNormal, "")
		handle(c)
	}))
	t.Cleanup(srv.Close)
	return srv, "ws" + strings.TrimPrefix(srv.URL, "http")
}

func echo(conn *Conn) {
	for {
		m, err := conn.Read()
		if err != nil || m.Type == Close {
			return
		}
		if err := conn.Write(m.Type, m.Data); err != nil {
			return
		}
	}
}

func dial(t *testing.T, url string, options Options) *Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := Dial(ctx, url, options)
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
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		got <- request.Header
		c, err := Upgrade(writer, request)
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
	_, url := echoServer(t, func(conn *Conn) { conn.Close(CloseGoingAway, "bye") })
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
	_, url := echoServer(t, func(conn *Conn) {
		conn.Write(Ping, []byte("are you there"))
		m, err := conn.Read()
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
	_, url := echoServer(t, func(conn *Conn) {
		conn.writeFrame(false, Text, []byte("hel"))
		conn.writeFrame(false, opContinuation, []byte("lo "))
		conn.writeFrame(true, opContinuation, []byte("world"))
	})
	c := dial(t, url, Options{})
	m, err := c.Read()
	if err != nil || string(m.Data) != "hello world" {
		t.Fatalf("got %q, %v", m.Data, err)
	}
}

func TestHandshakeFailuresAreReported(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "nope", http.StatusForbidden)
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

func TestOversizedHandshakeResponseIsRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		bufio.NewReader(conn).ReadString('\n')
		conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n"))
		line := "X-Filler: " + strings.Repeat("a", 1000) + "\r\n"
		for range 300 { // 300 KB of headers, far over the bound
			if _, err := conn.Write([]byte(line)); err != nil {
				return
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = Dial(ctx, "ws://"+listener.Addr().String(), Options{})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("got %v", err)
	}
}

func TestOversizedMessageIsRefused(t *testing.T) {
	_, url := echoServer(t, func(conn *Conn) { conn.Write(Binary, make([]byte, 5000)) })
	c := dial(t, url, Options{MaxMessage: 1000})
	if _, err := c.Read(); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("got %v", err)
	}
}

func TestClosingTheClientEndsTheServerLoop(t *testing.T) {
	done := make(chan struct{})
	_, url := echoServer(t, func(conn *Conn) { echo(conn); close(done) })
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

// A control frame between the fragments of a message (RFC 6455 allows
// it) made Read return the ping and drop the fragments read so far, so
// the next continuation frame failed as "unexpected continuation".
func TestAPingBetweenFragmentsKeepsTheMessage(t *testing.T) {
	_, url := echoServer(t, func(conn *Conn) {
		conn.writeFrame(false, Text, []byte("hel"))
		conn.writeFrame(true, Ping, []byte("p"))
		conn.writeFrame(true, opContinuation, []byte("lo"))
	})
	c := dial(t, url, Options{})
	m, err := c.Read()
	if err != nil || m.Type != Ping {
		t.Fatalf("got %+v, %v; want the ping first", m, err)
	}
	m, err = c.Read()
	if err != nil || m.Type != Text || string(m.Data) != "hello" {
		t.Fatalf("got %+v, %v; want the joined text", m, err)
	}
}

// FuzzReadFrame: any bytes a peer could send, read as a client and as a
// server, through the frame reader and the message reader above it.
func FuzzReadFrame(fuzz *testing.F) {
	for _, seed := range [][]byte{
		{0x81, 0x02, 'h', 'i'}, {0x01, 0x03, 'h', 'e', 'l', 0x80, 0x02, 'l', 'o'}, {0x89, 0x00}, {0x88, 0x02, 0x03, 0xe8},
		{0x82, 0x7e, 0x00, 0x05, 1, 2, 3, 4, 5}, {0x82, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, {0xf1, 0x00}, {},
	} {
		fuzz.Add(seed, true)
	}
	fuzz.Fuzz(func(t *testing.T, data []byte, client bool) {
		conn := &Conn{br: bufio.NewReader(bytes.NewReader(data)), nc: discardConn{}, client: client, max: 1 << 16}
		for range 8 {
			if _, err := conn.Read(); err != nil {
				return
			}
		}
	})
}

// discardConn takes the pongs and close frames Read answers with.
type discardConn struct{ net.Conn }

func (discardConn) Write(data []byte) (int, error) { return len(data), nil }
func (discardConn) Close() error                   { return nil }
