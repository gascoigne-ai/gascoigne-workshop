package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// paneSizes are the widths the panes are pinned at: the request pane's
// width in the test window and the narrowest main area.
var paneSizes = []struct {
	name  string
	width int
}{
	{"wide", settingsWideW},
	{"narrow", settingsNarrowW},
}

// messagePaneStates are the status lines a response pane shows in place
// of its content: not sent yet, in flight, and a failed send in red.
var messagePaneStates = []struct {
	name string
	resp *request.Response
	send bool
}{
	{"not-sent", nil, false},
	{"sending", nil, true},
	{"failed", &request.Response{Err: `Get "http://127.0.0.1:1/": dial tcp 127.0.0.1:1: connect: connection refused`}, false},
}

// The message pane is pinned as text (element kinds, options, rects) for
// each status at two widths, so moving it to a template cannot change it
// unseen.
func TestTheMessagePaneLaysOutAsPinned(t *testing.T) {
	for _, state := range messagePaneStates {
		for _, size := range paneSizes {
			name := "messagepane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				t.Cleanup(a.WaitWrites)
				tab := a.cur()
				tab.Resp = state.resp
				setSending(a, tab, state.send)
				var scroll widget.ScrollState
				assertDumpPinned(t, a, a.paneBody(responsePanes[0], tab, &scroll), name, size.width, settingsPaneH)
			})
		}
	}
}

// openBodyType gives the open request a body of bodyType holding text.
func openBodyType(t *testing.T, bodyType request.BodyType, text string) (*App, *reqTab) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	*a.body.Area(tab.Tab) = widget.NewTextArea(text)
	a.body.ChooseType(tab.Tab, bodyType)
	return a, tab
}

// The Form look of the body pane is pinned here, with the real table; the
// other looks are pinned by ui/editor/body. Pinned as text (element kinds,
// options, rects) at two widths.
func TestTheFormBodyPaneLaysOutAsPinned(t *testing.T) {
	for _, size := range paneSizes {
		name := "bodypane-form-" + size.name
		t.Run(name, func(t *testing.T) {
			a, tab := openBodyType(t, request.BodyForm, "")
			var scroll widget.ScrollState
			assertDumpPinned(t, a, a.paneBody(bodyPane, tab, &scroll), name, size.width, settingsPaneH)
		})
	}
}
