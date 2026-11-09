// Package tabs holds the state of one open request that does not depend on
// how it is drawn: the request and its saved copy, the shown response and
// the sending of it. The window keeps the widget state of the tab beside it.
package tabs

import (
	"reflect"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Tab is one open request.
type Tab struct {
	Req     *model.Request
	Saved   model.Request // the request as it was when opened or last saved
	Resp    *model.Response
	Sending bool
	Seq     int // numbers the sends, so a late result of an old one is ignored
}

// New opens req. A request from before body types gets one inferred.
func New(req *model.Request) *Tab {
	if req.BodyType == "" {
		req.BodyType = model.BodyNone
		if req.Body != "" {
			req.BodyType = model.BodyText
		}
	}
	return &Tab{Req: req, Saved: req.Clone()}
}

// Dirty reports whether the request differs from its saved copy.
func (tab *Tab) Dirty() bool { return !reflect.DeepEqual(tab.Saved, tab.Req.Clone()) }

// MarkSaved makes the request as it is the saved copy.
func (tab *Tab) MarkSaved() { tab.Saved = tab.Req.Clone() }

// BeginSend starts a send and returns its number; it does not start one
// while another is running.
func (tab *Tab) BeginSend() (seq int, started bool) {
	if tab.Sending {
		return 0, false
	}
	tab.Sending = true
	tab.Seq++
	return tab.Seq, true
}

// Accept reports whether the answer of send seq is still wanted, and if so
// ends sending. The answer of a send that was cancelled or replaced is not.
func (tab *Tab) Accept(seq int) bool {
	if seq != tab.Seq {
		return false
	}
	tab.Sending = false
	return true
}

// Cancel stops the running send, which makes its late answer an old one,
// and shows the cancellation as the response. It reports whether a send was
// running.
func (tab *Tab) Cancel() bool {
	if !tab.Sending {
		return false
	}
	tab.Sending = false
	tab.Seq++
	tab.Resp = &model.Response{Err: model.ErrCancelled}
	return true
}
