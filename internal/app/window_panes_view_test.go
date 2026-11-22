package app

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
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
				t.Cleanup(a.waitWrites)
				tab := a.cur()
				tab.Resp = state.resp
				setSending(tab, state.send)
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
				t.Cleanup(a.waitWrites)
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
	t.Cleanup(a.waitWrites)
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

// bodyPaneStates are the looks of the body pane: each editor kind, and
// for JSON a valid, an invalid (a long parse error) and an empty body.
// File is last in the chip row, so at the narrow width it pins the row
// scrolled to keep the chosen chip in view.
var bodyPaneStates = []struct {
	name     string
	bodyType request.BodyType
	text     string
}{
	{"none", request.BodyNone, ""},
	{"json-valid", request.BodyJSON, "{\n  \"hello\": \"world\"\n}"},
	{"json-invalid", request.BodyJSON, `{"hello": "world", "list": [1, 2`},
	{"json-empty", request.BodyJSON, ""},
	{"text", request.BodyText, "hello"},
	{"form", request.BodyForm, ""},
	{"file", request.BodyFile, ""},
}

// openBodyType gives the open request a body of bodyType holding text.
func openBodyType(t *testing.T, bodyType request.BodyType, text string) (*App, *reqTab) {
	a := testApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	tab.body = widget.NewTextArea(text)
	a.setBodyType(tab, bodyType)
	return a, tab
}

// The body pane is pinned as text (element kinds, options, rects) for
// each state at two widths, so moving it to a template cannot change it
// unseen.
func TestTheBodyPaneLaysOutAsPinned(t *testing.T) {
	for _, state := range bodyPaneStates {
		for _, size := range paneSizes {
			name := "bodypane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, tab := openBodyType(t, state.bodyType, state.text)
				var scroll widget.ScrollState
				assertDumpPinned(t, a, a.paneBody(bodyPane, tab, &scroll), name, size.width, settingsPaneH)
			})
		}
	}
}

// What Dump does not show of the chip row: it scrolls sideways with the
// tab's own state.
func TestTheBodyTypeChipsScrollWithTheTabsState(t *testing.T) {
	a, tab := openBodyType(t, request.BodyJSON, "")
	var scroll widget.ScrollState
	chips := findWidget(a.paneBody(bodyPane, tab, &scroll), "*widget.hscrollB")
	if chips == nil {
		t.Fatal("the body pane has no sideways scroll")
	}
	if settings, want := fmt.Sprintf("%+v", chips.Behavior()), fmt.Sprintf("%p", &tab.bodyTypesX); !strings.Contains(settings, want) {
		t.Fatalf("the chip row lacks the tab's state %s: %s", want, settings)
	}
}

// What Dump does not show of the chip row: only the chosen type's chip is
// kept in view, so choosing a type elsewhere scrolls to it.
func TestOnlyTheChosenBodyTypeChipIsKeptInView(t *testing.T) {
	choices := []struct {
		bodyType request.BodyType
		chip     int // its place in the row: None, JSON, Text, Form, Multipart, GraphQL, File
	}{
		{request.BodyNone, 0},
		{request.BodyFile, 6},
	}
	for _, chosen := range choices {
		t.Run(string(chosen.bodyType), func(t *testing.T) {
			a, tab := openBodyType(t, chosen.bodyType, "")
			var scroll widget.ScrollState
			chips := findWidget(a.paneBody(bodyPane, tab, &scroll), "*widget.hscrollB")
			var kept []int
			for i, chip := range chips.Children() {
				if chip.KeepsInView() {
					kept = append(kept, i)
				}
			}
			if !slices.Equal(kept, []int{chosen.chip}) {
				t.Fatalf("chips kept in view %v, want only %d", kept, chosen.chip)
			}
		})
	}
}

// chipLabelled is the element whose only child is the text label, nil
// when there is none.
func chipLabelled(root Node, label string) Node {
	if kids := root.Children(); len(kids) == 1 && strings.HasPrefix(Dump(kids[0]), `textB text="`+label+`" `) {
		return root
	}
	for _, child := range root.Children() {
		if found := chipLabelled(child, label); found != nil {
			return found
		}
	}
	return nil
}

// The Wrap chip of the JSON and Text editors switches wrap for the tab's
// body, as the response's Wrap chip does for the response; the other
// editors have no long text lines to wrap and show no chip.
func TestTheBodyWrapChipWrapsTheTabsBodyEditor(t *testing.T) {
	for _, state := range bodyPaneStates {
		t.Run(state.name, func(t *testing.T) {
			a, tab := openBodyType(t, state.bodyType, state.text)
			var scroll widget.ScrollState
			root := a.paneBody(bodyPane, tab, &scroll)
			Layout(a.ctx, root, 600, settingsPaneH)
			chip := chipLabelled(root, "Wrap")
			editable := state.bodyType == request.BodyJSON || state.bodyType == request.BodyText
			if (chip != nil) != editable {
				t.Fatalf("Wrap chip shown %v, want %v", chip != nil, editable)
			}
			if !editable {
				return
			}
			at := chip.Rect()
			a.ctx.In = core.Input{X: at.X + at.W/2, Y: at.Y + at.H/2, Click: true}
			Update(a.ctx, root)
			if !tab.body.Wrap {
				t.Fatal("the Wrap chip left the body unwrapped")
			}
			root = a.paneBody(bodyPane, tab, &scroll)
			Layout(a.ctx, root, 600, settingsPaneH)
			a.ctx.In = core.Input{X: 5000, Y: 5000}
			Update(a.ctx, root)
			if !tab.body.Wrap {
				t.Fatal("the next frame's editor dropped the wrap the chip chose")
			}
		})
	}
}

// What Dump does not show of the text editors: JSON and Text edit the
// tab's body area, colored by a styler of their own.
func TestTheBodyAreaEditsTheTabsBody(t *testing.T) {
	for _, bodyType := range []request.BodyType{request.BodyJSON, request.BodyText} {
		t.Run(string(bodyType), func(t *testing.T) {
			a, tab := openBodyType(t, bodyType, "{}")
			var scroll widget.ScrollState
			areas := behaviorStates(a.paneBody(bodyPane, tab, &scroll), "*widget.areaB")
			if len(areas) != 1 {
				t.Fatalf("%d areas, want 1: %q", len(areas), areas)
			}
			if want := fmt.Sprintf("%p", &tab.body); !strings.Contains(areas[0], want) {
				t.Errorf("the area lacks the tab's body %s: %s", want, areas[0])
			}
			if tab.body.Styler == nil {
				t.Error("the body area has no styler")
			}
		})
	}
}
