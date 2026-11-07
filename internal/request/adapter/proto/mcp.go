package proto

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// MCP speaks the Model Context Protocol, JSON-RPC 2.0, to a server over
// streamable HTTP (the URL is http or https) or stdio (the URL is
// "stdio:" followed by the command line). It is a small hand-written
// client, as the WebSocket one, so the project keeps no SDK dependency.
type MCP struct {
	// Sampler, when set, answers the sampling requests a server sends
	// (it asks the client to produce a model reply); without one the
	// client does not offer sampling and refuses such requests.
	Sampler Sampler
}

const (
	mcpVersion     = "2025-06-18"
	mcpInitTimeout = 30 * time.Second
	mcpMaxMessage  = 16 << 20
)

// StdioPrefix starts the URL of an MCP server that runs as a command.
const StdioPrefix = "stdio:"

// mcpTransport moves JSON-RPC messages to and from a server. Messages
// from the server go to the deliver function the transport was built with.
type mcpTransport interface {
	send(ctx context.Context, msg []byte) error
	setVersion(v string)
	close() error
}

// Connect implements Streamer: it opens the transport, runs the
// initialize handshake and asks for the server's tools, resources and
// prompts. Everything is logged as events.
func (m MCP) Connect(ctx context.Context, req model.Request) (Session, error) {
	s := &mcpSession{sampler: m.Sampler, events: make(chan Event, eventBuffer), done: make(chan struct{}), now: time.Now, waiters: map[int64]chan json.RawMessage{}}
	var tr mcpTransport
	var err error
	if strings.HasPrefix(req.URL, StdioPrefix) {
		tr, err = newMCPStdio(strings.TrimSpace(strings.TrimPrefix(req.URL, StdioPrefix)), s.deliver, s.finish)
	} else {
		tr, err = newMCPHTTP(req, s.deliver)
	}
	if err != nil {
		return nil, err
	}
	s.tr = tr
	s.emit(Event{Dir: Info, Kind: "open", Text: "Connecting to " + req.URL})
	ictx, cancel := context.WithTimeout(ctx, mcpInitTimeout)
	defer cancel()
	capabilities := map[string]any{}
	if m.Sampler != nil {
		capabilities["sampling"] = map[string]any{}
	}
	result, err := s.call(ictx, "initialize", map[string]any{
		"protocolVersion": mcpVersion,
		"capabilities":    capabilities,
		"clientInfo":      map[string]any{"name": "gascoigne-workshop", "version": "0"},
	})
	if err != nil {
		s.Close()
		return nil, err
	}
	var init struct {
		ProtocolVersion string         `json:"protocolVersion"`
		Capabilities    map[string]any `json:"capabilities"`
		ServerInfo      struct{ Name, Version string }
	}
	_ = json.Unmarshal(result, &init)
	tr.setVersion(init.ProtocolVersion)
	if listener, ok := tr.(interface{ listen() }); ok {
		listener.listen() // servers may push notifications on a stream of their own
	}
	if err := s.notify("notifications/initialized"); err != nil {
		s.Close()
		return nil, err
	}
	name := init.ServerInfo.Name
	if name == "" {
		name = req.URL
	}
	s.emit(Event{Dir: Info, Kind: "open", Text: "Connected to " + name})
	for _, list := range []struct{ capability, method string }{
		{"tools", "tools/list"}, {"resources", "resources/list"}, {"prompts", "prompts/list"},
	} {
		if _, ok := init.Capabilities[list.capability]; ok {
			if _, err := s.request(list.method, nil); err != nil {
				s.emit(Event{Dir: Info, Kind: "error", Text: err.Error()})
			}
		}
	}
	return s, nil
}

// Send implements Protocol as the one-shot form: connect, list the
// tools and return that answer as the body.
func (m MCP) Send(ctx context.Context, req model.Request) (model.Response, error) {
	start := time.Now()
	sess, err := m.Connect(ctx, req)
	if err != nil {
		return model.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
	}
	s := sess.(*mcpSession)
	defer s.Close()
	result, err := s.call(ctx, "tools/list", nil)
	if err != nil {
		return model.Response{Err: stopReason(ctx, err), Duration: time.Since(start)}, nil
	}
	return model.Response{Status: http.StatusOK, StatusText: "OK", Duration: time.Since(start), Size: len(result), Body: result}, nil
}

