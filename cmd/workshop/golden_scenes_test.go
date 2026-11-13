//go:build !release

package main

import (
	"image"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

// goldenScene is one picture the program must keep drawing: the flags that
// put it on screen and the size of the virtual screen.
type goldenScene struct {
	name          string
	width, height int
	args          []string
	mask          []image.Rectangle // places that show a measured duration or the clock, which change from run to run
}

// Every scene is a golden under testdata/golden. Add a row, run with -update,
// look at the PNG, then commit both.
// The response header and the status bar print how long the request took; the
// status pill left of it moves with the number of digits, so the mask covers it.
// A sent log row ends with the time of day it was sent.
var (
	durationStacked = image.Rect(1100, 462, 1280, 497) // response pane under the request
	durationSide    = image.Rect(1100, 150, 1280, 185) // response pane beside the request
	durationStatus  = image.Rect(0, 772, 400, 800)     // status bar line
	sentTimeFirst   = image.Rect(156, 62, 236, 80)     // clock time of the sent log's first row, after its URL
	sentTimeSecond  = image.Rect(132, 99, 210, 117)    // clock time of the second row
)

var goldenScenes = []goldenScene{
	{name: "default", width: shotWidth, height: shotHeight},
	{name: "response", width: shotWidth, height: shotHeight, args: []string{"-send"}, mask: []image.Rectangle{durationStacked, durationStatus}},
	{name: "settings-menu", width: shotWidth, height: shotHeight, args: []string{"-menu"}},
	{name: "light-theme", width: shotWidth, height: shotHeight, args: []string{"-set", "theme=light"}},
	{name: "flat-high-contrast", width: shotWidth, height: shotHeight, args: []string{"-set", "flat=true", "-set", "contrast=high"}},
	{name: "side-by-side", width: shotWidth, height: shotHeight, args: []string{"-send", "-set", "layout=side"}, mask: []image.Rectangle{durationSide, durationStatus}},
	{name: "large-text", width: shotWidth, height: shotHeight, args: []string{"-set", "textSize=large"}},
	{name: "request-post", width: shotWidth, height: shotHeight, args: []string{"-request", "Echo"}},
	{name: "request-websocket", width: shotWidth, height: shotHeight, args: []string{"-request", "WebSocket Echo"}},
	{name: "request-sse", width: shotWidth, height: shotHeight, args: []string{"-request", "SSE Ticks"}},
	{name: "request-mcp", width: shotWidth, height: shotHeight, args: []string{"-request", "MCP Tools"}},
	{name: "request-grpc", width: shotWidth, height: shotHeight, args: []string{"-request", "gRPC Health"}},
	{name: "settings-colors", width: shotWidth, height: shotHeight, args: []string{"-page", "colors"}},
	{name: "settings-shortcuts", width: shotWidth, height: shotHeight, args: []string{"-page", "shortcuts"}},
	{name: "settings-requests", width: shotWidth, height: shotHeight, args: []string{"-page", "requests"}},
	{name: "settings-general", width: shotWidth, height: shotHeight, args: []string{"-page", "general"}},
	{name: "narrow", width: 800, height: 520, args: []string{"-window", "800x520"}},
	{name: "response-find", width: shotWidth, height: shotHeight, args: []string{"-send", "-find", "/e"}, mask: []image.Rectangle{durationStacked, durationStatus}},
	{name: "request-sending", width: shotWidth, height: shotHeight, args: []string{"-request", "Delay 500ms", "-sending"}},
	{name: "request-tabs", width: shotWidth, height: shotHeight, args: []string{"-open-all", "-type-url", "?page=2"}},
	{name: "request-params", width: shotWidth, height: shotHeight, args: []string{"-type-url", "?q=gpu-rendered-desktop-clients-written-in-go-with-custom-widgets-and-three-panes&page=2&sort=stars"}},
	{name: "tree-filter", width: shotWidth, height: shotHeight, args: []string{"-filter", "Echo"}},
	{name: "sent-log", width: shotWidth, height: shotHeight, args: []string{"-send", "-request", "Echo", "-sent"}, mask: []image.Rectangle{durationStacked, durationStatus, sentTimeFirst, sentTimeSecond}},
	{name: "response-raw", width: shotWidth, height: shotHeight, args: []string{"-send", "-set", "responseView=raw"}, mask: []image.Rectangle{durationStacked, durationStatus}},
}

func TestGoldenScenes(t *testing.T) {
	serveExampleRoot(t) // once for all scenes: they run in parallel
	for _, scene := range goldenScenes {
		t.Run(scene.name, func(t *testing.T) {
			t.Parallel()
			img := renderShotSized(t, scene.width, scene.height, scene.args...)
			if got, want := img.Bounds().Size(), (image.Point{X: scene.width, Y: scene.height}); got != want {
				t.Fatalf("%s is %v, want the %v the scene asks for", scene.name, got, want)
			}
			assertGolden(t, scene.name, img, scene.mask...)
		})
	}
}

// exampleRoot is the page the example server answers at "/" (captured from
// `make example-server`).
const exampleRoot = "gascoigne example server\n/echo\n/status/{code}\n/delay/{ms}\n/auth\n"

// serveExampleRoot answers the sample request's address, so a scene that
// sends it shows a response and not "connection refused". A delay request
// is held until its client goes away, so a scene that captures a request in
// flight never races its answer. A busy port skips the test: another server
// there would draw something else.
func serveExampleRoot(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		t.Skipf("port 8081 is in use, the scene needs it free: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/delay/") {
			<-request.Context().Done()
			return
		}
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(writer, exampleRoot)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
}
