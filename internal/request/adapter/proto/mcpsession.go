package proto

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
)

type mcpSession struct {
	*eventLog
	tr      mcpTransport
	sampler port.Sampler // nil: sampling is not offered
	once    sync.Once    // finish

	mu      sync.Mutex // guards nextID and waiters
	nextID  int64
	waiters map[int64]chan rpcAnswer
}

// note logs a problem that leaves the session open.
func (s *mcpSession) note(err error) {
	s.emit(port.Event{Dir: port.Info, Kind: port.KindError, Text: err.Error()})
}

// finish ends the event stream, once, with a note unless the user closed it.
func (s *mcpSession) finish(err error) {
	s.once.Do(func() {
		if err != nil && !s.closing() {
			s.emit(port.Event{Dir: port.Info, Kind: port.KindClose, Code: ws.CloseAbnormal, Text: "Connection lost: " + err.Error()})
		}
		s.stopWaiting()
		s.mu.Lock()
		for id, w := range s.waiters {
			close(w)
			delete(s.waiters, id)
		}
		s.mu.Unlock()
		s.eventLog.finish()
	})
}

// rpcAnswer is what a waiting call gets: the result, or the server's error.
type rpcAnswer struct {
	result json.RawMessage
	err    error
}

type rpcMessage struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params json.RawMessage  `json:"params"`
	Result json.RawMessage  `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// listChanged maps the notifications that say a list changed to the
// request that fetches it again.
var listChanged = map[string]string{
	"notifications/tools/list_changed":     "tools/list",
	"notifications/resources/list_changed": "resources/list",
	"notifications/prompts/list_changed":   "prompts/list",
}

// deliver handles a message from the server: it is logged, and a
// response wakes the call waiting for it.
func (s *mcpSession) deliver(msg []byte) {
	var m rpcMessage
	if err := json.Unmarshal(msg, &m); err != nil {
		s.emit(port.Event{Dir: port.Received, Kind: port.KindInvalid, Text: string(msg), Size: len(msg)})
		return
	}
	kind := port.KindNotification
	switch {
	case m.Method == "" && m.Error != nil:
		kind = port.KindError
	case m.Method == "":
		kind = port.KindResponse
	case m.ID != nil:
		kind = port.KindRequest
	}
	s.emit(port.Event{Dir: port.Received, Kind: kind, Text: compactJSON(msg), Size: len(msg)})
	if kind == port.KindRequest {
		go s.answerServer(m)
		return
	}
	if list, ok := listChanged[m.Method]; ok {
		go func() { // ask again, so the lists the UI shows stay current
			if _, err := s.request(context.Background(), list, nil); err != nil {
				s.emit(port.Event{Dir: port.Info, Kind: port.KindError, Text: err.Error()})
			}
		}()
	}
	if m.Method == "" && m.ID != nil {
		var id int64
		if json.Unmarshal(*m.ID, &id) == nil {
			s.mu.Lock()
			w := s.waiters[id]
			delete(s.waiters, id)
			s.mu.Unlock()
			if w != nil {
				if m.Error != nil {
					w <- rpcAnswer{err: errors.New(m.Error.Message)}
				} else {
					w <- rpcAnswer{result: m.Result}
				}
			}
		}
	}
}

func compactJSON(data []byte) string {
	var out bytes.Buffer
	if json.Compact(&out, data) != nil {
		return string(data)
	}
	return out.String()
}

// build makes a request message with the next id.
func (s *mcpSession) build(method string, params any) (int64, []byte, error) {
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.mu.Unlock()
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	b, err := json.Marshal(msg)
	return id, b, err
}

// write sends a message and logs it. A context that has ended sends
// nothing: over HTTP it would also end the session.
func (s *mcpSession) write(ctx context.Context, kind port.EventKind, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-s.done:
		return errors.New("the connection is closed")
	default:
	}
	s.emit(port.Event{Dir: port.Sent, Kind: kind, Text: compactJSON(data), Size: len(data)}) // before the send: an answer may arrive during it
	return s.tr.send(ctx, data)
}

// request sends a request without waiting for the answer, which arrives
// as an event.
func (s *mcpSession) request(ctx context.Context, method string, params any) (int64, error) {
	id, b, err := s.build(method, params)
	if err != nil {
		return 0, err
	}
	return id, s.write(ctx, port.KindRequest, b)
}

func (s *mcpSession) notify(method string) error {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method})
	return s.write(context.Background(), port.KindNotification, b)
}

// call sends a request and waits for its result.
func (s *mcpSession) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id, b, err := s.build(method, params)
	if err != nil {
		return nil, err
	}
	w := make(chan rpcAnswer, 1)
	s.mu.Lock()
	s.waiters[id] = w
	s.mu.Unlock()
	forget := func() { // a call that ends unanswered must not stay registered
		s.mu.Lock()
		delete(s.waiters, id)
		s.mu.Unlock()
	}
	if err := s.write(ctx, port.KindRequest, b); err != nil {
		forget()
		return nil, err
	}
	select {
	case answer, ok := <-w:
		if !ok {
			return nil, errors.New("the connection closed before the server answered")
		}
		if answer.err != nil {
			return nil, fmt.Errorf("%s failed: %w", method, answer.err)
		}
		return answer.result, nil
	case <-ctx.Done():
		forget()
		return nil, ctx.Err()
	}
}

// shorthands turn what the user types into a method and its params:
// "call NAME {args}", "read URI", "prompt NAME {args}".
var shorthands = map[string]func(arg string) (string, any, error){
	"call": func(arg string) (string, any, error) {
		name, args, err := nameAndArgs(arg)
		return "tools/call", map[string]any{"name": name, "arguments": args}, err
	},
	"prompt": func(arg string) (string, any, error) {
		name, args, err := nameAndArgs(arg)
		return "prompts/get", map[string]any{"name": name, "arguments": args}, err
	},
	"read": func(arg string) (string, any, error) {
		if arg == "" {
			return "", nil, errors.New("read needs a resource URI")
		}
		return "resources/read", map[string]any{"uri": arg}, nil
	},
}

func nameAndArgs(arg string) (string, any, error) {
	name, rest, _ := strings.Cut(arg, " ")
	if name == "" {
		return "", nil, errors.New("a name is needed")
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return name, map[string]any{}, nil
	}
	var args any
	if err := json.Unmarshal([]byte(rest), &args); err != nil {
		return "", nil, fmt.Errorf("the arguments must be JSON: %w", err)
	}
	return name, args, nil
}

// Send implements Session. The text is a JSON-RPC message, a shorthand
// (see shorthands) or "method {json params}".
func (s *mcpSession) Send(ctx context.Context, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("nothing to send")
	}
	if strings.HasPrefix(text, "{") {
		if !json.Valid([]byte(text)) {
			return errors.New("the message is not valid JSON")
		}
		return s.write(ctx, port.KindRequest, []byte(text))
	}
	method, params, err := parseCommand(text)
	if err != nil {
		return err
	}
	_, err = s.request(ctx, method, params)
	return err
}

// parseCommand turns "call NAME {json}", another shorthand or
// "method {json params}" into a method and its params.
func parseCommand(text string) (method string, params any, err error) {
	head, rest, _ := strings.Cut(text, " ")
	rest = strings.TrimSpace(rest)
	if shorthand, ok := shorthands[head]; ok {
		return shorthand(rest)
	}
	if rest != "" {
		if err := json.Unmarshal([]byte(rest), &params); err != nil {
			return "", nil, fmt.Errorf("the params must be JSON: %w", err)
		}
	}
	return head, params, nil
}

// Request implements Requester: it sends the command like Send does and
// waits for the server's answer. The exchange is logged as usual.
func (s *mcpSession) Request(ctx context.Context, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "{") {
		return "", errors.New("a method, or a shorthand like call NAME {json}, is needed")
	}
	method, params, err := parseCommand(text)
	if err != nil {
		return "", err
	}
	result, err := s.call(ctx, method, params)
	return string(result), err
}

func (s *mcpSession) Ping(ctx context.Context) error {
	_, err := s.request(ctx, "ping", nil)
	return err
}

func (s *mcpSession) Close(ctx context.Context) error {
	s.close() // frees a finish that waits for room in a full channel
	err := s.tr.close(ctx)
	s.finish(nil)
	return err
}