type mcpSession struct {
	tr      mcpTransport
	sampler Sampler // nil: sampling is not offered
	events  chan Event
	done    chan struct{}
	once    sync.Once
	now     func() time.Time

	mu      sync.Mutex
	ended   bool
	closing atomic.Bool // the user closed the session, so the end is no loss
	nextID  int64
	waiters map[int64]chan json.RawMessage
}

func (s *mcpSession) Events() <-chan Event { return s.events }

// emit queues e unless the session is over.
func (s *mcpSession) emit(e Event) {
	e.Time = s.now()
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

// finish ends the event stream, once, with a note unless the user closed it.
func (s *mcpSession) finish(err error) {
	s.once.Do(func() {
		if err != nil && !s.closing.Load() {
			s.emit(Event{Dir: Info, Kind: "close", Code: 1006, Text: "Connection lost: " + err.Error()})
		}
		close(s.done)
		s.mu.Lock()
		s.ended = true
		for id, w := range s.waiters {
			close(w)
			delete(s.waiters, id)
		}
		close(s.events)
		s.mu.Unlock()
	})
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
		s.emit(Event{Dir: Received, Kind: "invalid", Text: string(msg), Size: len(msg)})
		return
	}
	kind := "notification"
	switch {
	case m.Method == "" && m.Error != nil:
		kind = "error"
	case m.Method == "":
		kind = "response"
	case m.ID != nil:
		kind = "request"
	}
	s.emit(Event{Dir: Received, Kind: kind, Text: compactJSON(msg), Size: len(msg)})
	if kind == "request" {
		go s.answerServer(m)
		return
	}
	if list, ok := listChanged[m.Method]; ok {
		go func() { // ask again, so the lists the UI shows stay current
			if _, err := s.request(list, nil); err != nil {
				s.emit(Event{Dir: Info, Kind: "error", Text: err.Error()})
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
					w <- nil
				} else {
					w <- m.Result
				}
			}
		}
	}
}

func compactJSON(b []byte) string {
	var out bytes.Buffer
	if json.Compact(&out, b) != nil {
		return string(b)
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

// write sends a message and logs it.
func (s *mcpSession) write(ctx context.Context, kind string, b []byte) error {
	select {
	case <-s.done:
		return errors.New("the connection is closed")
	default:
	}
	s.emit(Event{Dir: Sent, Kind: kind, Text: compactJSON(b), Size: len(b)}) // before the send: an answer may arrive during it
	return s.tr.send(ctx, b)
}

// request sends a request without waiting for the answer, which arrives
// as an event.
func (s *mcpSession) request(method string, params any) (int64, error) {
	id, b, err := s.build(method, params)
	if err != nil {
		return 0, err
	}
	return id, s.write(context.Background(), "request", b)
}

func (s *mcpSession) notify(method string) error {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method})
	return s.write(context.Background(), "notification", b)
}

// call sends a request and waits for its result.
func (s *mcpSession) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id, b, err := s.build(method, params)
	if err != nil {
		return nil, err
	}
	w := make(chan json.RawMessage, 1)
	s.mu.Lock()
	s.waiters[id] = w
	s.mu.Unlock()
	if err := s.write(ctx, "request", b); err != nil {
		return nil, err
	}
	select {
	case r, ok := <-w:
		if !ok {
			return nil, errors.New("the connection closed before the server answered")
		}
		if r == nil {
			return nil, fmt.Errorf("%s failed: the server answered with an error", method)
		}
		return r, nil
	case <-ctx.Done():
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
func (s *mcpSession) Send(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("nothing to send")
	}
	if strings.HasPrefix(text, "{") {
		if !json.Valid([]byte(text)) {
			return errors.New("the message is not valid JSON")
		}
		return s.write(context.Background(), "request", []byte(text))
	}
	head, rest, _ := strings.Cut(text, " ")
	rest = strings.TrimSpace(rest)
	method, params := head, any(nil)
	if sh, ok := shorthands[head]; ok {
		var err error
		if method, params, err = sh(rest); err != nil {
			return err
		}
	} else if rest != "" {
		if err := json.Unmarshal([]byte(rest), &params); err != nil {
			return fmt.Errorf("the params must be JSON: %w", err)
		}
	}
	_, err := s.request(method, params)
	return err
}

