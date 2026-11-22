package app

import (
	"context"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/bus"
	"time"
)

type sendResult struct {
	tab  *reqTab
	seq  int // which send of the tab this answers
	resp request.Response
}

// send runs t's request off the UI goroutine; the result comes back
// through the bus and is applied in drain.
func (a *App) send(tab *reqTab) {
	if isSocket(tab) {
		a.toggleConnection(tab)
		return
	}
	a.dispatch(tab, tab.Req)
}

// dispatch sends req on behalf of tab: the request of the tab, or a
// helper one such as a GraphQL schema query that uses the tab's URL and
// credentials.
func (a *App) dispatch(tab *reqTab, source *request.Request) {
	if !a.allowed(tab, source, func() { a.dispatch(tab, source) }) {
		return
	}
	seq, started := tab.BeginSend()
	if !started {
		return
	}
	a.status = "Sending " + source.URL
	req := source.Expanded(a.varsFor(tab.Req))
	a.calls.Sent(a.now(), req)
	ctx, cancel := context.WithTimeout(context.Background(), a.timeoutFor(req))
	tab.cancel = cancel
	reg := a.reg // the command must not read App fields the UI goroutine replaces
	a.bus.Run(ctx, func(ctx context.Context) bus.Msg {
		defer cancel()
		resp, err := reg.Send(ctx, req)
		if err != nil {
			resp = request.Response{Err: err.Error()}
		}
		return sendResult{tab: tab, seq: seq, resp: resp}
	})
}

// cancelSend stops the running request of tab. Its late result, which
// arrives after this, belongs to an older send and is dropped.
func (a *App) cancelSend(tab *reqTab) {
	if !tab.Cancel() {
		return
	}
	tab.cancel()
	tab.view.SetLines(nil)
	a.status = request.ErrCancelled
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
	case sendResult:
		a.apply(msg)
	case wsMessage:
		a.handleWS(msg)
	case func(): // something a goroutine wants done on the UI goroutine, such as a chosen file
		msg()
	case bus.Failed: // a background task panicked; the app lives on
		a.status = msg.Err.Error()
	}
}

func (a *App) apply(result sendResult) {
	t := result.tab
	if !t.Accept(result.seq) {
		return // the answer to a send that was cancelled or replaced
	}
	t.Remember(t.Resp, a.now())
	t.Resp = &result.resp
	a.saveHistory(t)
	if owner, path, _ := a.holderOf(t.Req); path != "" {
		a.sent.Record(a, a.now(), owner.Dir, path, t.Req, result.resp.Status)
	}
	a.calls.Answer(a.now(), t.Req, &result.resp)
	a.rememberSchema(t, &result.resp)
	a.refreshView(t)
	if result.resp.Err != "" {
		a.status = "Request failed"
		a.runTests(t)
		return
	}
	a.status = fmt.Sprintf("%s %s: %d %s in %s", t.Req.Method, t.Req.URL, result.resp.Status, result.resp.StatusText, textfmt.Duration(result.resp.Duration))
	a.runTests(t)
}
