package proto

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// SSE reads Server-Sent Events: a long GET whose answer is a stream of
// events. It only receives; Send on its session is an error.
type SSE struct {
	// Client is used when set.
	Client *http.Client
}

// sseEvent is one event of the stream, by the fields of the format.
type sseEvent struct {
	name, data, id string
}

// readEvents parses the event-stream format from r until it ends and
// calls emit for each event that has data. Comment lines (":") and the
// retry field are ignored.
func readEvents(sc *bufio.Scanner, emit func(sseEvent)) {
	var ev sseEvent
	var data []string
	flush := func() {
		if len(data) > 0 {
			ev.data = strings.Join(data, "\n")
			emit(ev)
		}
		ev, data = sseEvent{id: ev.id}, nil // the id sticks until a new one is sent
	}
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			flush()
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch {
		case strings.HasPrefix(line, ":"):
		case field == "event":
			ev.name = value
		case field == "data":
			data = append(data, value)
		case field == "id":
			ev.id = value
		}
	}
	flush()
}

// Connect implements Streamer.
func (s SSE) Connect(ctx context.Context, req model.Request) (Session, error) {
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		return nil, fmt.Errorf("%q is not an http or https URL", req.URL)
	}
	cctx, cancel := context.WithCancel(context.Background())
	httpReq, err := http.NewRequestWithContext(cctx, http.MethodGet, req.URL, nil)
	if err != nil {
		cancel()
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	for _, h := range req.Headers {
		if h.Enabled && h.Name != "" {
			httpReq.Header.Set(h.Name, h.Value)
		}
	}
	if apply, ok := authAppliers[req.Auth.Type]; ok {
		apply(req.Auth, httpReq)
	}
	stop := context.AfterFunc(ctx, cancel) // the connect timeout ends only the connect
	client := s.Client
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(httpReq)
	stop()
	if err != nil {
		cancel()
		return nil, errors.New(failureText(err))
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		return nil, fmt.Errorf("%s", resp.Status)
	}
	se := &sseSession{events: make(chan Event, eventBuffer), done: make(chan struct{}), cancel: cancel}
	se.emit(Event{Dir: Info, Kind: "open", Text: "Connected to " + req.URL})
	go func() {
		defer func() {
			resp.Body.Close()
			se.finish()
		}()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 64<<10), 16<<20)
		readEvents(sc, func(ev sseEvent) {
			kind := ev.name
			if kind == "" {
				kind = "message"
			}
			text := ev.data
			if ev.id != "" {
				text = fmt.Sprintf("[%s] %s", ev.id, ev.data)
			}
			se.emit(Event{Dir: Received, Kind: kind, Text: text, Size: len(ev.data), ID: ev.id})
		})
		if err := sc.Err(); err != nil && !se.isClosing() {
			se.emit(Event{Dir: Info, Kind: "close", Code: 1006, Text: "Connection lost: " + err.Error()})
		} else if !se.isClosing() {
			se.emit(Event{Dir: Info, Kind: "close", Text: "The server ended the stream"})
		}
	}()
	return se, nil
}

// Send implements Protocol as the one-shot form: connect and return the
// first event.
func (s SSE) Send(ctx context.Context, req model.Request) (model.Response, error) {
	start := time.Now()
	sess, err := s.Connect(ctx, req)
	if err != nil {
		return model.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
	}
	defer sess.Close()
	for {
		select {
		case e, ok := <-sess.Events():
			if !ok {
				return model.Response{Err: "the stream ended before an event", Duration: time.Since(start)}, nil
			}
			if e.Dir == Received {
				return model.Response{Status: http.StatusOK, StatusText: "OK", Duration: time.Since(start), Size: e.Size, Body: []byte(e.Text)}, nil
			}
		case <-ctx.Done():
			return model.Response{Err: stopReason(ctx, ctx.Err()), Duration: time.Since(start)}, nil
		}
	}
}

type sseSession struct {
	events chan Event
	done   chan struct{}
	cancel context.CancelFunc

	mu      sync.Mutex
	ended   bool
	closing bool
}

func (s *sseSession) Events() <-chan Event { return s.events }

func (s *sseSession) isClosing() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.closing }

func (s *sseSession) emit(e Event) {
	e.Time = time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return
	}
	select {
	case s.events <- e:
	case <-s.done:
	}
}

func (s *sseSession) finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ended {
		s.ended = true
		close(s.events)
	}
}

func (s *sseSession) Send(string) error { return errors.New("an event stream only receives") }
func (s *sseSession) Ping() error       { return errors.New("an event stream has no ping") }

func (s *sseSession) Close() error {
	s.mu.Lock()
	s.closing = true
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	s.mu.Unlock()
	s.cancel()
	return nil
}
