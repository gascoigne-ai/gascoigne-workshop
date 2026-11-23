package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/outbox"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"time"
)

// send runs t's request off the UI goroutine; the result comes back
// through the bus and is applied in drain.
func (a *App) send(tab *reqTab) {
	if a.isSocket(tab) {
		a.toggleConnection(tab)
		return
	}
	a.dispatch(tab, tab.Req)
}

// dispatch sends req on behalf of tab: the request of the tab, or a
// helper one such as a GraphQL schema query that uses the tab's URL and
// credentials.
func (a *App) dispatch(tab *reqTab, source *request.Request) {
	if !a.auth.Allowed(tab.Tab, source, func() { a.dispatch(tab, source) }) {
		return
	}
	req := source.Expanded(a.varsFor(tab.Req))
	ctx, cancel := context.WithTimeout(context.Background(), a.timeoutFor(req))
	number, started := a.sends.Begin(tab.Req, cancel)
	if !started {
		cancel()
		return
	}
	a.notices.SetStatus("Sending " + source.URL)
	a.over.Calls().Sent(a.now(), req)
	reg := a.reg // the command must not read App fields the UI goroutine replaces
	a.bus.Run(ctx, func(ctx context.Context) bus.Msg {
		defer cancel()
		resp, err := reg.Send(ctx, req)
		if err != nil {
			resp = request.Response{Err: err.Error()}
		}
		return sends.Result{Owner: tab.Req, Number: number, Response: resp}
	})
}

// cancelSend stops the running request of tab. Its late result, which
// arrives after this, belongs to an older send and is dropped.
func (a *App) cancelSend(tab *reqTab) {
	if !a.sends.Cancel(tab.Req) {
		return
	}
	tab.Resp = &request.Response{Err: request.ErrCancelled}
	a.resp.ClearView(tab.Tab)
	a.notices.SetStatus(request.ErrCancelled)
}

// timeoutFor is how long a request may run: its own setting, else the
// default from the settings, else a minute.
func (a *App) timeoutFor(req request.Request) time.Duration {
	return a.settings.TimeoutFor(req.TimeoutSec)
}

// drain applies what other goroutines reported and reports whether there was
// anything.
func (a *App) drain() bool { return a.bus.Drain(a.handle) }

// handle applies one message from the bus.
func (a *App) handle(msg bus.Msg) {
	switch msg := msg.(type) {
	case sends.Result:
		a.apply(msg)
	case wsMessage:
		a.handleWS(msg)
	case func(): // something a goroutine wants done on the UI goroutine, such as a chosen file
		msg()
	case bus.Failed: // a background task panicked; the app lives on
		a.notices.SetStatus(msg.Err.Error())
	}
}

func (a *App) apply(result sends.Result) {
	if !a.sends.Accept(result) {
		return // the answer to a send that was cancelled or replaced
	}
	t := a.tabOfRequest(result.Owner)
	if t == nil {
		return
	}
	t.Remember(t.Resp, a.now())
	t.Resp = &result.Response
	a.saveHistory(t)
	if owner, path, _ := a.holderOf(t.Req); path != "" {
		a.side.Sent().Record(a, a.now(), owner.Dir, path, t.Req, result.Response.Status)
	}
	a.over.Calls().Answer(a.now(), t.Req, &result.Response)
	a.graphql.Remember(t.Tab, &result.Response)
	a.resp.Refresh(t.Tab)
	if result.Response.Err != "" {
		a.notices.SetStatus("Request failed")
		a.assertions.Run(t.Tab)
		return
	}
	a.notices.SetStatus(fmt.Sprintf("%s %s: %d %s in %s", t.Req.Method, t.Req.URL, result.Response.Status, result.Response.StatusText, format.Duration(result.Response.Duration)))
	a.assertions.Run(t.Tab)
}

// A WebSocket or MCP tab keeps a session open instead of making one send.
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
		session request.Session
	}
	wsFailed struct {
		wsTarget
		err error
	}
	wsReceived struct {
		wsTarget
		event request.Event
	}
	wsClosed struct{ wsTarget }
	wsRetry  struct{ wsTarget }
)

// defaultRetryDelay is the wait before a dropped connection is reopened.
const defaultRetryDelay = 2 * time.Second

// isSocket reports whether the tab's protocol keeps a connection open: always,
// or on a stream the gRPC component says the tab has turned on.
func (a *App) isSocket(tab *reqTab) bool {
	connection := tab.Req.Protocol.Info().Connection
	return connection == request.Always || (connection == request.OnStream && a.grpc.Streaming(tab.Tab))
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
	if !a.auth.Allowed(tab.Tab, tab.Req, func() { a.connect(tab) }) {
		return
	}
	gen, ok := tab.Begin()
	if !ok {
		return
	}
	a.notices.SetStatus("Connecting to " + tab.Req.URL)
	req := tab.Req.Expanded(a.varsFor(tab.Req))
	if header := tab.Req.Protocol.Info().ResumeHeader; header != "" && tab.LastEventID != "" { // resume where the stream stopped
		req.Headers = append(req.Headers, request.KV{Name: header, Value: tab.LastEventID, Enabled: true})
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
		a.notices.SetStatus(request.ErrCancelled)
	}
}

// sendInOrder runs send off the UI goroutine, after the sends and pings
// already asked for on this tab: messages go out in the order they were
// written. A failure is logged in the message log.
func (a *App) sendInOrder(tab *reqTab, send outbox.Send) {
	if first, now := tab.outbox.Enqueue(send); now {
		a.runOutbox(tab, first)
	}
}

func (a *App) runOutbox(tab *reqTab, send outbox.Send) {
	a.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		err := send(ctx)
		return func() { // on the UI goroutine
			if err != nil {
				a.ws.Note(tab.Tab, err.Error(), severity.Error)
			}
			if next, more := tab.outbox.Done(); more {
				a.runOutbox(tab, next)
			}
		}
	})
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
		a.ws.Log(msg.tab.Tab, msg.event)
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
	a.over.Calls().Info(a.now(), "Connected to "+tab.Req.URL)
	a.notices.SetStatus("Connected to " + tab.Req.URL)
}

func (a *App) applyFailed(msg wsFailed) {
	tab := msg.tab
	if !tab.Failed() {
		return
	}
	a.over.Calls().Error(a.now(), fmt.Sprintf("✗ %s: %s", tab.Req.URL, msg.err))
	a.ws.Note(tab.Tab, msg.err.Error(), severity.Error)
	a.notices.SetStatus("Connection failed")
	a.scheduleRetry(tab)
}

func (a *App) applyClosed(msg wsClosed) {
	tab := msg.tab
	if !tab.Closed() {
		return
	}
	a.over.Calls().Info(a.now(), "Disconnected from "+tab.Req.URL)
	a.notices.SetStatus("Disconnected from " + tab.Req.URL)
	a.scheduleRetry(tab)
}

// scheduleRetry reopens a connection the user did not close, when the
// tab asks for it.
func (a *App) scheduleRetry(tab *reqTab) {
	if !tab.Retrying() {
		return
	}
	a.ws.Note(tab.Tab, "Reconnecting in "+a.retryDelay.String(), severity.Info)
	gen := tab.Gen
	time.AfterFunc(a.retryDelay, func() { a.bus.Dispatch(wsRetry{wsTarget{tab: tab, gen: gen}}) })
}
