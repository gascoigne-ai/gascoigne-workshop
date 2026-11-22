package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strings"
	"time"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// A WebSocket or MCP tab keeps a session open instead of making one request.
// The Send button becomes Connect / Disconnect, the response side shows
// the message log, and the composer under the log sends messages.

// wsTarget says which connect of which tab a message belongs to, so a
// message of a connect that was abandoned or replaced is dropped.
type wsTarget struct {
	tab *reqTab
	gen int
}

func (target wsTarget) target() wsTarget { return target }

// wsMessage is what a session's goroutines tell the UI; all five carry a wsTarget.
type wsMessage interface{ target() wsTarget }

// What a session's goroutines tell the UI, one type each: a connect that
// produced a session, a connect that failed, an event of the session, the
// end of the session, and the end of the reconnect delay.
type (
	wsOpened struct {
		wsTarget
		session reqport.Session
	}
	wsFailed struct {
		wsTarget
		err error
	}
	wsReceived struct {
		wsTarget
		event reqport.Event
	}
	wsClosed struct{ wsTarget }
	wsRetry  struct{ wsTarget }
)

// defaultRetryDelay is the wait before a dropped connection is reopened.
const defaultRetryDelay = 2 * time.Second

// isSocket reports whether the tab's protocol keeps a connection open.
func isSocket(tab *reqTab) bool {
	connection := tab.Req.Protocol.Info().Connection
	return connection == model.Always || (connection == model.OnStream && tab.grpcStream)
}

// wsPanes are the panes of a socket's response side.
var wsPanes = []pane{{label: "Messages", kind: paneMessages}}

// mcpPanes are the response side of an MCP server: what it offers, and
// the JSON-RPC traffic with the composer for anything else.
var mcpPanes = []pane{{label: "Explorer", kind: paneExplorer}, {label: "JSON-RPC", kind: paneMessages}}

// responsePanesFor are the panes of the response side of tab.
func responsePanesFor(tab *reqTab) []pane {
	if isSocket(tab) {
		return viewOf(tab.Req.Protocol).streamPanes
	}
	return responsePanes
}

// toggleConnection connects a disconnected socket tab and closes a
// connected one.
func (a *App) toggleConnection(tab *reqTab) {
	if tab.Active() {
		a.disconnect(tab)
		return
	}
	a.connect(tab)
}

// connect dials in the background; the outcome arrives as a wsOpened or wsFailed.
func (a *App) connect(tab *reqTab) {
	if !a.allowed(tab, tab.Req, func() { a.connect(tab) }) {
		return
	}
	gen, ok := tab.Begin()
	if !ok {
		return
	}
	a.status = "Connecting to " + tab.Req.URL
	req := tab.Req.Expanded(a.varsFor(tab.Req))
	if header := tab.Req.Protocol.Info().ResumeHeader; header != "" && tab.LastEventID != "" { // resume where the stream stopped
		req.Headers = append(req.Headers, model.KV{Name: header, Value: tab.LastEventID, Enabled: true})
	}
	ctx, cancel := context.WithTimeout(context.Background(), a.timeoutFor(req))
	tab.wsCancel = cancel
	reg := a.reg // the goroutine must not read App fields the UI goroutine replaces
	a.bus.Run(context.Background(), func(context.Context) bus.Msg {
		defer cancel()
		s, err := reg.Connect(ctx, req) // ctx ends only the connect, not the session
		if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = context.DeadlineExceeded // the protocol reports the closed socket, not the cause
		}
		if err == nil {
			select {
			case <-ctx.Done(): // dialing took longer than the timeout
				if ctx.Err() == context.DeadlineExceeded {
					_ = s.Close(context.WithoutCancel(ctx))
					s, err = nil, context.DeadlineExceeded
				}
			default:
			}
		}
		target := wsTarget{tab: tab, gen: gen}
		if err != nil {
			a.bus.Dispatch(wsFailed{target, err})
			return nil
		}
		a.bus.Dispatch(wsOpened{target, s})
		for event := range s.Events() {
			a.bus.Dispatch(wsReceived{target, event})
		}
		return wsClosed{target}
	})
}

