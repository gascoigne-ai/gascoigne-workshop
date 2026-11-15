//go:build !release

package main

import (
	"bytes"
	"context"
	"image"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/exampleserver"
)

// goldenScene is one picture the program must keep drawing: the flags that
// put it on screen and the size of the virtual screen.
type goldenScene struct {
	name          string
	width, height int
	args          []string
	mask          []image.Rectangle // places that show a measured duration or the clock, which change from run to run
	// collections are folders under testdata opened beside the start
	// collection (-add-collection); each is copied to a temp dir first, so
	// nothing the program writes lands in testdata.
	collections []string
	// browse is a folder under testdata the in-app file browser opens on
	// (-browse), copied to a temp dir like collections.
	browse string
}

// Every scene is a golden under testdata/golden. Add a row, run with -update,
// look at the PNG, then commit both.
// The response header and the status bar print how long the request took; the
// status pill left of it moves with the number of digits, so the mask covers it.
// A sent log row ends with the time of day it was sent. The file browser's
// path field shows the temp folder it was opened on.
var (
	durationStacked = image.Rect(1100, 462, 1280, 497) // response pane under the request
	durationSide    = image.Rect(1100, 150, 1280, 185) // response pane beside the request
	durationStatus  = image.Rect(0, 772, 400, 800)     // status bar line
	sentTimeFirst   = image.Rect(156, 62, 236, 80)     // clock time of the sent log's first row, after its URL
	sentTimeSecond  = image.Rect(132, 99, 210, 117)    // clock time of the second row
	logStamps       = image.Rect(268, 204, 374, 240)   // clock times of the Log dialog's two lines, under its title bar and toolbar
	logDuration     = image.Rect(446, 224, 640, 240)   // duration and size of the Log dialog's answer line
	browserPath     = image.Rect(377, 191, 839, 217)   // inside the file browser's path field: the temp folder's absolute path and the caret after it
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
	{name: "settings-shortcuts-edited", width: shotWidth, height: shotHeight, args: []string{"-page", "shortcuts", "-record", "close-tab=Ctrl+L", "-record", "new-request=J", "-record", "duplicate-row=Backspace", "-record", "send=Ctrl+C"}},
	{name: "settings-requests", width: shotWidth, height: shotHeight, args: []string{"-page", "requests"}},
	{name: "settings-general", width: shotWidth, height: shotHeight, args: []string{"-page", "general"}},
	{name: "settings-json", width: shotWidth, height: shotHeight, args: []string{"-raw-settings"}},
	{name: "narrow", width: 800, height: 520, args: []string{"-window", "800x520"}},
	{name: "response-find", width: shotWidth, height: shotHeight, args: []string{"-send", "-find", "/e"}, mask: []image.Rectangle{durationStacked, durationStatus}},
	{name: "request-sending", width: shotWidth, height: shotHeight, args: []string{"-request", "Delay 500ms", "-sending"}},
	{name: "request-tabs", width: shotWidth, height: shotHeight, args: []string{"-open-all", "-type-url", "?page=2"}},
	{name: "request-params", width: shotWidth, height: shotHeight, args: []string{"-type-url", "?q=gpu-rendered-desktop-clients-written-in-go-with-custom-widgets-and-three-panes&page=2&sort=stars"}},
	{name: "tree-filter", width: shotWidth, height: shotHeight, args: []string{"-filter", "Echo"}},
	{name: "sent-log", width: shotWidth, height: shotHeight, args: []string{"-send", "-request", "Echo", "-sent"}, mask: []image.Rectangle{durationStacked, durationStatus, sentTimeFirst, sentTimeSecond}},
	{name: "request-settings", width: shotWidth, height: shotHeight, args: []string{"-pane", "Settings"}},
	{name: "request-body-json", width: shotWidth, height: shotHeight, args: []string{"-request", "Echo", "-pane", "Body"}},
	{name: "request-body-none", width: shotWidth, height: shotHeight, args: []string{"-pane", "Body"}},
	{name: "request-body-file", width: shotWidth, height: shotHeight, args: []string{"-request", "Echo", "-pane", "Body", "-body", "file"}},
	{name: "request-body-graphql", width: shotWidth, height: shotHeight, args: []string{"-request", "Echo", "-pane", "Body", "-body", "graphql"}},
	{name: "request-tests", width: shotWidth, height: shotHeight, args: []string{"-pane", "Tests", "-response-pane", "Tests"}},
	{name: "request-auth", width: shotWidth, height: shotHeight, args: []string{"-request", "Auth", "-pane", "Auth"}},
	{name: "toasts", width: shotWidth, height: shotHeight, args: []string{"-toast", "There is no response to save", "-toast", "Enter the gRPC server first, for example grpc://localhost:50051"}},
	{name: "dialog-rename", width: shotWidth, height: shotHeight, args: []string{"-dialog", "rename"}},
	{name: "dialog-delete", width: shotWidth, height: shotHeight, args: []string{"-dialog", "delete"}},
	{name: "call-log", width: shotWidth, height: shotHeight, args: []string{"-send", "-log"}, mask: []image.Rectangle{durationStacked, durationStatus, logStamps, logDuration}},
	{name: "mcp-tool-form", width: shotWidth, height: shotHeight, args: []string{"-request", "MCP Tools", "-send", "-tool", "add"}, mask: []image.Rectangle{durationStatus}},
	{name: "mcp-connected", width: shotWidth, height: shotHeight, args: []string{"-request", "MCP Tools", "-send"}, mask: []image.Rectangle{durationStatus}},
	{name: "response-raw", width: shotWidth, height: shotHeight, args: []string{"-send", "-set", "responseView=raw"}, mask: []image.Rectangle{durationStacked, durationStatus}},
	{name: "response-wrap", width: shotWidth, height: shotHeight, args: []string{"-request", "Index", "-type-url", "users/octocat", "-send", "-wrap", "-set", "responseView=raw"}, mask: []image.Rectangle{durationStacked, durationStatus}},
	{name: "request-body-wrap", width: shotWidth, height: shotHeight, args: []string{"-request", "Echo", "-pane", "Body", "-wrap", "-type-body", wrapBody}},
	// github-collection is what the store wrote for the GitHub sample collection
	{name: "tree-two-collections", width: shotWidth, height: shotHeight, collections: []string{"github-collection"}},
	{name: "tree-empty", width: shotWidth, height: shotHeight, args: []string{"-close-all"}},
	// the in-app file browser on a copy of the same folder
	{name: "file-browser", width: shotWidth, height: shotHeight, browse: "github-collection", mask: []image.Rectangle{browserPath}},
	// grpc-proto-collection names its .proto files relative to itself; the
	// first request is answered by the example server's port without
	// reflection, the second names a file cut off mid-message
	{name: "grpc-proto-files", width: shotWidth, height: shotHeight, args: []string{"-request", "Health from proto file", "-pane", "Proto", "-send"}, collections: []string{"grpc-proto-collection"}, mask: []image.Rectangle{durationStacked, durationStatus}},
	{name: "grpc-proto-error", width: shotWidth, height: shotHeight, args: []string{"-request", "Broken proto file", "-pane", "Proto"}, collections: []string{"grpc-proto-collection"}},
}

