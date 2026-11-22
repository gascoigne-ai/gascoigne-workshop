package proto

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

type mcpHTTP struct {
	req     request.Request
	client  *http.Client
	deliver func([]byte)
	finish  func(error) // ends the session when an answer stream breaks
	note    func(error) // logs a problem that leaves the session open
	debug   debugFunc   // logs what is normal or harmless, with its reason

	mu        sync.Mutex
	sessionID string
	version   string
	cancel    context.CancelFunc
	ctx       context.Context
}

func newMCPHTTP(req request.Request, network Network, deliver func([]byte), finish, note func(error)) (*mcpHTTP, error) {
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		return nil, fmt.Errorf("%q is not an http or https URL, or a %s command", req.URL, request.StdioPrefix)
	}
	if err := checkAuth(req.Auth); err != nil {
		return nil, err
	}
	client, err := newHTTPClient(nil, network, req)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &mcpHTTP{req: req, client: client, deliver: deliver, finish: finish, note: note, debug: slog.Debug, ctx: ctx, cancel: cancel}, nil
}

func (h *mcpHTTP) setSession(id string) {
	h.mu.Lock()
	h.sessionID = id
	h.mu.Unlock()
}

func (h *mcpHTTP) setVersion(value string) {
	h.mu.Lock()
	h.version = value
	h.mu.Unlock()
}

func (h *mcpHTTP) newRequest(ctx context.Context, method string, body []byte) (*http.Request, error) {
	r, err := newHTTPRequest(ctx, method, h.req, body, "")
	if err != nil {
		return nil, err
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
			if err := h.readStream(resp.Body); err != nil { // the answer is cut, so the call cannot complete
				h.finish(err)
			}
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
			h.debug("mcp http: listen: build the request", "error", err)
			return
		}
		r.Header.Set("Accept", "text/event-stream")
		resp, err := h.client.Do(r)
		if err != nil {
			h.debug("mcp http: listen: connect", "error", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
			h.debug("mcp http: listen: no server stream", "status", resp.Status) // the usual answer of a server that never pushes
			return
		}
		err = h.readStream(resp.Body)
		switch {
		case errors.Is(err, errEventTooLarge):
			h.finish(err)
		case err != nil: // the stream is optional: only notifications are lost
			h.note(err)
		}
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

// readStream delivers the data of each event the server streams. It
// returns why the stream broke or went over maxEventBytes, and nil for a
// stream that ended or was cut by close.
func (h *mcpHTTP) readStream(body io.Reader) error {
	err := readEventStream(body, func(event sseEvent) { deliverJSON([]byte(event.data), h.deliver) })
	if err == nil || h.ctx.Err() != nil {
		return nil
	}
	return fmt.Errorf("read the server's stream: %w", err)
}

// goodbye tells the server the session is over; a failure is only logged
// because the session ends either way.
func (h *mcpHTTP) goodbye(ctx context.Context) {
	r, err := h.newRequest(ctx, http.MethodDelete, nil)
	if err != nil {
		h.debug("mcp http: end the session: build the request", "error", err)
		return
	}
	resp, err := h.client.Do(r)
	if err != nil {
		h.debug("mcp http: end the session", "error", err)
		return
	}
	resp.Body.Close()
}

func (h *mcpHTTP) close(ctx context.Context) error {
	h.mu.Lock()
	id := h.sessionID
	h.mu.Unlock()
	if id != "" { // tell the server the session is over; it may refuse, which is fine
		ctx, cancel := context.WithTimeout(ctx, mcpGoodbyeTimeout)
		defer cancel()
		h.goodbye(ctx)
	}
	h.cancel()
	return nil
}
