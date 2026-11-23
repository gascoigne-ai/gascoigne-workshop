package response

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths and height the panes are pinned at: the response pane of the test
// window, one so narrow that the toolbar's chips are squeezed, the main
// column of a 1280px window and the narrowest one the divider allows.
const (
	paneWideW   = 640
	paneNarrowW = 300
	paneH       = 400
	tableWideW  = 1020
	tableNarW   = 420
	noteH       = 300
)

// responseBody is the JSON every pinned state shows.
const responseBody = `{"id":1,"ids":[1]}`

// findWidgets is every node under root whose behavior has the Go type
// kind, in tree order.
func findWidgets(root Node, kind string) []Node {
	var found []Node
	if fmt.Sprintf("%T", root.Behavior()) == kind {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findWidgets(child, kind)...)
	}
	return found
}

// findWidget is the first node findWidgets finds, nil when there is none.
func findWidget(root Node, kind string) Node {
	if found := findWidgets(root, kind); len(found) > 0 {
		return found[0]
	}
	return nil
}

// assertDumpPinned lays root out in a window of width by height and
// compares its dump with testdata/name.dump.
func assertDumpPinned(t *testing.T, ctx *core.Ctx, root Node, name string, width, height int) {
	t.Helper()
	Layout(ctx, root, width, height)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Dump(root); got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// respond gives the open tab a response as a finished send would: the shown
// one goes into the history and the view is rebuilt.
func (world *world) respond(resp request.Response) {
	world.t.Helper()
	world.tab.Remember(world.tab.Resp, world.fake.now)
	world.tab.Resp = &resp
	world.pane.Refresh(world.tab)
}

// body is the Body pane of the open tab.
func (world *world) body() Node {
	return world.pane.View(world.ctx, world.tab, Body, &widget.ScrollState{})
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

// viewText is the text of the visible lines of the code view of the open tab.
func (world *world) viewText() string {
	view := world.pane.Code(world.tab)
	var text strings.Builder
	for _, i := range view.Visible() {
		for _, span := range view.Lines[i].Spans {
			text.WriteString(span.Text)
		}
		text.WriteString("\n")
	}
	return text.String()
}

// lineText is the text of one line.
func lineText(line widget.Line) string {
	var text strings.Builder
	for _, span := range line.Spans {
		text.WriteString(span.Text)
	}
	return text.String()
}

// compactBody has a nested match for "cy" and blocks that stay out of its way.
const compactBody = `{"total":3,"items":[` +
	`{"id":1,"name":"alpha","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"ann","site":"https://a.example.org/very/long/path/to/ann"}},` +
	`{"id":2,"name":"beta","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"bob","site":"https://b.example.org/very/long/path/to/bob"}},` +
	`{"id":3,"name":"gamma","tags":["cyan","magenta","teal","olive","navy","maroon","lime","aqua"],"owner":{"login":"cy","site":"https://c.example.org/very/long/path/to/cy"}}]}`
