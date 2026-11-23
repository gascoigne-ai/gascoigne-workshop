package window

import (
	"bytes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
	setUp func(t *testing.T, app *Component, tab *reqTab)
}

// responseStates put the open tab in each look the response pane has:
// one per view mode, wrapped lines, the find bar open, earlier responses
// to pick from and an image body.
var responseStates = []responseState{
	{"compact", func(t *testing.T, app *Component, tab *reqTab) { app.resp.SetMode(tab.Tab, viewer.Compact) }},
	{"pretty", func(t *testing.T, app *Component, tab *reqTab) { app.resp.SetMode(tab.Tab, viewer.Pretty) }},
	{"raw", func(t *testing.T, app *Component, tab *reqTab) { app.resp.SetMode(tab.Tab, viewer.Raw) }},
	{"wrap", func(t *testing.T, app *Component, tab *reqTab) { app.resp.Code(tab.Tab).Wrap = true }},
	{"find", func(t *testing.T, app *Component, tab *reqTab) {
		app.resp.OpenFind()
		app.resp.Find(tab.Tab, "id")
	}},
	{"history", func(t *testing.T, app *Component, tab *reqTab) {
		answer(app, tab, request.Response{Status: 201, StatusText: "Created", Body: []byte(responseBody)})
	}},
	{"picture", func(t *testing.T, app *Component, tab *reqTab) {
		answer(app, tab, request.Response{Status: 200, StatusText: "OK", Body: pngBody(t)})
	}},
}

// answer gives tab a response as a finished send would.
func answer(app *Component, tab *reqTab, resp request.Response) {
	number, _ := app.sends.Begin(tab.Req, func() {})
	app.apply(sends.Result{Owner: tab.Req, Number: number, Response: resp})
}

// pngBody is a 40x20 PNG, an image response body. It is stored uncompressed
// because the compressed size differs between Go versions and the pinned
// layouts show the size.
func pngBody(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.NoCompression}
	if err := enc.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// responseIn is a test app whose open tab has responseBody in the look
// named state.
func responseIn(t *testing.T, state string) (*Component, *reqTab) {
	t.Helper()
	a := jsonResponse(t, responseBody)
	tab := a.cur()
	for _, candidate := range responseStates {
		if candidate.name == state {
			candidate.setUp(t, a, tab)
			a.Step(core.Input{})
			return a, tab
		}
	}
	t.Fatalf("no response state %q", state)
	return a, tab
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
	if !a.resp.Look(tab.Tab).Compact() {
		t.Fatalf("after Compact the mode is %v", a.resp.Look(tab.Tab).Mode)
	}
	click(a, wrapX, toolbarY)
	if !a.resp.Code(tab.Tab).Wrap {
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