// disconnect closes the session of tab, or abandons a connect in progress.
func (a *App) disconnect(tab *reqTab) {
	session, abandoned := tab.Disconnect()
	if session != nil {
		_ = session.Close(context.Background()) // the end of the event stream clears the session
		return
	}
	if abandoned {
		tab.wsCancel()
		a.status = model.ErrCancelled
	}
}

// setComposer replaces the composer's text with text and focuses it.
func (tab *reqTab) setComposer(text string) {
	tab.msg = NewTextField(text)
	tab.msg.Focused = true
}

// sendMessage sends the composer's text and empties it.
func (a *App) sendMessage(tab *reqTab) {
	if tab.Session() == nil || tab.msg.Text == "" {
		return
	}
	send, err := composed(tab.Session(), tab.msg.Text)
	if err != nil {
		a.logInfo(tab, err.Error(), w.Error)
		return
	}
	a.sendInOrder(tab, send)
	tab.setComposer("")
}

// queuedSend is a send or ping waiting in a tab's outbox; it writes with
// the context it is run with.
type queuedSend func(ctx context.Context) error

// sendInOrder runs send off the UI goroutine, after the sends and pings
// already asked for on this tab: messages go out in the order they were
// written. A failure is logged in the message log.
func (a *App) sendInOrder(tab *reqTab, send queuedSend) {
	tab.outbox = append(tab.outbox, send)
	if tab.outboxBusy {
		return
	}
	tab.outboxBusy = true
	a.runOutbox(tab)
}

func (a *App) runOutbox(tab *reqTab) {
	send := tab.outbox[0]
	tab.outbox = tab.outbox[1:]
	a.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		err := send(ctx)
		return func() { // on the UI goroutine
			if err != nil {
				a.logInfo(tab, err.Error(), w.Error)
			}
			if len(tab.outbox) > 0 {
				a.runOutbox(tab)
				return
			}
			tab.outboxBusy = false
		}
	})
}

// pingSocket sends a ping.
func (a *App) pingSocket(tab *reqTab) {
	if tab.Session() == nil {
		return
	}
	a.sendInOrder(tab, tab.Session().Ping)
}

// clearLog empties the message log.
func (a *App) clearLog(tab *reqTab) {
	tab.wsView.SetLines(nil)
}

// handleWS drops a message of a connect that is no longer the tab's
// current one, closing the session of a stale wsOpened; it applies the rest.
func (a *App) handleWS(msg wsMessage) {
	target := msg.target()
	if target.tab.Current(target.gen) {
		a.applyWS(msg)
		return
	}
	if opened, ok := msg.(wsOpened); ok { // a connect that was abandoned meanwhile
		_ = opened.session.Close(context.Background())
	}
}

// applyWS applies one message of a session's goroutines.
func (a *App) applyWS(msg wsMessage) {
	switch msg := msg.(type) {
	case wsOpened:
		a.applyOpened(msg)
	case wsFailed:
		a.applyFailed(msg)
	case wsReceived:
		msg.tab.Received(msg.event)
		a.logEvent(msg.tab, msg.event)
	case wsClosed:
		a.applyClosed(msg)
	case wsRetry:
		if msg.tab.Retrying() {
			a.connect(msg.tab)
		}
	}
}

func (a *App) applyOpened(msg wsOpened) {
	tab := msg.tab
	if !tab.Opened(msg.session) { // the tab is no longer connecting
		_ = msg.session.Close(context.Background())
		return
	}
	a.calls.Info(a.now(), "Connected to "+tab.Req.URL)
	a.status = "Connected to " + tab.Req.URL
}

func (a *App) applyFailed(msg wsFailed) {
	tab := msg.tab
	if !tab.Failed() {
		return
	}
	a.calls.Error(a.now(), fmt.Sprintf("✗ %s: %s", tab.Req.URL, msg.err))
	a.logInfo(tab, msg.err.Error(), w.Error)
	a.status = "Connection failed"
	a.scheduleRetry(tab)
}

