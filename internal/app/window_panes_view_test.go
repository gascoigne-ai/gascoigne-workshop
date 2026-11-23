package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

// timelinePane is the response side's Timeline tab, a note pane.
var timelinePane = responsePanes[2]

// pinnedTimeline is a response whose every phase took time, so the note
// has all its lines; truncated adds the closing remark.
func pinnedTimeline(truncated bool) *request.Response {
	return &request.Response{
		Status: 200, StatusText: "OK", Duration: 412 * time.Millisecond, Size: 160430, Truncated: truncated,
		Timing: request.Timing{DNS: 3 * time.Millisecond, Connect: 11 * time.Millisecond, TLS: 48 * time.Millisecond,
			FirstByte: 290 * time.Millisecond, Transfer: 60 * time.Millisecond},
	}
}

// The note pane is pinned as text for a whole timeline and one that ends
// with the truncation remark, at two widths.
func TestTheNotePaneLaysOutAsPinned(t *testing.T) {
	states := []struct {
		name      string
		truncated bool
	}{
		{"timeline", false},
		{"truncated", true},
	}
	for _, state := range states {
		for _, size := range paneSizes {
			name := "notepane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				t.Cleanup(a.WaitWrites)
				tab := a.cur()
				tab.Resp = pinnedTimeline(state.truncated)
				var scroll widget.ScrollState
				assertDumpPinned(t, a, a.paneBody(timelinePane, tab, &scroll), name, size.width, settingsPaneH)
			})
		}
	}
}

// What Dump does not show of the note pane: it scrolls with the scroll
// state the pane is given.
func TestTheNotePaneScrollsWithTheGivenState(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	tab.Resp = pinnedTimeline(false)
	var scroll widget.ScrollState
	found := findWidget(a.paneBody(timelinePane, tab, &scroll), "*widget.scrollB")
	if found == nil {
		t.Fatal("the note pane has no scroll")
	}
	if settings, want := fmt.Sprintf("%+v", found.Behavior()), fmt.Sprintf("%p", &scroll); !strings.Contains(settings, want) {
		t.Fatalf("the scroll lacks the pane's state %s: %s", want, settings)
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
