package proto

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// eventBuffer is how many events a session queues before the reader
// waits for the consumer.
const eventBuffer = 256

// WebSocket talks WebSocket with the ws package.
type WebSocket struct {
	// Network supplies the TLS setting; Jar and Proxy do not apply.
	Network Network
}

// handshake builds the dial options and the URL to dial from the request:
// enabled headers, the credentials of the Auth pane (an API key in the query
// lands in the URL), the subprotocols and the TLS setting. Proxy is not
// applied: ws.Dial has no proxy support.
func (w WebSocket) handshake(ctx context.Context, req model.Request) (ws.Options, string, error) {
	built, err := newHTTPRequest(ctx, http.MethodGet, req, nil, "")
	if err != nil {
		return ws.Options{}, "", err
	}
	var o ws.Options
	for _, value := range built.Header.Values("Sec-WebSocket-Protocol") {
		for _, p := range strings.Split(value, ",") {
			if p = strings.TrimSpace(p); p != "" {
				o.Subprotocols = append(o.Subprotocols, p)
			}
		}
	}
	built.Header.Del("Sec-WebSocket-Protocol") // ws.Dial sends the subprotocols itself
	o.Header = built.Header
	o.TLS = &tls.Config{}
	if err := applyTLS(o.TLS, w.Network, req); err != nil {
		return ws.Options{}, "", err
	}
	return o, built.URL.String(), nil
}

// Connect implements Streamer.
func (w WebSocket) Connect(ctx context.Context, req model.Request) (Session, error) {
	o, target, err := w.handshake(ctx, req)
	if err != nil {
		return nil, err
	}
	conn, err := ws.Dial(ctx, target, o)
	if err != nil {
		return nil, err
	}
	s := &wsSession{eventLog: newEventLog(), conn: conn}
	open := "Connected to " + req.URL
	if p := conn.Subprotocol(); p != "" {
		open += " (" + p + ")"
	}
	s.emit(Event{Dir: Info, Kind: "open", Text: open})
	go s.readLoop()
	return s, nil
}

// Send implements Protocol as the one-shot form.
func (w WebSocket) Send(ctx context.Context, req model.Request) (model.Response, error) {
	start := time.Now()
	o, target, err := w.handshake(ctx, req)
	if err != nil {
		return model.Response{}, err
	}
	conn, err := ws.Dial(ctx, target, o)
	if err != nil {
		return model.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
	}
	stop := context.AfterFunc(ctx, func() { conn.Close(ws.CloseGoingAway, "") })
	defer stop()
	defer conn.Close(ws.CloseNormal, "")
	resp := model.Response{Status: http.StatusSwitchingProtocols, StatusText: "Switching Protocols"}
	if p := conn.Subprotocol(); p != "" {
		resp.Headers = []model.KV{{Name: "Sec-WebSocket-Protocol", Value: p, Enabled: true}}
	}
	if req.Body != "" {
		if err := conn.Write(ws.Text, []byte(req.Body)); err != nil {
			return model.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
		}
	}
	for {
		m, err := conn.Read()
		if err != nil {
			resp.Err = stopReason(ctx, err)
			break
		}
		if m.Type == ws.Text || m.Type == ws.Binary {
			resp.Body, resp.Size = m.Data, len(m.Data)
			break
		}
		if m.Type == ws.Close {
			resp.Err = fmt.Sprintf("Closed by server: %d %s", m.Code, m.Data)
			break
		}
	}
	resp.Duration = time.Since(start)
	return resp, nil
}

// stopReason words a failure like REST does for cancelled and timed-out
// requests.
func stopReason(ctx context.Context, err error) string {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return ErrTimedOut
	case errors.Is(ctx.Err(), context.Canceled):
		return ErrCancelled
	}
	return err.Error()
}

type wsSession struct {
	*eventLog
	conn *ws.Conn
}

func (s *wsSession) readLoop() {
	defer s.finish()
	for {
		m, err := s.conn.Read()
		if err != nil {
			if !s.closing() { // closed by us: no loss
				s.emit(Event{Dir: Info, Kind: "close", Code: ws.CloseAbnormal, Text: "Connection lost: " + err.Error()})
			}
			return
		}
		switch m.Type {
		case ws.Text, ws.Binary:
			s.emit(Event{Dir: Received, Kind: m.Type.String(), Text: render(m), Size: len(m.Data)})
		case ws.Ping, ws.Pong:
			s.emit(Event{Dir: Received, Kind: m.Type.String(), Text: string(m.Data), Size: len(m.Data)})
		case ws.Close:
			text := fmt.Sprintf("Closed by server: %d", m.Code)
			if len(m.Data) > 0 {
				text += " " + string(m.Data)
			}
			s.emit(Event{Dir: Info, Kind: "close", Code: m.Code, Text: text})
			return
		}
	}
}

// render shows a message as text; binary data as hex, cut at 64 bytes.
func render(message ws.Message) string {
	if message.Type == ws.Text || utf8.Valid(message.Data) {
		return string(message.Data)
	}
	d := message.Data
	suffix := ""
	if len(d) > 64 {
		d, suffix = d[:64], "…"
	}
	return hex.EncodeToString(d) + suffix
}

// alive reports why a write cannot happen: its context ended or the
// connection is closed.
func (s *wsSession) alive(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.ended.Load() {
		return errors.New("the connection is closed")
	}
	return nil
}

func (s *wsSession) Send(ctx context.Context, text string) error {
	if err := s.alive(ctx); err != nil {
		return err
	}
	// Logged before the write: an echo can arrive as soon as the bytes are
	// out, and the read loop would log it ahead of the message it answers.
	s.emit(Event{Dir: Sent, Kind: "text", Text: text, Size: len(text)})
	return s.conn.Write(ws.Text, []byte(text))
}

// SendBinary implements BinarySender. The log shows the bytes as hex.
func (s *wsSession) SendBinary(ctx context.Context, data []byte) error {
	if err := s.alive(ctx); err != nil {
		return err
	}
	s.emit(Event{Dir: Sent, Kind: "binary", Text: hex.EncodeToString(data), Size: len(data)}) // before the write, as in Send
	return s.conn.Write(ws.Binary, data)
}

func (s *wsSession) Ping(ctx context.Context) error {
	if err := s.alive(ctx); err != nil {
		return err
	}
	if err := s.conn.Write(ws.Ping, nil); err != nil {
		return err
	}
	s.emit(Event{Dir: Sent, Kind: "ping"})
	return nil
}

func (s *wsSession) Close(context.Context) error {
	s.close()
	return s.conn.Close(ws.CloseNormal, "")
}