func (a *App) applyClosed(msg wsClosed) {
	tab := msg.tab
	if !tab.Closed() {
		return
	}
	a.calls.Info(a.now(), "Disconnected from "+tab.Req.URL)
	a.status = "Disconnected from " + tab.Req.URL
	a.scheduleRetry(tab)
}

// scheduleRetry reopens a connection the user did not close, when the
// tab asks for it.
func (a *App) scheduleRetry(tab *reqTab) {
	if !tab.Retrying() {
		return
	}
	a.logInfo(tab, "Reconnecting in "+a.retryDelay.String(), w.Muted)
	gen := tab.Gen
	time.AfterFunc(a.retryDelay, func() { a.bus.Dispatch(wsRetry{wsTarget{tab: tab, gen: gen}}) })
}

// logEvent appends an event to the log, one line per line of its text.
func (a *App) logEvent(tab *reqTab, event reqport.Event) {
	if event.Dir == reqport.Info && event.Kind == reqport.KindInitialize {
		return
	}
	mark, tok := Span{Text: "•", Icon: "dot"}, w.Muted
	switch event.Dir {
	case reqport.Sent:
		mark, tok = Span{Text: "↑", Icon: "arrow-up"}, w.Info
	case reqport.Received:
		mark, tok = Span{Text: "↓", Icon: "arrow-down"}, w.OK
	}
	text := event.Text
	if text == "" && !event.Kind.CarriesPayload() {
		text = event.Label() // a ping has no payload
	}
	if event.Dir == reqport.Info && strings.HasPrefix(event.Text, "Connection lost") {
		tok = w.Error
	}
	mark.Style = w.Muted // as the stamp, so the picture is the same as when the mark was a character
	stamp := event.Time.Format("15:04:05")
	if event.Time.IsZero() {
		stamp = time.Now().Format("15:04:05")
	}
	lines := tab.wsView.Lines
	for i, part := range strings.Split(text, "\n") {
		spans := []Span{{Text: stamp + " ", Style: w.Muted}, mark, {Text: " ", Style: w.Muted}}
		if i > 0 {
			spans = []Span{{Text: strings.Repeat(" ", len(stamp)) + "   ", Style: w.Muted}}
		}
		spans = append(spans, Span{Text: strings.TrimRight(part, "\r"), Style: textToken(event.Dir, tok)})
		lines = append(lines, Line{Spans: spans})
	}
	tab.wsView.SetLines(lines)
}

// textToken is the color of a log line's text: the direction's own for
// problems and notes, plain for messages.
func textToken(direction reqport.Direction, tok w.Token) w.Token {
	if direction == reqport.Info {
		return tok
	}
	return w.Text
}

// logInfo appends a note of the UI itself to the log.
func (a *App) logInfo(tab *reqTab, text string, tok w.Token) {
	a.logEvent(tab, reqport.Event{Time: time.Now(), Dir: reqport.Info, Kind: reqport.KindInfo, Text: text})
	if tok == w.Error { // recolor the line just added
		last := &tab.wsView.Lines[len(tab.wsView.Lines)-1]
		last.Spans[1].Style = w.Error
	}
}

var messagesPaneTemplate = markup.Must("messagespane.vml", messagespaneSource)

// messagesPane is the log with its toolbar and the composer.
func (a *App) messagesPane(_ pane, tab *reqTab, _ *ScrollState) Node {
	return messagesPaneTemplate.MustBuild(a.scope(markup.Scope{
		"tightGap":  w.TightGap,
		"gutter":    w.Gutter,
		"composerH": a.ctx.Sizes.Row + 2*w.LooseGap,
		"toolbar":   a.messagesToolbar(tab),
		"log":       &tab.wsView,
		"composer":  viewOf(tab.Req.Protocol).composer,
		"message":   &tab.msg,
		"hint":      composerHint(tab, tab.Session() != nil),
		"canFormat": viewOf(tab.Req.Protocol).formatsJSON,
		"send":      func() { a.sendMessage(tab) },
		"format":    func() { a.formatMessage(tab) },
	}))
}

// connLabels is what the connection bar says in each state.
var connLabels = [...]string{
	tabs.ConnIdle:       "Not connected",
	tabs.ConnConnecting: "Connecting...",
	tabs.ConnOpen:       "Connected",
	tabs.ConnClosing:    "Connected",
	tabs.ConnRetrying:   "Not connected",
}

