// Package tabs holds the state of one open request that does not depend on
// how it is drawn: the request and its saved copy, the shown response and
// the sending of it. The window keeps the widget state of the tab beside it.
package tabs

import (
	"reflect"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

// Tab is one open request.
type Tab struct {
	Req       *request.Request
	Saved     request.Request // the request as it was when opened or last saved
	Resp      *request.Response
	Earlier   []PastResponse         // earlier responses, newest first
	histories map[string]*RowHistory // undo history of each table
	Send
	Connection
}

// New opens req. A request from before body types gets one inferred.
func New(req *request.Request) *Tab {
	if req.BodyType == "" {
		req.BodyType = request.BodyNone
		if req.Body != "" {
			req.BodyType = request.BodyText
		}
	}
	return &Tab{Req: req, Saved: req.Clone()}
}

// Dirty reports whether the request differs from its saved copy.
func (tab *Tab) Dirty() bool { return !reflect.DeepEqual(tab.Saved, tab.Req.Clone()) }

// MarkSaved makes the request as it is the saved copy.
func (tab *Tab) MarkSaved() { tab.Saved = tab.Req.Clone() }

// Cancel stops the running send, which makes its late answer an old one,
// and shows the cancellation as the response. It reports whether a send was
// running.
func (tab *Tab) Cancel() bool {
	if !tab.cancelSend() {
		return false
	}
	tab.Resp = &request.Response{Err: request.ErrCancelled}
	return true
}

// Next is the tab step places away from active among count tabs, wrapping
// at the ends.
func Next(active, count, step int) int { return (active + step + count) % count }

// AfterClose is the active tab once tab closed is gone and remaining tabs
// are left: the same tab when it is still there, else its neighbour.
func AfterClose(active, closed, remaining int) int {
	switch {
	case remaining == 0:
		return 0
	case closed < active:
		return active - 1
	case active >= remaining:
		return remaining - 1
	}
	return active
}

// AfterMove is the active tab once tab from has moved to place to: the tab
// that was active stays the active one.
func AfterMove(active, from, to int) int {
	switch {
	case active == from:
		return to
	case from < active && active <= to:
		return active - 1
	case to <= active && active < from:
		return active + 1
	}
	return active
}
