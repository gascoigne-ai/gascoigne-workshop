package app

import (
	"context"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"time"
)

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
	a.calls.Sent(a.now(), req)
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
	tab.view.SetLines(nil)
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
	a.calls.Answer(a.now(), t.Req, &result.Response)
	a.rememberSchema(t, &result.Response)
	a.refreshView(t)
	if result.Response.Err != "" {
		a.notices.SetStatus("Request failed")
		a.runTests(t)
		return
	}
	a.notices.SetStatus(fmt.Sprintf("%s %s: %d %s in %s", t.Req.Method, t.Req.URL, result.Response.Status, result.Response.StatusText, format.Duration(result.Response.Duration)))
	a.runTests(t)
}
