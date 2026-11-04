// Package ws is a small RFC 6455 WebSocket client (plus the server side
// of the handshake, which the tests use). It is hand-written on the
// standard library: the workshop keeps its dependencies to the toolkit.
package ws

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Type is a frame kind.
type Type byte

// Message kinds. Text and Binary carry data; the rest are control frames.
const (
	opContinuation Type = 0
	Text           Type = 1
	Binary         Type = 2
	Close          Type = 8
	Ping           Type = 9
	Pong           Type = 10
)

func (t Type) String() string {
	switch t {
	case Text:
		return "text"
	case Binary:
		return "binary"
	case Close:
		return "close"
	case Ping:
		return "ping"
	case Pong:
		return "pong"
	}
	return fmt.Sprintf("type(%d)", byte(t))
}

// Close status codes (RFC 6455 section 7.4).
const (
	CloseNormal      = 1000
	CloseGoingAway   = 1001
	CloseProtocol    = 1002
	CloseUnsupported = 1003
	CloseNoStatus    = 1005
	CloseAbnormal    = 1006
	CloseTooBig      = 1009
)

// DefaultMaxMessage bounds one message when Options.MaxMessage is unset.
const DefaultMaxMessage = 16 << 20

const handshakeGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Message is one received message. For Close, Code is the status code
// (CloseNoStatus when the peer sent none) and Data the reason.
type Message struct {
	Type Type
	Data []byte
	Code int
}

// Options tune Dial.
type Options struct {
	Header       http.Header // extra handshake headers
	Subprotocols []string    // offered, in order of preference
	MaxMessage   int         // largest accepted message; 0 for DefaultMaxMessage
	TLS          *tls.Config // for wss; nil for the defaults
}

// Conn is one WebSocket connection. Read must be called from one
// goroutine at a time; Write and Close may be called from any.
type Conn struct {
	nc     net.Conn
	br     *bufio.Reader
	client bool
	sub    string
	max    int

	wmu       sync.Mutex
	closeSent bool
	rmu       sync.Mutex
	gotClose  bool
}

// Subprotocol is the subprotocol the server chose, "" for none.
func (c *Conn) Subprotocol() string { return c.sub }

// SetReadDeadline limits the next Read; the zero time clears it.
func (c *Conn) SetReadDeadline(t time.Time) error { return c.nc.SetReadDeadline(t) }

// Dial opens a connection to a ws:// or wss:// URL.
func Dial(ctx context.Context, rawURL string, o Options) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("websocket: %w", err)
	}
	var secure bool
	switch u.Scheme {
	case "ws":
	case "wss":
		secure = true
	default:
		return nil, fmt.Errorf("websocket: URL scheme %q is not ws or wss", u.Scheme)
	}
	host := u.Host
	if u.Port() == "" {
		port := "80"
		if secure {
			port = "443"
		}
		host = net.JoinHostPort(u.Hostname(), port)
	}
	var d net.Dialer
	nc, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("websocket: %w", err)
	}
	if secure {
		cfg := o.TLS
		if cfg == nil {
			cfg = &tls.Config{}
		}
		cfg = cfg.Clone()
		if cfg.ServerName == "" {
			cfg.ServerName = u.Hostname()
		}
		tc := tls.Client(nc, cfg)
		if err := tc.HandshakeContext(ctx); err != nil {
			nc.Close()
			return nil, fmt.Errorf("websocket: %w", err)
		}
		nc = tc
	}
	stop := context.AfterFunc(ctx, func() { nc.Close() })
	c, err := clientHandshake(nc, u, o)
	if !stop() && err == nil { // the context fired while handshaking
		err = ctx.Err()
	}
	if err != nil {
		nc.Close()
		return nil, err
	}
	return c, nil
}

func clientHandshake(nc net.Conn, u *url.URL, o Options) (*Conn, error) {
	keyBytes := make([]byte, 16)
	rand.Read(keyBytes)
	key := base64.StdEncoding.EncodeToString(keyBytes)
	req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: u.Path, RawQuery: u.RawQuery}, Host: u.Host, Header: http.Header{}}
	for k, vs := range o.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", key)
	req.Header.Set("Sec-WebSocket-Version", "13")
	if len(o.Subprotocols) > 0 {
		req.Header.Set("Sec-WebSocket-Protocol", strings.Join(o.Subprotocols, ", "))
	}
	if err := req.Write(nc); err != nil {
		return nil, fmt.Errorf("websocket: %w", err)
	}
	br := bufio.NewReader(nc)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		return nil, fmt.Errorf("websocket handshake: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("websocket handshake: %s", resp.Status)
	}
	if resp.Header.Get("Sec-WebSocket-Accept") != acceptKey(key) {
		return nil, errors.New("websocket handshake: bad Sec-WebSocket-Accept")
	}
	c := &Conn{nc: nc, br: br, client: true, sub: resp.Header.Get("Sec-WebSocket-Protocol"), max: o.MaxMessage}
	if c.max <= 0 {
		c.max = DefaultMaxMessage
	}
	return c, nil
}