// wrapBody is a JSON body with one string longer than the body pane, so the
// request-body-wrap scene shows where the editor breaks it.
const wrapBody = `{
  "name": "The Octocat",
  "bio": "A long description that runs past the width of the body pane, so the editor has to break it before a word and keep going on the next visual row without a new line number",
  "public_repos": 8
}`

func TestGoldenScenes(t *testing.T) {
	serveExampleRoot(t) // once for all scenes: they run in parallel
	serveExampleServer(t)
	for _, scene := range goldenScenes {
		t.Run(scene.name, func(t *testing.T) {
			t.Parallel()
			args := append([]string(nil), scene.args...)
			for _, name := range scene.collections {
				args = append(args, "-add-collection", copyTestdata(t, name))
			}
			if scene.browse != "" {
				args = append(args, "-browse", copyTestdata(t, scene.browse))
			}
			img := renderShotSized(t, scene.width, scene.height, args...)
			if got, want := img.Bounds().Size(), (image.Point{X: scene.width, Y: scene.height}); got != want {
				t.Fatalf("%s is %v, want the %v the scene asks for", scene.name, got, want)
			}
			assertGolden(t, scene.name, img, scene.mask...)
		})
	}
}

// copyTestdata copies the folder name under testdata to a temp dir, so
// nothing the program writes lands in testdata.
func copyTestdata(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("testdata", name))); err != nil {
		t.Fatal(err)
	}
	return dir
}

// exampleRoot is the page the example server answers at "/" (captured from
// `make example-server`).
const exampleRoot = "gascoigne example server\n/echo\n/status/{code}\n/delay/{ms}\n/auth\n"

// serveExampleRoot answers the sample request's address, so a scene that
// sends it shows a response and not "connection refused". A delay request
// is held until its client goes away, so a scene that captures a request in
// flight never races its answer. A busy port skips the test: another server
// there would draw something else. /users/octocat answers with a captured
// GitHub user, one long line of JSON, for the scene that wraps a response.
func serveExampleRoot(t *testing.T) {
	t.Helper()
	octocat, err := os.ReadFile(filepath.Join("testdata", "octocat.json"))
	if err != nil {
		t.Fatal(err)
	}
	octocat = bytes.TrimSuffix(octocat, []byte("\n")) // the file ends in a newline, the API's minified answer does not
	listener, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		t.Skipf("port 8081 is in use, the scene needs it free: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/delay/") {
			<-request.Context().Done()
			return
		}
		if request.URL.Path == "/users/octocat" {
			writer.Header().Set("Content-Type", "application/json; charset=utf-8")
			writer.Write(octocat)
			return
		}
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(writer, exampleRoot)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
}

// serveExampleServer runs the example server's MCP endpoint and its gRPC
// port without reflection at the addresses the requests use, so a scene
// can show a connected server and the tools it announces, and a call made
// from a .proto file. The other protocols get free ports nobody calls.
func serveExampleServer(t *testing.T) {
	t.Helper()
	defaults := exampleserver.DefaultConfig()
	config := exampleserver.Config{Host: "127.0.0.1", MCP: defaults.MCP, GRPCNoReflection: defaults.GRPCNoReflection}
	endpoints, err := exampleserver.Listen(config)
	if err != nil {
		t.Skipf("the MCP or gRPC port is in use, the scenes need them free: %v", err)
	}
	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- exampleserver.Serve(ctx, endpoints) }()
	t.Cleanup(func() {
		stop()
		if err := <-served; err != nil {
			t.Errorf("example server: %v", err)
		}
	})
}
