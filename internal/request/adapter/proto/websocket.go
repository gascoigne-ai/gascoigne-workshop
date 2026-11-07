package proto

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// eventBuffer is how many events a session queues before the reader
// waits for the consumer.
const eventBuffer = 256

// WebSocket talks WebSocket with the ws package.
type WebSocket struct{}

// handshake builds the dial options from the request: enabled headers,
// the Auth pane's credentials, and the subprotocols the user listed in a
// Sec-WebSocket-Protocol header.
func (WebSocket) handshake(req model.Request) (ws.Options, error) {
	probe, err := http.NewRequest(http.MethodGet, "http://probe.invalid/", nil)
	if err != nil {
		return ws.Options{}, err
	}
	var o ws.Options
	for _, h := range req.Headers {
		if !h.Enabled || h.Name == "" {
			continue
		}
		if strings.EqualFold(h.Name, "Sec-WebSocket-Protocol") {
			for _, p := range strings.Split(h.Value, ",") {
				if p = strings.TrimSpace(p); p != "" {
					o.Subprotocols = append(o.Subprotocols, p)
				}
			}
			continue
		}
		probe.Header.Add(h.Name, h.Value)
	}
	apply, ok := authAppliers[req.Auth.Type]
	if !ok {
		return ws.Options{}, fmt.Errorf("unknown auth type %q", req.Auth.Type)
	}
	apply(req.Auth, probe)
	o.Header = probe.Header
	return o, nil
}

// Connect implements Streamer.
func (w WebSocket) Connect(ctx context.Context, req model.Request) (Session, error) {
	o, err := w.handshake(req)
	if err != nil {
		return nil, err
	}
	conn, err := ws.Dial(ctx, req.URL, o)
	if err != nil {
		return nil, err
	}
	s := &wsSession{conn: conn, events: make(chan Event, eventBuffer), done: make(chan struct{}), now: time.Now}
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
	o, err := w.handshake(req)
	if err != nil {
		return model.Response{}, err
	}
	conn, err := ws.Dial(ctx, req.URL, o)
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
	conn   *ws.Conn
	events chan Event
	done   chan struct{} // closed by Close
	once   sync.Once
	now    func() time.Time

	mu    sync.Mutex
	ended bool // the read loop has finished
}

func (s *wsSession) Events() <-chan Event { return s.events }

// emit queues e, waiting for a slow consumer, unless the session was closed.
func (s *wsSession) emit(e Event) {
	e.Time = s.now()
	select {
	case s.events <- e:
	case <-s.done:
	}
}

func (s *wsSession) readLoop() {
	defer func() {
		s.mu.Lock()
		s.ended = true
		s.mu.Unlock()
		close(s.events)
	}()
	for {
		m, err := s.conn.Read()
		if err != nil {
			select {
			case <-s.done: // we closed it ourselves
			default:
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
func render(m ws.Message) string {
	if m.Type == ws.Text || utf8.Valid(m.Data) {
		return string(m.Data)
	}
	d := m.Data
	suffix := ""
	if len(d) > 64 {
		d, suffix = d[:64], "…"
	}
	return hex.EncodeToString(d) + suffix
}

func (s *wsSession) alive() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return errors.New("the connection is closed")
	}
	return nil
}

func (s *wsSession) Send(text string) error {
	if err := s.alive(); err != nil {
		return err
	}
	if err := s.conn.Write(ws.Text, []byte(text)); err != nil {
		return err
	}
	s.emit(Event{Dir: Sent, Kind: "text", Text: text, Size: len(text)})
	return nil
}

// SendBinary implements BinarySender. The log shows the bytes as hex.
func (s *wsSession) SendBinary(data []byte) error {
	if err := s.alive(); err != nil {
		return err
	}
	if err := s.conn.Write(ws.Binary, data); err != nil {
		return err
	}
	s.emit(Event{Dir: Sent, Kind: "binary", Text: hex.EncodeToString(data), Size: len(data)})
	return nil
}

func (s *wsSession) Ping() error {
	if err := s.alive(); err != nil {
		return err
	}
	if err := s.conn.Write(ws.Ping, nil); err != nil {
		return err
	}
	s.emit(Event{Dir: Sent, Kind: "ping"})
	return nil
}

func (s *wsSession) Close() error {
	s.once.Do(func() { close(s.done) })
	return s.conn.Close(ws.CloseNormal, "")
}
