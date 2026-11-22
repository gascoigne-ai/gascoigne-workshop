// Package apptest drives a whole app.App from outside package app, so tests
// of the window can live in e2e. It is test support in ordinary .go files:
// a _test.go file cannot be imported. It reaches the App only through its
// exported surface (Step, SetSize, WaitWrites, Tree), never a private field,
// so a helper that needs private state stays with the tests in package app.
//
// Why a subfolder: internal/app is capped in size (e2e/appsize), and this
// code is test support, not application. It sits in the app zone of the
// layer table (app may import everything); it imports adapters to give the
// App the real stores, as cmd/workshop does.
package apptest

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/app"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	historyadapter "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history/adapter"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The test window in logical pixels.
const (
	WinW, WinH = 1280, 800
	idleTries  = 400
	idleWait   = 5 * time.Millisecond
	retryDelay = 10 * time.Millisecond
)

// build is what Start starts an App from.
type build struct {
	opts    app.Options
	prefs   settings.Settings
	path    string // the settings file; "" is a new file in a temporary folder
	memory  bool   // settings kept in memory only
	written bool   // prefs are saved to the settings file before the start
	dir     string // the workspace folder
}

// Option changes how Start starts the App.
type Option func(*build)

// WithSettings adjusts the settings the test app starts with.
func WithSettings(adjust func(*settings.Settings)) Option {
	return func(started *build) { adjust(&started.prefs) }
}

// DefaultLayout puts the layout setting back to its default.
func DefaultLayout(prefs *settings.Settings) { prefs.Layout = settings.Default().Layout }

// WithOptions starts the App with opts; the settings file and the
// workspace folder still come from the other options.
func WithOptions(opts app.Options) Option {
	return func(started *build) { started.opts = opts }
}

// InWorkspace opens the workspace folder dir.
func InWorkspace(dir string) Option {
	return func(started *build) { started.dir = dir }
}

// SettingsAt starts from the settings file at path, as it is.
func SettingsAt(path string) Option {
	return func(started *build) { started.path, started.written = path, false }
}

// UntouchedSettings starts from a settings file that does not exist, so
// every setting has its default.
func UntouchedSettings() Option {
	return func(started *build) { started.written = false }
}

// MemorySettings keeps the settings in memory, with no file.
func MemorySettings() Option {
	return func(started *build) { started.memory, started.written = true, false }
}

// Start is a test app after its first frame, in a WinW by WinH window.
// Unless an option says otherwise it has the sample collection, the
// "pretty" response view and the side-by-side layout, and its settings are
// saved to a new file in a temporary folder.
//
// Start redirects HOME and every XDG_* folder to the test's temporary
// folder with t.Setenv, so nothing reads or writes the user's files. That
// makes Start unusable in a test that calls t.Parallel (t.Setenv panics).
func Start(t testing.TB, options ...Option) *app.App {
	t.Helper()
	sandbox(t)
	started := build{prefs: settings.Default(), written: true}
	started.prefs.ResponseView = "pretty"
	started.prefs.Layout = "side"
	for _, option := range options {
		option(&started)
	}
	if started.path == "" && !started.memory {
		started.path = filepath.Join(t.TempDir(), "settings.json")
	}
	if started.written {
		if err := (settingsfile.Files{}).Save(started.path, started.prefs); err != nil {
			t.Fatal(err)
		}
	}
	started.opts.SettingsPath = started.path
	if started.dir != "" {
		started.opts.WorkspaceDir = started.dir
	}
	a, err := app.NewApp(WithAdapters(started.opts))
	if err != nil {
		t.Fatal(err)
	}
	// A save is written in the background and the tabs read as saved at once,
	// so a test that reads the file or ends its folder must wait for it.
	t.Cleanup(a.WaitWrites)
	a.SetSize(WinW, WinH)
	a.Step(core.Input{}) // first frame: lay everything out once
	return a
}

