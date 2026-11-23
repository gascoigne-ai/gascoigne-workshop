package protocol

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol/ws"
)

// SSE reads Server-Sent Events: a long GET whose answer is a stream of
// events. It only receives; Send on its session is an error.
type SSE struct {
	// Client is used when set.
	Client *http.Client

	// Network is the cookie jar, proxy and TLS setting of the connection.
	Network Network
}

// sseEvent is one event of the stream, by the fields of the format.
type sseEvent struct {
	name, data, id string
}

// Connect implements Streamer.
func (s SSE) Connect(ctx context.Context, req request.Request) (request.Session, error) {
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		return nil, fmt.Errorf("%q is not an http or https URL", req.URL)
	}
	cctx, cancel := context.WithCancel(context.Background())
	httpReq, err := newHTTPRequest(cctx, http.MethodGet, req, nil, "")
	if err != nil {
		cancel()
		return nil, err
	}
	for name, value := range map[string]string{"Accept": "text/event-stream", "Cache-Control": "no-cache"} {
		if httpReq.Header.Get(name) == "" { // a header typed by the user wins
			httpReq.Header.Set(name, value)
		}
	}
	client, err := newHTTPClient(s.Client, s.Network, req)
	if err != nil {
		cancel()
		return nil, err
	}
	stop := context.AfterFunc(ctx, cancel) // the connect timeout ends only the connect
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
	se := &sseSession{eventLog: newEventLog(), cancel: cancel}
	se.emit(request.Event{Dir: request.Info, Kind: request.KindOpen, Text: "Connected to " + req.URL})
	go func() {
		defer func() {
			resp.Body.Close()
			se.finish()
		}()
		err := readEventStream(resp.Body, func(ev sseEvent) {
			text := ev.data
			if ev.id != "" {
				text = fmt.Sprintf("[%s] %s", ev.id, ev.data)
			}
			se.emit(request.Event{Dir: request.Received, Kind: request.KindMessage, Name: ev.name, Text: text, Size: len(ev.data), ID: ev.id})
		})
		if err != nil && !se.closing() {
			se.emit(request.Event{Dir: request.Info, Kind: request.KindClose, Code: ws.CloseAbnormal, Text: "Connection lost: " + err.Error()})
		} else if !se.closing() {
			se.emit(request.Event{Dir: request.Info, Kind: request.KindClose, Text: "The server ended the stream"})
		}
	}()
	return se, nil
}

// Send implements Protocol as the one-shot form: connect and return the
// first event.
func (s SSE) Send(ctx context.Context, req request.Request) (request.Response, error) {
	start := time.Now()
	sess, err := s.Connect(ctx, req)
	if err != nil {
		return request.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
	}
	defer sess.Close(context.WithoutCancel(ctx))
	for {
		select {
		case e, ok := <-sess.Events():
			if !ok {
				return request.Response{Err: "the stream ended before an event", Duration: time.Since(start)}, nil
			}
			if e.Dir == request.Received {
				return request.Response{Status: http.StatusOK, StatusText: "OK", Duration: time.Since(start), Size: e.Size, Body: []byte(e.Text)}, nil
			}
		case <-ctx.Done():
			return request.Response{Err: stopReason(ctx, ctx.Err()), Duration: time.Since(start)}, nil
		}
	}
}

type sseSession struct {
	*eventLog
	cancel context.CancelFunc
}

func (s *sseSession) Send(context.Context, string) error {
	return errors.New("an event stream only receives")
}
func (s *sseSession) Ping(context.Context) error { return errors.New("an event stream has no ping") }

func (s *sseSession) Close(context.Context) error {
	s.close()
	s.cancel()
	return nil
}
