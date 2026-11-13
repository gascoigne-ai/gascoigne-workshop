package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the response pane is pinned at: about the pane in the test
// window, and one so narrow that the toolbar's chips are squeezed.
const (
	responseWideW   = 640
	responseNarrowW = 300
	responsePaneH   = 400
)

// responseBody is the JSON every pinned state shows.
const responseBody = `{"id":1,"ids":[1]}`

// Points of the response toolbar that app_flow_test.go has no name for:
// the Wrap chip, the History button left of Find, the Save button of a
// picture and a point inside the picture.
const (
	wrapX    = 1018
	historyX = 1121
	saveX    = 1240
	pictureX = 1000
	pictureY = 500
)

// responseState is a named look of the response pane and how to give
// the open tab that look.
type responseState struct {
	name  string
	setUp func(t *testing.T, a *App, tab *reqTab)
}

// responseStates put the open tab in each look the response pane has:
// one per view mode, wrapped lines, the find bar open, earlier responses
// to pick from and an image body.
var responseStates = []responseState{
	{"compact", func(t *testing.T, a *App, tab *reqTab) { a.setView(tab, viewer.Compact) }},
	{"pretty", func(t *testing.T, a *App, tab *reqTab) { a.setView(tab, viewer.Pretty) }},
	{"raw", func(t *testing.T, a *App, tab *reqTab) { a.setView(tab, viewer.Raw) }},
	{"wrap", func(t *testing.T, a *App, tab *reqTab) { a.toggleWrap(tab) }},
	{"find", func(t *testing.T, a *App, tab *reqTab) {
		a.openFind()
		tab.find.Insert("id")
		a.runFind(tab)
	}},
	{"history", func(t *testing.T, a *App, tab *reqTab) {
		answer(a, tab, model.Response{Status: 201, StatusText: "Created", Body: []byte(responseBody)})
	}},
	{"picture", func(t *testing.T, a *App, tab *reqTab) {
		answer(a, tab, model.Response{Status: 200, StatusText: "OK", Body: pngBody(t)})
	}},
}

// answer gives tab a response as a finished send would.
func answer(a *App, tab *reqTab, resp model.Response) {
	tab.Sending, tab.Seq = true, tab.Seq+1
	a.apply(sendResult{tab: tab, seq: tab.Seq, resp: resp})
}

// pngBody is a 40x20 PNG, an image response body.
func pngBody(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// responseIn is a test app whose open tab has responseBody in the look
// named state.
func responseIn(t *testing.T, state string) (*App, *reqTab) {
	t.Helper()
	a := jsonResponse(t, responseBody)
	tab := a.cur()
	for _, candidate := range responseStates {
		if candidate.name == state {
			candidate.setUp(t, a, tab)
			a.step(core.Input{})
			return a, tab
		}
	}
	t.Fatalf("no response state %q", state)
	return a, tab
}

// The response pane is pinned as text (element kinds, options, handlers,
// rectangles) at two widths in every look, so moving it to markup cannot
// change a pixel unnoticed.
func TestTheResponsePaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", responseWideW},
		{"narrow", responseNarrowW},
	}
	for _, state := range responseStates {
		for _, size := range sizes {
			name := "response-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, tab := responseIn(t, state.name)
				root := a.responsePane(responsePanes[0], tab, nil)
				Layout(a.ctx, root, size.width, responsePaneH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// Every button of the toolbar, and the picture, says what it does on
// hover.
func TestTheResponseToolbarsButtonsHaveTooltips(t *testing.T) {
	cases := []struct {
		name  string
		state string
		x, y  int
		tip   string
	}{
		{"history", "history", historyX, toolbarY, "Earlier responses of this request"},
		{"find", "compact", findBtnX, toolbarY, "Find in response (Ctrl+F)"},
		{"copy", "compact", copyBtnX, toolbarY, "Copy the response body"},
		{"curl", "compact", curlBtnX, toolbarY, "Copy the request as a cURL command"},
		{"save", "picture", saveX, toolbarY, "Save the image to a file"},
		{"picture", "picture", pictureX, pictureY, "The image, scaled down to fit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, _ := responseIn(t, tc.state)
			if got := tipAt(a, tc.x, tc.y); got != tc.tip {
				t.Fatalf("tooltip %q, want %q", got, tc.tip)
			}
		})
	}
}

// The Compact and Wrap chips switch the view, which Dump shows only as a
// handler.
func TestTheCompactAndWrapChipsSwitchTheView(t *testing.T) {
	a, tab := responseIn(t, "pretty")
	click(a, compactX, toolbarY)
	if !tab.look.Compact() {
		t.Fatalf("after Compact the mode is %v", tab.look.Mode)
	}
	click(a, wrapX, toolbarY)
	if !tab.view.Wrap {
		t.Fatal("Wrap did not wrap the lines")
	}
}

// History lists the earlier responses at the click.
func TestTheHistoryButtonListsEarlierResponses(t *testing.T) {
	a, _ := responseIn(t, "history")
	click(a, historyX, toolbarY)
	if widget.PopupMenu(a.ctx) == nil || !strings.HasPrefix(menuLabels(a)[0], "200") {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}

// Right-clicking the body opens the response's menu.
func TestRightClickingTheResponseBodyOpensItsMenu(t *testing.T) {
	a, _ := responseIn(t, "compact")
	rightClick(a, findFieldX, findBarY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[1] != "Copy body" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}

// The code view shows the tab's own view and the picture the tab's
// image: state Dump does not print, read from the widgets themselves.
func TestTheResponseWidgetsShowTheTabsState(t *testing.T) {
	a, tab := responseIn(t, "compact")
	code := findWidget(a.responsePane(responsePanes[0], tab, nil), "widget.codeB")
	if code == nil {
		t.Fatal("the response pane has no code view")
	}
	if got, want := fmt.Sprintf("%+v", code.Behavior()), fmt.Sprintf("view:%p", &tab.view); !strings.Contains(got, want) {
		t.Errorf("the code view lacks %q: %s", want, got)
	}
	a, tab = responseIn(t, "picture")
	picture := findWidget(a.responsePane(responsePanes[0], tab, nil), "widget.imageB")
	if picture == nil {
		t.Fatal("the picture pane has no image")
	}
	if got, want := fmt.Sprintf("%+v", picture.Behavior()), fmt.Sprintf("img:%p", tab.picture); !strings.Contains(got, want) {
		t.Errorf("the image lacks %q: %s", want, got)
	}
}
