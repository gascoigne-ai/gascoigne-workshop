package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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
	resp *model.Response
	send bool
}{
	{"not-sent", nil, false},
	{"sending", nil, true},
	{"failed", &model.Response{Err: `Get "http://127.0.0.1:1/": dial tcp 127.0.0.1:1: connect: connection refused`}, false},
}

// The message pane is pinned as text (element kinds, options, rects) for
// each status at two widths, so moving it to a template cannot change it
// unseen.
func TestTheMessagePaneLaysOutAsPinned(t *testing.T) {
	for _, state := range messagePaneStates {
		for _, size := range paneSizes {
			name := "messagepane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				tab := a.cur()
				tab.Resp, tab.Sending = state.resp, state.send
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
func pinnedTimeline(truncated bool) *model.Response {
	return &model.Response{
		Status: 200, StatusText: "OK", Duration: 412 * time.Millisecond, Size: 160430, Truncated: truncated,
		Timing: model.Timing{DNS: 3 * time.Millisecond, Connect: 11 * time.Millisecond, TLS: 48 * time.Millisecond,
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
				a, _ := newTestApp(t)
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
	a, _ := newTestApp(t)
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
	name, bodyType, text string
}{
	{"none", model.BodyNone, ""},
	{"json-valid", model.BodyJSON, "{\n  \"hello\": \"world\"\n}"},
	{"json-invalid", model.BodyJSON, `{"hello": "world", "list": [1, 2`},
	{"json-empty", model.BodyJSON, ""},
	{"text", model.BodyText, "hello"},
	{"form", model.BodyForm, ""},
	{"file", model.BodyFile, ""},
}

// openBodyType gives the open request a body of bodyType holding text.
func openBodyType(t *testing.T, bodyType, text string) (*App, *reqTab) {
	a, _ := newTestApp(t)
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
	a, tab := openBodyType(t, model.BodyJSON, "")
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
		bodyType string
		chip     int // its place in the row: None, JSON, Text, Form, Multipart, GraphQL, File
	}{
		{model.BodyNone, 0},
		{model.BodyFile, 6},
	}
	for _, chosen := range choices {
		t.Run(chosen.bodyType, func(t *testing.T) {
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

// What Dump does not show of the text editors: JSON and Text edit the
// tab's body area, colored by a styler of their own.
func TestTheBodyAreaEditsTheTabsBody(t *testing.T) {
	for _, bodyType := range []string{model.BodyJSON, model.BodyText} {
		t.Run(bodyType, func(t *testing.T) {
			a, tab := openBodyType(t, bodyType, "{}")
			var scroll widget.ScrollState
			areas := areaStates(a.paneBody(bodyPane, tab, &scroll))
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