// sandbox points the folders an App may read or write at the test's own.
func sandbox(t testing.TB) {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		folder := filepath.Join(root, strings.ToLower(name))
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv(name, folder)
	}
}

// WithAdapters gives opts the real history and font adapters, as main does.
func WithAdapters(opts app.Options) app.Options {
	opts.HostOS = cmp.Or(opts.HostOS, "linux") // the same shortcut text on every machine
	opts.RetryDelay = cmp.Or(opts.RetryDelay, retryDelay)
	opts.History = func(dir string) history.History { return historyadapter.Dir(dir) }
	opts.Fonts = fonts.System{}
	opts.Protocols = proto.Configured
	opts.GRPCMethods = proto.ListGRPCMethods
	opts.GRPCSchema = proto.GRPCInputSchema
	opts.Curl = proto.Curl
	opts.Store = store.Files{}
	opts.States = statefile.Files{}
	opts.SettingsFiles = settingsfile.Files{}
	return opts
}

// Step runs one frame with in.
func Step(a *app.App, in core.Input) { a.Step(in) }

// Click is a press followed by a release at the same place.
func Click(a *app.App, posX, posY int) {
	a.Step(core.Input{X: posX, Y: posY, Click: true, Held: true})
	a.Step(core.Input{X: posX, Y: posY})
}

// RightClick presses the right button at the place.
func RightClick(a *app.App, posX, posY int) {
	a.Step(core.Input{X: posX, Y: posY, RightClick: true})
}

// Drag presses at fromX, moves to toX and releases, all on row posY.
func Drag(a *app.App, fromX, toX, posY int) {
	a.Step(core.Input{X: fromX, Y: posY, Click: true, Held: true})
	a.Step(core.Input{X: toX, Y: posY, Held: true})
	a.Step(core.Input{X: toX, Y: posY}) // release
}

// Type types text in one frame.
func Type(a *app.App, text string) { a.Step(core.Input{Typed: []rune(text)}) }

// Combo is a key press with the modifiers held.
func Combo(key ebiten.Key, ctrl, shift, alt bool) core.Input {
	return core.Input{Keys: []ebiten.Key{key}, Ctrl: ctrl, Shift: shift, Alt: alt}
}

