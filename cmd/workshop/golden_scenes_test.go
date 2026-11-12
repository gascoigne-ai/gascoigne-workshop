//go:build !release

package main

import (
	"image"
	"io"
	"net"
	"net/http"
	"testing"
)

// goldenScene is one picture the program must keep drawing: the flags that
// put it on screen and the size of the virtual screen.
type goldenScene struct {
	name          string
	width, height int
	args          []string
	server        bool // serve the example root page on :8081, which the sample request calls
}

// Every scene is a golden under testdata/golden. Add a row, run with -update,
// look at the PNG, then commit both.
var goldenScenes = []goldenScene{
	{name: "default", width: shotWidth, height: shotHeight},
	{name: "response", width: shotWidth, height: shotHeight, args: []string{"-send"}, server: true},
	{name: "settings-menu", width: shotWidth, height: shotHeight, args: []string{"-menu"}},
	{name: "light-theme", width: shotWidth, height: shotHeight, args: []string{"-set", "theme=light"}},
	{name: "flat-high-contrast", width: shotWidth, height: shotHeight, args: []string{"-set", "flat=true", "-set", "contrast=high"}},
	{name: "side-by-side", width: shotWidth, height: shotHeight, args: []string{"-send", "-set", "layout=side"}, server: true},
	{name: "large-text", width: shotWidth, height: shotHeight, args: []string{"-set", "textSize=large"}},
	{name: "narrow", width: 800, height: 520, args: []string{"-window", "800x520"}},
}

func TestGoldenScenes(t *testing.T) {
	for _, scene := range goldenScenes {
		t.Run(scene.name, func(t *testing.T) {
			if scene.server {
				serveExampleRoot(t)
			}
			img := renderShotSized(t, scene.width, scene.height, scene.args...)
			if got, want := img.Bounds().Size(), (image.Point{X: scene.width, Y: scene.height}); got != want {
				t.Fatalf("%s is %v, want the %v the scene asks for", scene.name, got, want)
			}
			assertGolden(t, scene.name, img)
		})
	}
}

// exampleRoot is the page the example server answers at "/" (captured from
// `make example-server`).
const exampleRoot = "gascoigne example server\n/echo\n/status/{code}\n/delay/{ms}\n/auth\n"

// serveExampleRoot answers the sample request's address, so a scene that
// sends it shows a response and not "connection refused". A busy port skips
// the scene: another server there would draw something else.
func serveExampleRoot(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		t.Skipf("port 8081 is in use, the scene needs it free: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(writer, exampleRoot)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
}