func acceptKey(key string) string {
	sum := sha1.Sum([]byte(key + handshakeGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// Upgrade answers a client's handshake on the server side and returns
// the connection. The first offered subprotocol is chosen.
func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || r.Header.Get("Sec-WebSocket-Key") == "" {
		http.Error(w, "not a websocket handshake", http.StatusBadRequest)
		return nil, errors.New("websocket: not a handshake")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "cannot hijack", http.StatusInternalServerError)
		return nil, errors.New("websocket: no hijacker")
	}
	nc, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	sub := ""
	if offered := r.Header.Get("Sec-WebSocket-Protocol"); offered != "" {
		sub = strings.TrimSpace(strings.Split(offered, ",")[0])
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + acceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n"
	if sub != "" {
		resp += "Sec-WebSocket-Protocol: " + sub + "\r\n"
	}
	if _, err := rw.WriteString(resp + "\r\n"); err != nil {
		nc.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		nc.Close()
		return nil, err
	}
	return &Conn{nc: nc, br: rw.Reader, sub: sub, max: DefaultMaxMessage}, nil
}

// Write sends one data or control message. Use Close to end the
// connection properly.
func (c *Conn) Write(t Type, data []byte) error {
	switch t {
	case Text, Binary:
	case Ping, Pong:
		if len(data) > 125 {
			return errors.New("websocket: control message over 125 bytes")
		}
	default:
		return fmt.Errorf("websocket: cannot write a %s message", t)
	}
	return c.writeFrame(true, t, data)
}

func (c *Conn) writeFrame(fin bool, op Type, data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if c.closeSent {
		return net.ErrClosed
	}
	return c.writeLocked(fin, op, data)
}

func (c *Conn) writeLocked(fin bool, op Type, data []byte) error {
	head := make([]byte, 2, 14)
	head[0] = byte(op)
	if fin {
		head[0] |= 0x80
	}
	n := len(data)
	switch {
	case n < 126:
		head[1] = byte(n)
	case n <= 0xffff:
		head[1] = 126
		head = binary.BigEndian.AppendUint16(head, uint16(n))
	default:
		head[1] = 127
		head = binary.BigEndian.AppendUint64(head, uint64(n))
	}
	payload := data
	if c.client {
		head[1] |= 0x80
		var key [4]byte
		rand.Read(key[:])
		head = append(head, key[:]...)
		payload = make([]byte, n)
		for i, b := range data {
			payload[i] = b ^ key[i%4]
		}
	}
	_, err := c.nc.Write(append(head, payload...))
	return err
}

// Close sends a close frame with code and reason, then drops the
// connection without waiting for the peer's answer. Calling it again,
// or after the peer closed, is harmless.
func (c *Conn) Close(code int, reason string) error {
	c.wmu.Lock()
	if !c.closeSent {
		c.closeSent = true
		body := binary.BigEndian.AppendUint16(nil, uint16(code))
		if len(reason) > 123 {
			reason = reason[:123]
		}
		c.writeLocked(true, Close, append(body, reason...))
	}
	c.wmu.Unlock()
	return c.nc.Close()
}

// Read returns the next message. Pings are answered with a pong and
// still returned; a close frame is answered once and returned, and every
// Read after it fails with io.EOF.
func (c *Conn) Read() (Message, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	if c.gotClose {
		return Message{}, io.EOF
	}
	var (
		buf     []byte
		msgType Type
		started bool
	)
	for {
		fin, op, payload, err := c.readFrame(len(buf))
		if err != nil {
			return Message{}, err
		}
		switch op {
		case Ping:
			c.writeFrame(true, Pong, payload)
			return Message{Type: Ping, Data: payload}, nil
		case Pong:
			return Message{Type: Pong, Data: payload}, nil
		case Close:
			c.gotClose = true
			code, reason := CloseNoStatus, []byte(nil)
			if len(payload) >= 2 {
				code, reason = int(binary.BigEndian.Uint16(payload)), payload[2:]
			}
			c.Close(code, "") // echo the status; ends the connection
			return Message{Type: Close, Code: code, Data: reason}, nil
		case Text, Binary:
			if started {
				return Message{}, errors.New("websocket: new message inside a fragmented one")
			}
			started, msgType = true, op
		case opContinuation:
			if !started {
				return Message{}, errors.New("websocket: unexpected continuation frame")
			}
		default:
			return Message{}, fmt.Errorf("websocket: unknown opcode %d", byte(op))
		}
		buf = append(buf, payload...)
		if fin {
			return Message{Type: msgType, Data: buf}, nil
		}
	}
}

// readFrame reads one frame; have is the size of the message so far,
// for the size limit.
func (c *Conn) readFrame(have int) (fin bool, op Type, payload []byte, err error) {
	var h [2]byte
	if _, err = io.ReadFull(c.br, h[:]); err != nil {
		return
	}
	fin, op = h[0]&0x80 != 0, Type(h[0]&0x0f)
	if h[0]&0x70 != 0 {
		err = errors.New("websocket: reserved bits set")
		return
	}
	masked := h[1]&0x80 != 0
	if masked == c.client {
		err = errors.New("websocket: wrong masking from the peer")
		return
	}
	n := uint64(h[1] & 0x7f)
	switch n {
	case 126:
		var b [2]byte
		if _, err = io.ReadFull(c.br, b[:]); err != nil {
			return
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err = io.ReadFull(c.br, b[:]); err != nil {
			return
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	if op >= Close && (n > 125 || !fin) {
		err = errors.New("websocket: bad control frame")
		return
	}
	if n > uint64(c.max) || have+int(n) > c.max {
		err = fmt.Errorf("websocket: message too large (over %d bytes)", c.max)
		return
	}
	var key [4]byte
	if masked {
		if _, err = io.ReadFull(c.br, key[:]); err != nil {
			return
		}
	}
	payload = make([]byte, n)
	if _, err = io.ReadFull(c.br, payload); err != nil {
		return
	}
	if masked {
		for i := range payload {
			payload[i] ^= key[i%4]
		}
	}
	return
}