// Until steps the app until cond holds.
func Until(t testing.TB, a *app.App, what string, cond func() bool) {
	t.Helper()
	for range idleTries {
		a.Step(core.Input{})
		if cond() {
			return
		}
		time.Sleep(idleWait)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// Dump is the window as text with every element's rectangle. It lays out
// the tree the next Draw paints (so it moves the app's builds counter).
func Dump(a *app.App) string { return view.Dump(a.Tree()) }

// Lines are the lines of the window's dump that contain needle.
func Lines(a *app.App, needle string) []string {
	var found []string
	for _, line := range strings.Split(Dump(a), "\n") {
		if strings.Contains(line, needle) {
			found = append(found, line)
		}
	}
	return found
}

var rectPattern = regexp.MustCompile(`rect=(-?\d+),(-?\d+),(\d+),(\d+)`)

// LineCenter is the center of the rect on a dump line.
func LineCenter(t testing.TB, line string) (x, y int) {
	t.Helper()
	match := rectPattern.FindStringSubmatch(line)
	if match == nil {
		t.Fatalf("no rectangle on %q", line)
	}
	var left, top, width, height int
	fmt.Sscan(match[1], &left)
	fmt.Sscan(match[2], &top)
	fmt.Sscan(match[3], &width)
	fmt.Sscan(match[4], &height)
	return left + width/2, top + height/2
}

// TextCenter is the middle of the first element of the window that shows
// text exactly.
func TextCenter(t testing.TB, a *app.App, text string) (x, y int) {
	t.Helper()
	lines := Lines(a, fmt.Sprintf("text=%q", text))
	if len(lines) == 0 {
		t.Fatalf("no element with text %q", text)
	}
	return LineCenter(t, lines[0])
}

// ClickLine clicks the middle of the nth (from 0) dump line that contains
// needle. It first moves the pointer away so a hover does not change the
// layout under the click.
func ClickLine(t testing.TB, a *app.App, needle string, nth int) {
	t.Helper()
	a.Step(core.Input{X: 5000, Y: 5000})
	lines := Lines(a, needle)
	if nth >= len(lines) {
		t.Fatalf("no element with %s (#%d) in:\n%s", needle, nth, Dump(a))
	}
	x, y := LineCenter(t, lines[nth])
	Click(a, x, y)
}

// AssertPinned lays the window out at width by height and compares its
// dump with the golden file at path. The settings file's path, a temp
// folder that differs every run, reads as /config/settings.json. It dumps
// Tree, which rebuilds only what a Step marked stale: run a Step after a
// change made outside the frame loop.
func AssertPinned(t testing.TB, a *app.App, path string, width, height int) {
	t.Helper()
	a.SetSize(width, height)
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.ReplaceAll(Dump(a), a.SettingsPath(), "/config/settings.json")
	if got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// FindWidgets is every node under root whose behavior has the Go type
// kind (as %T prints it), in tree order.
func FindWidgets(root view.Node, kind string) []view.Node {
	var found []view.Node
	if fmt.Sprintf("%T", root.Behavior()) == kind {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, FindWidgets(child, kind)...)
	}
	return found
}

// FindWidget is the first node FindWidgets finds, nil when there is none.
func FindWidget(root view.Node, kind string) view.Node {
	if found := FindWidgets(root, kind); len(found) > 0 {
		return found[0]
	}
	return nil
}

// BehaviorStates prints the behavior of every node FindWidgets finds.
func BehaviorStates(root view.Node, kind string) []string {
	var states []string
	for _, node := range FindWidgets(root, kind) {
		states = append(states, fmt.Sprintf("%+v", node.Behavior()))
	}
	return states
}

// FileText is the text of the file at path, empty when it cannot be read.
func FileText(t testing.TB, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// Ran records which menu items ran.
type Ran struct{ Log []string }

// Item is a menu item called label that records itself in Log when run.
func (ran *Ran) Item(label string) widget.MenuItem {
	return widget.MenuItem{Label: label, Run: func() { ran.Log = append(ran.Log, label) }}
}

// Session is an open session that records what it is asked to send. Set
// Release to hold Send until the channel is closed.
type Session struct {
	mu      sync.Mutex
	sent    []string
	asked   []string
	Result  string
	Err     error
	Release chan struct{}
	Closed  bool
}

func (session *Session) Events() <-chan request.Event { return nil }

func (session *Session) Close(context.Context) error {
	session.Closed = true
	return nil
}

func (session *Session) Ping(ctx context.Context) error { return session.Send(ctx, "ping") }

func (session *Session) Send(_ context.Context, text string) error {
	if session.Release != nil {
		<-session.Release
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.sent = append(session.sent, text)
	return nil
}

func (session *Session) Request(_ context.Context, text string) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.asked = append(session.asked, text)
	return session.Result, session.Err
}

// Log is what was sent, in order.
func (session *Session) Log() []string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return append([]string(nil), session.sent...)
}

// JSONServer answers every request with a small JSON body.
func JSONServer(t testing.TB) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		io.WriteString(writer, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// PathServer serves /fast at once, /redirect as a redirect to /fast and
// /slow only when released (or when the request is cancelled).
func PathServer(t testing.TB) (srv *httptest.Server, release func()) {
	t.Helper()
	gate := make(chan struct{})
	var once sync.Once
	srv = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		switch call.URL.Path {
		case "/slow":
			select {
			case <-gate:
				io.WriteString(writer, "slow")
			case <-call.Context().Done():
			}
		case "/redirect":
			http.Redirect(writer, call, "/fast", http.StatusFound)
		default:
			io.WriteString(writer, "fast")
		}
	}))
	release = func() { once.Do(func() { close(gate) }) }
	t.Cleanup(func() { release(); srv.Close() })
	return srv, release
}
