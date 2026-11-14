package calltrace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// sizedHost is a window of any size.
type sizedHost struct{ width, height int }

func (host sizedHost) Size() (int, int) { return host.width, host.height }

// Window sizes the dialog is pinned at: tall enough for its full height,
// and so short that the window cuts it.
var (
	tallHost  = sizedHost{1000, 800}
	shortHost = sizedHost{700, 500}
)

func newCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// withEntries is a log with a sent request, its answer and a failure.
func withEntries() *Log {
	var log Log
	log.Info(clock, "→ GET https://one.test")
	log.Info(clock.Add(42*time.Millisecond), "← 200 OK in 42ms, 1.2KB")
	log.Error(clock.Add(time.Second), "✗ GET https://two.test failed: connection refused")
	return &log
}

// layOut builds the open dialog for host and lays it out in a window of
// host's size; the dialog floats, so a Stack stands for the window.
func layOut(t *testing.T, ctx *core.Ctx, log *Log, host sizedHost) *El {
	t.Helper()
	root := Stack(log.View(host))
	Layout(ctx, root, host.width, host.height)
	return root
}

// The laid-out dialog is pinned as text (testdata/*.dump, captured from the
// dialog built in Go), one case per state and window size.
func TestTheLogDialogLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name    string
		host    sizedHost
		prepare func() *Log
	}{
		{"empty-tall", tallHost, func() *Log { return &Log{} }},
		{"entries-tall", tallHost, withEntries},
		{"entries-short", shortHost, withEntries},
		{"filtered-short", shortHost, func() *Log {
			log := withEntries()
			log.SetFilter("two")
			return log
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			log := tc.prepare()
			log.Open()
			root := layOut(t, newCtx(t), log, tc.host)
			got := Dump(root)
			want, err := os.ReadFile(filepath.Join("testdata", tc.name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// findAll is every element under root, in tree order, whose own line of
// Dump has all the parts.
func findAll(root *El, parts ...string) []*El {
	var found []*El
	line, _, _ := strings.Cut(Dump(root), "\n")
	matches := true
	for _, part := range parts {
		matches = matches && strings.Contains(line, part)
	}
	if matches {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findAll(child, parts...)...)
	}
	return found
}

// findWidget is the first element under root whose behavior has the Go
// type kind.
func findWidget(root Node, kind string) Node {
	if fmt.Sprintf("%T", root.Behavior()) == kind {
		return root
	}
	for _, child := range root.Children() {
		if found := findWidget(child, kind); found != nil {
			return found
		}
	}
	return nil
}

// clickOn runs one frame that clicks the middle of the first element
// matching parts.
func clickOn(t *testing.T, ctx *core.Ctx, log *Log, parts ...string) {
	t.Helper()
	root := layOut(t, ctx, log, tallHost)
	found := findAll(root, parts...)
	if len(found) == 0 {
		t.Fatalf("nothing matches %q:\n%s", parts, Dump(root))
	}
	rect := found[0].Rect()
	ctx.In = core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	Update(ctx, root)
}

// Clear forgets every entry and leaves the dialog open.
func TestClickingClearEmptiesTheLog(t *testing.T) {
	log := withEntries()
	log.Open()
	clickOn(t, newCtx(t), log, `text="Clear"`)
	if len(log.Entries()) != 0 || !log.Shown() {
		t.Fatalf("entries %d, shown %v, want none and still open", len(log.Entries()), log.Shown())
	}
}

// Typing into the filter narrows the shown lines.
func TestTypingIntoTheFilterNarrowsTheLines(t *testing.T) {
	ctx := newCtx(t)
	log := withEntries()
	log.Open()
	clickOn(t, ctx, log, "fieldB")
	root := layOut(t, ctx, log, tallHost)
	ctx.In = core.Input{X: 5000, Y: 5000, Typed: []rune("two")}
	Update(ctx, root)
	if lines := log.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "two.test") {
		t.Fatalf("lines after typing %q", lines)
	}
}

// A click outside the dialog closes the log.
func TestAClickOutsideClosesTheLog(t *testing.T) {
	ctx := newCtx(t)
	log := withEntries()
	log.Open()
	root := layOut(t, ctx, log, tallHost)
	ctx.In = core.Input{X: 10, Y: 10, Click: true}
	Update(ctx, root)
	if log.Shown() {
		t.Fatal("a click outside left the log open")
	}
}

// What Dump does not print of the dialog: the filter's placeholder, its
// text menu and the state it edits, and the state of the code view with
// its lines and their colors.
func TestTheLogDialogsUnprintedParts(t *testing.T) {
	log := withEntries()
	log.Open()
	root := layOut(t, newCtx(t), log, tallHost)
	field := findWidget(root, "*widget.fieldB")
	if field == nil {
		t.Fatalf("no field:\n%s", Dump(root))
	}
	got := fmt.Sprintf("%+v", field.Behavior())
	for _, want := range []string{"placeholder:Filter", fmt.Sprintf("%p", &log.filter)} {
		if !strings.Contains(got, want) {
			t.Errorf("field %s, want %q in it", got, want)
		}
	}
	code := findWidget(root, "widget.codeB")
	if code == nil {
		t.Fatalf("no code view:\n%s", Dump(root))
	}
	if got := fmt.Sprintf("%+v", code.Behavior()); !strings.Contains(got, fmt.Sprintf("%p", &log.code)) {
		t.Errorf("code view %s, want the log's own state %p", got, &log.code)
	}
	want := []string{
		"12:00:00.000  |→ GET https://one.test",
		"12:00:00.042  |← 200 OK in 42ms, 1.2KB",
		"12:00:01.000  |✗ GET https://two.test failed: connection refused",
	}
	if len(log.code.Lines) != len(want) {
		t.Fatalf("%d lines, want %d", len(log.code.Lines), len(want))
	}
	for i, line := range log.code.Lines {
		got := line.Spans[0].Text + "|" + line.Spans[1].Text
		if got != want[i] || line.Spans[0].Style != core.Muted {
			t.Errorf("line %d = %q in %v, want %q in muted", i, got, line.Spans[0].Style, want[i])
		}
	}
	if log.code.Lines[0].Spans[1].Style != core.Text || log.code.Lines[2].Spans[1].Style != core.Error {
		t.Errorf("an info line is Text and a failure Error: %+v", log.code.Lines)
	}
}