func (s *mcpSession) Ping() error {
	_, err := s.request("ping", nil)
	return err
}

func (s *mcpSession) Close() error {
	s.closing.Store(true)
	err := s.tr.close()
	s.finish(nil)
	return err
}

// ---- streamable HTTP ----------------------------------------------------------

type mcpHTTP struct {
	req     model.Request
	client  *http.Client
	deliver func([]byte)

	mu        sync.Mutex
	sessionID string
	version   string
	cancel    context.CancelFunc
	ctx       context.Context
}

func newMCPHTTP(req model.Request, deliver func([]byte)) (*mcpHTTP, error) {
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		return nil, fmt.Errorf("%q is not an http or https URL, or a %s command", req.URL, StdioPrefix)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &mcpHTTP{req: req, client: &http.Client{}, deliver: deliver, ctx: ctx, cancel: cancel}, nil
}

func (h *mcpHTTP) setVersion(v string) {
	h.mu.Lock()
	h.version = v
	h.mu.Unlock()
}

func (h *mcpHTTP) newRequest(ctx context.Context, method string, body []byte) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, method, h.req.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for _, kv := range h.req.Headers {
		if kv.Enabled && kv.Name != "" {
			r.Header.Add(kv.Name, kv.Value)
		}
	}
	if apply, ok := authAppliers[h.req.Auth.Type]; ok {
		apply(h.req.Auth, r)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sessionID != "" {
		r.Header.Set("Mcp-Session-Id", h.sessionID)
	}
	if h.version != "" {
		r.Header.Set("MCP-Protocol-Version", h.version)
	}
	return r, nil
}

func (h *mcpHTTP) send(ctx context.Context, msg []byte) error {
	r, err := h.newRequest(h.ctx, http.MethodPost, msg)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	stop := context.AfterFunc(ctx, h.cancel)
	defer stop()
	resp, err := h.client.Do(r)
	if err != nil {
		return err
	}
	if id := resp.Header.Get("Mcp-Session-Id"); id != "" {
		h.mu.Lock()
		h.sessionID = id
		h.mu.Unlock()
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(snippet)))
	}
	ct := resp.Header.Get("Content-Type")
	switch {
	case resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent:
		resp.Body.Close()
	case strings.HasPrefix(ct, "text/event-stream"):
		go func() {
			defer resp.Body.Close()
			readSSE(resp.Body, h.deliver)
		}()
	default:
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, mcpMaxMessage))
		if err != nil {
			return err
		}
		deliverJSON(body, h.deliver)
	}
	return nil
}

// listen opens the optional stream a streamable HTTP server pushes
// requests and notifications on, in the background. A server without one
// answers 405 or another status, which is not an error: the stream is
// only read when the server offers it.
func (h *mcpHTTP) listen() {
	go func() {
		r, err := h.newRequest(h.ctx, http.MethodGet, nil)
		if err != nil {
			return
		}
		r.Header.Set("Accept", "text/event-stream")
		resp, err := h.client.Do(r)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
			return
		}
		readSSE(resp.Body, h.deliver)
	}()
}

// deliverJSON hands over one message, or each message of a batch.
func deliverJSON(body []byte, deliver func([]byte)) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return
	}
	if body[0] == '[' {
		var batch []json.RawMessage
		if json.Unmarshal(body, &batch) == nil {
			for _, m := range batch {
				deliver(m)
			}
			return
		}
	}
	deliver(body)
}

// readSSE reads server-sent events and delivers each event's data.
func readSSE(r io.Reader, deliver func([]byte)) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), mcpMaxMessage)
	var data []string
	flush := func() {
		if len(data) > 0 {
			deliverJSON([]byte(strings.Join(data, "\n")), deliver)
			data = nil
		}
	}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	flush()
}

