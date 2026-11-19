package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The helpers shared by the app tests live here; a test file holds only
// what its own tests need.

// ---- the app under test ----------------------------------------------------

// appBuild is what testApp starts an App from.
type appBuild struct {
	opts    Options
	prefs   settings.Settings
	path    string // the settings file; "" is a new file in a temporary folder
	memory  bool   // settings kept in memory only
	written bool   // prefs are saved to the settings file before the start
	dir     string // the workspace folder
}

// appOption changes how testApp starts the App.
type appOption func(*appBuild)

// withSettings adjusts the settings the test app starts with.
func withSettings(adjust func(*settings.Settings)) appOption {
	return func(build *appBuild) { adjust(&build.prefs) }
}

// defaultLayout puts the layout setting back to its default.
func defaultLayout(prefs *settings.Settings) { prefs.Layout = settings.Default().Layout }

// withOptions starts the App with opts; the settings file and the
// workspace folder come from the other options.
func withOptions(opts Options) appOption {
	return func(build *appBuild) { build.opts = opts }
}

// inWorkspace opens the workspace folder dir.
func inWorkspace(dir string) appOption {
	return func(build *appBuild) { build.dir = dir }
}

// settingsAt starts from the settings file at path, as it is.
func settingsAt(path string) appOption {
	return func(build *appBuild) { build.path, build.written = path, false }
}

// untouchedSettings starts from a settings file that does not exist, so
// every setting has its default.
func untouchedSettings() appOption {
	return func(build *appBuild) { build.written = false }
}

// memorySettings keeps the settings in memory, with no file.
func memorySettings() appOption {
	return func(build *appBuild) { build.memory, build.written = true, false }
}

// testApp is a test app, after its first frame. Unless an option says
// otherwise it has the panes side by side, whose geometry the click
// coordinates of most tests assume, and the pretty-printed response,
// which most tests were written against, in a settings file of its own
// (a.SettingsPath()). The defaults have their own tests: defaultLayout
// for the layout, untouchedSettings for the response view.
func testApp(t *testing.T, options ...appOption) *App {
	t.Helper()
	build := appBuild{prefs: settings.Default(), written: true}
	build.prefs.ResponseView = "pretty"
	build.prefs.Layout = "side"
	for _, option := range options {
		option(&build)
	}
	if build.path == "" && !build.memory {
		build.path = filepath.Join(t.TempDir(), "settings.json")
	}
	if build.written {
		if err := settingsDisk.Save(build.path, build.prefs); err != nil {
			t.Fatal(err)
		}
	}
	build.opts.SettingsPath = build.path
	if build.dir != "" {
		build.opts.WorkspaceDir = build.dir
	}
	app, err := NewApp(withAdapters(build.opts))
	if err != nil {
		t.Fatal(err)
	}
	// A save is written in the background and the tabs read as saved at once,
	// so a test can end while the file is still being written, and its
	// temporary folder then cannot be removed ("directory not empty").
	t.Cleanup(app.waitWrites)
	app.w, app.h = winW, winH
	app.step(core.Input{}) // first frame: lay everything out once
	return app
}

// ---- pollers ---------------------------------------------------------------