func connState(tab *reqTab) string { return connLabels[tab.State()] }

func composerHint(tab *reqTab, connected bool) string {
	if !connected {
		return "Connect to send messages"
	}
	if hint := viewOf(tab.Req.Protocol).composerHint; hint != "" {
		return hint
	}
	return "Message"
}

// composed is the send of the composer's text, ready to run; a text that
// cannot be sent (bad hex) is refused here, before anything is sent.
func composed(session reqport.Session, text string) (queuedSend, error) {
	binarySender, canBinary := session.(reqport.BinarySender)
	if !canBinary {
		return func(ctx context.Context) error { return session.Send(ctx, text) }, nil
	}
	data, binary, err := model.BinaryMessage(text)
	if err != nil {
		return nil, err
	}
	if !binary {
		return func(ctx context.Context) error { return session.Send(ctx, text) }, nil
	}
	return func(ctx context.Context) error { return binarySender.SendBinary(ctx, data) }, nil
}

// toolMenu lists the tools the server announced; picking one puts a
// ready-to-edit call, with the arguments the tool's schema asks for, in
// the composer.
func (a *App) toolMenu(tab *reqTab) []MenuItem {
	items := make([]MenuItem, 0, len(tab.Tools))
	for _, tool := range tab.Tools {
		items = append(items, MenuItem{Label: tool.Name, Run: func() { a.openToolForm(tab, tool) }})
	}
	return items
}

// composeCall fills the composer with a call of tool.
func (a *App) composeCall(tab *reqTab, tool model.MCPTool) {
	args := "{}"
	if tool.Schema.Type != "" {
		args = model.Skeleton(tool.Schema)
	}
	tab.setComposer("call " + tool.Name + " " + args)
}

// resourceMenu lists the announced resources; picking one composes a read.
func (a *App) resourceMenu(tab *reqTab) []MenuItem {
	items := make([]MenuItem, 0, len(tab.Resources))
	for _, r := range tab.Resources {
		label := r.URI
		if r.Name != "" {
			label = r.Name + "  " + r.URI
		}
		items = append(items, MenuItem{Label: label, Run: func() {
			tab.setComposer("read " + r.URI)
		}})
	}
	return items
}

// promptMenu lists the announced prompts; picking one composes a request
// with its required arguments filled in.
func (a *App) promptMenu(tab *reqTab) []MenuItem {
	items := make([]MenuItem, 0, len(tab.Prompts))
	for _, p := range tab.Prompts {
		items = append(items, MenuItem{Label: p.Name, Run: func() {
			tab.setComposer("prompt " + p.Name + " " + p.ArgsJSON())
		}})
	}
	return items
}

// formatMessage turns the composer's JSON into one line, so a pasted,
// indented message goes out as a single frame; text that is not JSON
// stays as typed and a toast says where it breaks.
func (a *App) formatMessage(tab *reqTab) {
	compact, err := textfmt.CompactJSON(tab.msg.Text)
	if err != nil {
		a.toast(err.Error(), severity.Error)
		return
	}
	tab.setComposer(compact)
}

// Templates, parsed once above.
const (
	messagespaneSource = `<col grow="1">
  <!-- same top inset as the request pane, so first rows line up -->
  <spacer h="{{gutter}}"/>
  <slot name="toolbar"/>
  <code state="log" grow="1" bg="sunken"/>
  <!-- an event stream only receives, so it has no composer -->
  <row if="composer" h="{{composerH}}" pad-x="{{gutter}}" pad-y="{{looseGap}}" gap="{{tightGap}}" border-t="border">
    <field state="message" grow="1" placeholder="{{hint}}" on-enter="send" text-menu="message"/>
    <button if="canFormat" min-w="{{formatMsgMinW}}" on-click="format" tooltip="Make the message one line of JSON, or say where it is not JSON">{ }</button>
    <primary-button min-w="{{sendMsgMinW}}" on-click="send" tooltip="Send the message (Enter)">Send</primary-button>
  </row>
</col>
`
)
