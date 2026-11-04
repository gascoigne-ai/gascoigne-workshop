package ui

import (
	"context"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// A WebSocket or MCP tab keeps a session open instead of making one request.
// The Send button becomes Connect / Disconnect, the response side shows
// the message log, and the composer under the log sends messages.

// wsEvent is something a session's goroutines tell the UI: a finished
// connect (sess or err), a logged event, or the end of the session.
type wsEvent struct {
	tab    *reqTab
	gen    int // which connect of the tab this belongs to
	sess   proto.Session
	err    error
	ev     proto.Event
	hasEv  bool
	closed bool
	retry  bool // the reconnect delay is over
}

// retryDelay is the wait before a dropped connection is reopened.
var retryDelay = 2 * time.Second

// isSocket reports whether the tab's protocol keeps a connection open.
func isSocket(t *reqTab) bool { return t.req.Protocol == "websocket" || t.req.Protocol == "mcp" }

// wsPanes are the panes of a socket's response side.
var wsPanes = []pane{{label: "Messages", kind: paneMessages}}

// responsePanesFor are the panes of the response side of t.
func responsePanesFor(t *reqTab) []pane {
	if isSocket(t) {
		return wsPanes
	}
	return responsePanes
}

// toggleConnection connects a disconnected socket tab and closes a
// connected one.
func (a *App) toggleConnection(t *reqTab) {
	if t.sess != nil || t.connecting || t.wantOpen {
		a.disconnect(t)
		return
	}
	a.connect(t)
}

// connect dials in the background; the outcome arrives as a wsEvent.
func (a *App) connect(t *reqTab) {
	t.wantOpen = true
	t.wsGen++
	gen := t.wsGen
	t.connecting = true
	a.status = "Connecting to " + t.req.URL
	req := t.req.Expanded(a.vars())
	ctx, cancel := context.WithTimeout(context.Background(), a.timeoutFor(req))
	t.wsCancel = cancel
	go func() {
		defer cancel()
		s, err := a.reg.Connect(context.Background(), req)
		if err == nil {
			select {
			case <-ctx.Done(): // dialing took longer than the timeout
				if ctx.Err() == context.DeadlineExceeded {
					_ = s.Close()
					s, err = nil, context.DeadlineExceeded
				}
			default:
			}
		}
		a.wsEvents <- wsEvent{tab: t, gen: gen, sess: s, err: err}
		if err != nil {
			return
		}
		for e := range s.Events() {
			a.wsEvents <- wsEvent{tab: t, gen: gen, ev: e, hasEv: true}
		}
		a.wsEvents <- wsEvent{tab: t, gen: gen, closed: true}
	}()
}

// disconnect closes the session of t, or abandons a connect in progress.
func (a *App) disconnect(t *reqTab) {
	t.wantOpen = false
	if t.sess != nil {
		_ = t.sess.Close() // the end of the event stream clears the session
		return
	}
	if t.connecting {
		t.wsGen++ // the answer of the abandoned connect is dropped
		t.connecting = false
		t.wsCancel()
		a.status = proto.ErrCancelled
	}
}

// sendMessage sends the composer's text and empties it.
func (a *App) sendMessage(t *reqTab) {
	if t.sess == nil || t.msg.Text == "" {
		return
	}
	if err := t.sess.Send(t.msg.Text); err != nil {
		a.logInfo(t, err.Error(), w.Error)
		return
	}
	t.msg = NewTextField("")
	t.msg.Focused = true
}

// pingSocket sends a ping.
func (a *App) pingSocket(t *reqTab) {
	if t.sess == nil {
		return
	}
	if err := t.sess.Ping(); err != nil {
		a.logInfo(t, err.Error(), w.Error)
	}
}

// clearLog empties the message log.
func (a *App) clearLog(t *reqTab) {
	t.wsView.SetLines(nil)
}

// applyWS applies one wsEvent.
func (a *App) applyWS(e wsEvent) {
	t := e.tab
	if e.gen != t.wsGen {
		if e.sess != nil { // a connect that was abandoned meanwhile
			_ = e.sess.Close()
		}
		return
	}
	switch {
	case e.retry:
		if t.wantOpen && t.sess == nil && !t.connecting {
			a.connect(t)
		}
	case e.closed:
		t.sess, t.connecting = nil, false
		a.status = "Disconnected from " + t.req.URL
		a.scheduleRetry(t)
	case e.hasEv:
		a.logEvent(t, e.ev)
	case e.err != nil:
		t.connecting = false
		a.logInfo(t, e.err.Error(), w.Error)
		a.status = "Connection failed"
		a.scheduleRetry(t)
	default:
		t.sess, t.connecting = e.sess, false
		a.status = "Connected to " + t.req.URL
	}
}

// scheduleRetry reopens a connection the user did not close, when the
// tab asks for it.
func (a *App) scheduleRetry(t *reqTab) {
	if !t.reconnect || !t.wantOpen {
		return
	}
	a.logInfo(t, "Reconnecting in "+retryDelay.String(), w.Muted)
	gen := t.wsGen
	time.AfterFunc(retryDelay, func() { a.wsEvents <- wsEvent{tab: t, gen: gen, retry: true} })
}

// logEvent appends an event to the log, one line per line of its text.
func (a *App) logEvent(t *reqTab, e proto.Event) {
	mark, tok := "•", w.Muted
	switch e.Dir {
	case proto.Sent:
		mark, tok = "↑", w.Info
	case proto.Received:
		mark, tok = "↓", w.OK
	}
	text := e.Text
	if text == "" && e.Kind != "text" && e.Kind != "binary" {
		text = e.Kind // a ping has no payload
	}
	if e.Dir == proto.Info && strings.HasPrefix(e.Text, "Connection lost") {
		tok = w.Error
	}
	stamp := e.Time.Format("15:04:05")
	if e.Time.IsZero() {
		stamp = time.Now().Format("15:04:05")
	}
	lines := t.wsView.Lines
	for i, part := range strings.Split(text, "\n") {
		head := stamp + " " + mark + " "
		if i > 0 {
			head = strings.Repeat(" ", len(stamp)) + "   "
		}
		lines = append(lines, Line{Spans: []Span{
			{Text: head, Style: w.Muted},
			{Text: strings.TrimRight(part, "\r"), Style: textToken(e.Dir, tok)},
		}})
	}
	t.wsView.SetLines(lines)
}

// textToken is the color of a log line's text: the direction's own for
// problems and notes, plain for messages.
func textToken(d proto.Direction, tok w.Token) w.Token {
	if d == proto.Info {
		return tok
	}
	return w.Text
}

// logInfo appends a note of the UI itself to the log.
func (a *App) logInfo(t *reqTab, text string, tok w.Token) {
	a.logEvent(t, proto.Event{Time: time.Now(), Dir: proto.Info, Kind: "info", Text: text})
	if tok == w.Error { // recolor the line just added
		last := &t.wsView.Lines[len(t.wsView.Lines)-1]
		last.Spans[1].Style = w.Error
	}
}

// messagesPane is the log with its toolbar and the composer.
func (a *App) messagesPane(_ pane, t *reqTab, _ *ScrollState) Node {
	connected := t.sess != nil
	return Col(Grow(1),
		Spacer(H(w.Pad)),
		Row(H(w.RowH+8), PadX(w.Pad), Gap(6), AlignItems(AlignCenter), BorderBottom(w.Border),
			Text(connState(t), Small(), Muted()),
			Spacer(),
			a.viewChip("Auto-reconnect", t.reconnect, 116, func() { t.reconnect = !t.reconnect }),
			Button("Ping", W(56), H(w.CompactH), OnClick(func() { a.pingSocket(t) }), Tooltip("Send a ping frame")),
			Button("Clear", W(56), H(w.CompactH), OnClick(func() { a.clearLog(t) }), Tooltip("Empty the message log")),
		),
		Code(&t.wsView, Grow(1), Bg(w.Sunken)),
		Row(H(w.RowH+16), PadX(w.Pad), PadY(8), Gap(6), BorderTop(w.Border),
			Field(&t.msg, Grow(1), Placeholder(composerHint(t, connected)),
				OnEnter(func() { a.sendMessage(t) }),
				a.textMenuOpt(&t.msg, nil),
			),
			PrimaryButton("Send", W(70), OnClick(func() { a.sendMessage(t) }), Tooltip("Send the message (Enter)")),
		),
	)
}

func connState(t *reqTab) string {
	switch {
	case t.connecting:
		return "Connecting..."
	case t.sess != nil:
		return "Connected"
	}
	return "Not connected"
}

func composerHint(t *reqTab, connected bool) string {
	switch {
	case !connected:
		return "Connect to send messages"
	case t.req.Protocol == "mcp":
		return "call TOOL {json}  ·  read URI  ·  prompt NAME {json}  ·  method {json}  ·  or raw JSON-RPC"
	}
	return "Message"
}