// until steps the app until cond holds.
func until(t *testing.T, app *App, what string, cond func() bool) {
	t.Helper()
	for range 400 {
		app.step(core.Input{})
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// statusSoon steps the app until its status holds want.
func statusSoon(t *testing.T, app *App, want string) {
	t.Helper()
	until(t, app, fmt.Sprintf("a status holding %q", want), func() bool {
		return strings.Contains(app.status, want)
	})
}

// waitForResponse steps the app until tab has a response.
func waitForResponse(t *testing.T, app *App, tab *reqTab) {
	t.Helper()
	until(t, app, "a response", func() bool { return tab.Resp != nil })
}

// ---- dumps and the element tree -------------------------------------------

// assertDumpPinned lays root out in a window of width by height and
// compares its dump with testdata/name.dump.
func assertDumpPinned(t *testing.T, app *App, root Node, name string, width, height int) {
	t.Helper()
	Layout(app.ctx, root, width, height)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Dump(root); got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

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

// behaviorStates prints the behavior of every node under root whose
// behavior has the Go type kind, in order.
func behaviorStates(root Node, kind string) []string {
	var states []string
	for _, node := range findWidgets(root, kind) {
		states = append(states, fmt.Sprintf("%+v", node.Behavior()))
	}
	return states
}

// lineCenter is the center of the rect on a layout dump line.
func lineCenter(t *testing.T, line string) (x, y int) {
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

// dumpedLines are the lines of the window's layout dump that contain needle.
func dumpedLines(app *App, needle string) []string {
	var lines []string
	for _, line := range strings.Split(layoutDump(app), "\n") {
		if strings.Contains(line, needle) {
			lines = append(lines, line)
		}
	}
	return lines
}

// textCenter is the middle of the first element of the window that shows
// text exactly.
func textCenter(t *testing.T, app *App, text string) (x, y int) {
	t.Helper()
	lines := dumpedLines(app, fmt.Sprintf("text=%q", text))
	if len(lines) == 0 {
		t.Fatalf("no element with text %q", text)
	}
	return lineCenter(t, lines[0])
}

// clickDumped clicks the middle of the nth layout dump line that contains needle.
func clickDumped(t *testing.T, app *App, needle string, nth int) {
	t.Helper()
	app.step(core.Input{X: 5000, Y: 5000})
	lines := dumpedLines(app, needle)
	if nth >= len(lines) {
		t.Fatalf("no element with %s (#%d) in:\n%s", needle, nth, layoutDump(app))
	}
	x, y := lineCenter(t, lines[nth])
	click(app, x, y)
}

// ---- toasts and menus ------------------------------------------------------

// toastsWith are the toasts on the screen whose text contains part.
func toastsWith(app *App, part string) []widget.Toast {
	var found []widget.Toast
	for _, toast := range app.toasts {
		if strings.Contains(toast.Text, part) {
			found = append(found, toast)
		}
	}
	return found
}

// toastSays is true when a toast on the screen contains text.
func toastSays(app *App, text string) bool { return len(toastsWith(app, text)) > 0 }

// toastWith is true when a toast of the kind isErr contains text.
func toastWith(app *App, text string, isErr bool) bool {
	for _, toast := range toastsWith(app, text) {
		if toast.IsErr == isErr {
			return true
		}
	}
	return false
}

// errorToasts are the texts of the error toasts that contain part.
func errorToasts(app *App, part string) []string {
	var found []string
	for _, toast := range toastsWith(app, part) {
		if toast.IsErr {
			found = append(found, toast.Text)
		}
	}
	return found
}

// toastTexts is the text of every toast on the screen, one per line.
func toastTexts(app *App) string {
	var texts []string
	for _, toast := range app.toasts {
		texts = append(texts, toast.Text)
	}
	return strings.Join(texts, "\n")
}

// itemByLabel is the open menu's item called label.
func itemByLabel(t *testing.T, app *App, label string) *widget.MenuItem {
	t.Helper()
	items := widget.PopupMenu(app.ctx).Items
	for i := range items {
		if items[i].Label == label {
			return &items[i]
		}
	}
	t.Fatalf("no menu item %q in %v", label, menuLabels(app))
	return nil
}

// ---- input and files -------------------------------------------------------

// combo is a key press with the modifiers held.
func combo(key ebiten.Key, ctrl, shift, alt bool) core.Input {
	return core.Input{Keys: []ebiten.Key{key}, Ctrl: ctrl, Shift: shift, Alt: alt}
}

// fileText is the text of the file at path, empty when it cannot be read.
func fileText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// ---- sessions --------------------------------------------------------------

// fakeSession is an open session. It records what it is asked to send
// and to request, answers a request with result and err, and, when
// release is set, holds every send until release is closed.
type fakeSession struct {
	mu      sync.Mutex
	sent    []string
	asked   []string
	result  string
	err     error
	release chan struct{}
	closed  bool
}

func (session *fakeSession) Events() <-chan reqport.Event { return nil }

func (session *fakeSession) Close(context.Context) error {
	session.closed = true
	return nil
}

func (session *fakeSession) Ping(ctx context.Context) error { return session.Send(ctx, "ping") }

func (session *fakeSession) Send(_ context.Context, text string) error {
	if session.release != nil {
		<-session.release
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.sent = append(session.sent, text)
	return nil
}

func (session *fakeSession) Request(_ context.Context, text string) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.asked = append(session.asked, text)
	return session.result, session.err
}

// log is what was sent, in order.
func (session *fakeSession) log() []string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return append([]string(nil), session.sent...)
}