func (h *mcpHTTP) close() error {
	h.mu.Lock()
	id := h.sessionID
	h.mu.Unlock()
	if id != "" { // tell the server the session is over; it may refuse, which is fine
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if r, err := h.newRequest(ctx, http.MethodDelete, nil); err == nil {
			if resp, err := h.client.Do(r); err == nil {
				resp.Body.Close()
			}
		}
	}
	h.cancel()
	return nil
}

// ---- stdio ----------------------------------------------------------------------

type mcpStdio struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser
	mu    sync.Mutex
}

func newMCPStdio(command string, deliver func([]byte), finish func(error)) (*mcpStdio, error) {
	args, err := model.ShellSplit(command)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, errors.New("stdio: needs a command")
	}
	cmd := exec.Command(args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64<<10), mcpMaxMessage)
		for sc.Scan() {
			if line := bytes.TrimSpace(sc.Bytes()); len(line) > 0 {
				deliver(append([]byte(nil), line...))
			}
		}
		err := cmd.Wait()
		if err == nil {
			err = errors.New("the server exited")
		}
		finish(err)
	}()
	return &mcpStdio{cmd: cmd, stdin: stdin}, nil
}

func (m *mcpStdio) send(_ context.Context, msg []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.stdin.Write(append(bytes.TrimSpace(msg), '\n'))
	return err
}

func (m *mcpStdio) setVersion(string) {}

func (m *mcpStdio) close() error {
	m.stdin.Close()
	t := time.AfterFunc(time.Second, func() { _ = m.cmd.Process.Kill() })
	defer t.Stop()
	return nil
}

// answerServer replies to a request the server sent: it answers ping and
// roots/list (this client offers no roots), and refuses anything else with
// "method not found", so a server that waits for an answer is not left
// hanging.
func (s *mcpSession) answerServer(m rpcMessage) {
	reply := map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(*m.ID)}
	switch m.Method {
	case "ping":
		reply["result"] = map[string]any{}
	case "roots/list":
		reply["result"] = map[string]any{"roots": []any{}}
	case "sampling/createMessage":
		if s.sampler == nil {
			reply["error"] = map[string]any{"code": -32601, "message": "method not found: " + m.Method}
			break
		}
		s.sample(m, reply)
	default:
		reply["error"] = map[string]any{"code": -32601, "message": "method not found: " + m.Method}
	}
	b, err := json.Marshal(reply)
	if err != nil {
		return
	}
	_ = s.write(context.Background(), "response", b)
}

// sampleRejected is the answer to a sampling request that was declined.
const sampleRejected = "User rejected sampling request"

// sample puts a sampling request to the Sampler and fills reply with its
// answer: a text message, or an error when the request is declined or
// cannot be read.
func (s *mcpSession) sample(m rpcMessage, reply map[string]any) {
	var params struct {
		Messages []struct {
			Role    string `json:"role"`
			Content struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"messages"`
		SystemPrompt string `json:"systemPrompt"`
		MaxTokens    int    `json:"maxTokens"`
	}
	if err := json.Unmarshal(m.Params, &params); err != nil {
		reply["error"] = map[string]any{"code": -32602, "message": "invalid sampling request: " + err.Error()}
		return
	}
	request := SamplingRequest{SystemPrompt: params.SystemPrompt, MaxTokens: params.MaxTokens}
	for _, message := range params.Messages {
		text := message.Content.Text
		if message.Content.Type != "text" {
			text = "[" + message.Content.Type + " content]"
		}
		request.Messages = append(request.Messages, SamplingMessage{Role: message.Role, Text: text})
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { // the session ending abandons the question
		select {
		case <-s.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	result, ok := s.sampler(ctx, request)
	if !ok {
		reply["error"] = map[string]any{"code": -1, "message": sampleRejected}
		return
	}
	reply["result"] = map[string]any{
		"role": "assistant", "content": map[string]any{"type": "text", "text": result.Text},
		"model": result.Model, "stopReason": "endTurn",
	}
}
