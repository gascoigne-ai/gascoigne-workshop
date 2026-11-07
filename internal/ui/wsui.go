package ui

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/textfmt"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
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
func isSocket(t *reqTab) bool {
	return t.req.Protocol == "websocket" || t.req.Protocol == "mcp" || t.req.Protocol == "sse" ||
		(t.req.Protocol == "grpc" && t.grpcStream)
}

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
	req := t.req.Expanded(a.varsFor(t.req))
	if t.req.Protocol == "sse" && t.lastEventID != "" { // resume where the stream stopped
		req.Headers = append(req.Headers, model.KV{Name: "Last-Event-ID", Value: t.lastEventID, Enabled: true})
	}
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
	t.lastEventID = "" // a deliberate disconnect starts the stream over next time
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
	if err := sendComposed(t.sess, t.msg.Text); err != nil {
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
		a.traceInfo("Disconnected from " + t.req.URL)
		t.tools, t.resources, t.prompts = nil, nil, nil
		a.status = "Disconnected from " + t.req.URL
		a.scheduleRetry(t)
	case e.hasEv:
		if tools := proto.ParseTools(e.ev.Text); e.ev.Dir == proto.Received && tools != nil {
			t.tools = tools
		}
		if e.ev.Dir == proto.Received && e.ev.ID != "" {
			t.lastEventID = e.ev.ID
		}
		if e.ev.Dir == proto.Received {
			if res := proto.ParseResources(e.ev.Text); res != nil {
				t.resources = res
			}
			if prompts := proto.ParsePrompts(e.ev.Text); prompts != nil {
				t.prompts = prompts
			}
		}
		a.logEvent(t, e.ev)
	case e.err != nil:
		t.connecting = false
		a.traceError(fmt.Sprintf("✗ %s: %s", t.req.URL, e.err))
		a.logInfo(t, e.err.Error(), w.Error)
		a.status = "Connection failed"
		a.scheduleRetry(t)
	default:
		t.sess, t.connecting = e.sess, false
		a.traceInfo("Connected to " + t.req.URL)
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
		Spacer(H(w.Gutter)),
		a.messagesToolbar(t),
		Code(&t.wsView, Grow(1), Bg(w.Sunken)),
		If(t.req.Protocol != "sse", // an event stream only receives, so it has no composer
			Row(H(w.RowH+16), PadX(w.Gutter), PadY(8), Gap(6), BorderT(w.Border),
				Field(&t.msg, Grow(1), Placeholder(composerHint(t, connected)),
					OnEnter(func() { a.sendMessage(t) }),
					a.textMenuOpt(&t.msg, nil),
				),
				If(t.req.Protocol == "websocket", Button("{ }", W(40), OnClick(func() { a.formatMessage(t) }), Tooltip("Make the message one line of JSON, or say where it is not JSON"))),
				PrimaryButton("Send", W(70), OnClick(func() { a.sendMessage(t) }), Tooltip("Send the message (Enter)")),
			)),
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

// binaryPrefix starts a composer line that is sent as a binary message:
// "hex: de ad be ef".
const binaryPrefix = "hex:"

// sendComposed sends the composer's text: as a binary message when it
// starts with "hex:" and the session can send binary, else as text.
func sendComposed(s proto.Session, text string) error {
	rest, ok := strings.CutPrefix(text, binaryPrefix)
	bs, canBinary := s.(proto.BinarySender)
	if !ok || !canBinary {
		return s.Send(text)
	}
	data, err := hex.DecodeString(strings.Join(strings.Fields(rest), ""))
	if err != nil {
		return fmt.Errorf("hex: %v", err)
	}
	return bs.SendBinary(data)
}

// toolMenu lists the tools the server announced; picking one puts a
// ready-to-edit call, with the arguments the tool's schema asks for, in
// the composer.
func (a *App) toolMenu(t *reqTab) []menuItem {
	items := make([]menuItem, 0, len(t.tools))
	for _, tool := range t.tools {
		items = append(items, menuItem{label: tool.Name, run: func() { a.openToolForm(t, tool) }})
	}
	return items
}

// composeCall fills the composer with a call of tool.
func (a *App) composeCall(t *reqTab, tool proto.MCPTool) {
	args := "{}"
	if tool.Schema != nil {
		args = proto.Skeleton(tool.Schema)
	}
	t.msg = NewTextField("call " + tool.Name + " " + args)
	t.msg.Focused = true
}

// resourceMenu lists the announced resources; picking one composes a read.
func (a *App) resourceMenu(t *reqTab) []menuItem {
	items := make([]menuItem, 0, len(t.resources))
	for _, r := range t.resources {
		label := r.URI
		if r.Name != "" {
			label = r.Name + "  " + r.URI
		}
		items = append(items, menuItem{label: label, run: func() {
			t.msg = NewTextField("read " + r.URI)
			t.msg.Focused = true
		}})
	}
	return items
}

// promptMenu lists the announced prompts; picking one composes a request
// with its required arguments filled in.
func (a *App) promptMenu(t *reqTab) []menuItem {
	items := make([]menuItem, 0, len(t.prompts))
	for _, p := range t.prompts {
		items = append(items, menuItem{label: p.Name, run: func() {
			t.msg = NewTextField("prompt " + p.Name + " " + p.ArgsJSON())
			t.msg.Focused = true
		}})
	}
	return items
}

// formatMessage turns the composer's JSON into one line, so a pasted,
// indented message goes out as a single frame; text that is not JSON
// stays as typed and a toast says where it breaks.
func (a *App) formatMessage(t *reqTab) {
	compact, err := textfmt.CompactJSON(t.msg.Text)
	if err != nil {
		a.toast(err.Error(), true)
		return
	}
	t.msg = NewTextField(compact)
	t.msg.Focused = true
}
