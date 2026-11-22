package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Window sizes the whole window is pinned at: the test window and one so
// narrow that the response status gives way to the response tabs.
var windowSizes = []struct {
	name          string
	width, height int
}{
	{"wide", winW, winH},
	{"narrow", 640, 480},
}

// respondWith gives the open tab a response with the status, so the
// response pane shows its status pill.
func respondWith(status int, text string) func(app *App) {
	return func(app *App) {
		app.cur().Resp = &request.Response{Status: status, StatusText: text, Duration: 1400 * time.Millisecond, Size: 160434}
	}
}

// windowStates put the window in each look its root, main column and
// response status have: no response, each pill color, a failed request
// (no pill), and each overlay the root stacks over the frame.
var windowStates = []struct {
	name    string
	stacked bool
	setUp   func(app *App)
}{
	{"empty", false, func(*App) {}},
	{"empty-stacked", true, func(*App) {}},
	{"ok", false, respondWith(200, "OK")},
	{"ok-stacked", true, respondWith(200, "OK")},
	{"redirect", false, respondWith(301, "Moved Permanently")},
	{"not-found", false, respondWith(404, "Not Found")},
	{"failed", false, func(app *App) { app.cur().Resp = &request.Response{Err: "connection refused"} }},
	{"menu", false, func(app *App) { app.toggleSettings() }},
	{"popup", false, func(app *App) { widget.OpenMenu(app.ctx, 100, 100, []widget.MenuItem{{Label: "Copy"}}) }},
	{"dialog", false, func(app *App) { app.confirm("Delete Search Repos?", "This cannot be undone.", "Delete", func() {}) }},
	{"call-log", false, func(app *App) { app.calls.Open() }},
	{"toast", false, func(app *App) { app.toast("There is no response to save", severity.Error) }},
}

// The whole window is pinned in each state at both window sizes.
func TestTheWindowLaysOutAsPinned(t *testing.T) {
	for _, state := range windowStates {
		for _, size := range windowSizes {
			name := "window-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				layout := "side"
				if state.stacked {
					layout = "stacked"
				}
				a := testApp(t, withSettings(func(prefs *settings.Settings) { prefs.Layout = layout }))
				t.Cleanup(a.WaitWrites)
				a.w, a.h = size.width, size.height
				state.setUp(a)
				assertDumpPinned(t, a, a.view(), name, size.width, size.height)
			})
		}
	}
}

// The dump does not show a split's settings: the sidebar split keeps its
// width in pixels between its minimums, the pane split shares the room
// by ratio, stacked or side by side as the layout setting says.
func TestTheWindowSplitsAreWiredToTheApp(t *testing.T) {
	for _, layout := range []string{"side", "stacked"} {
		t.Run(layout, func(t *testing.T) {
			a := testApp(t, withSettings(func(prefs *settings.Settings) { prefs.Layout = layout }))
			t.Cleanup(a.WaitWrites)
			splits := findWidgets(a.view(), "*widget.splitB")
			if len(splits) != 2 {
				t.Fatalf("the window has %d splits, want 2", len(splits))
			}
			wants := [][]string{
				{fmt.Sprintf("state:%p", &a.sidebarSplit), "def:259 ", "vertical:false ratio:false", "minFirst:160 minSecond:420"},
				{fmt.Sprintf("state:%p", &a.paneSplit), "def:0.5 ", fmt.Sprintf("vertical:%t ratio:true", layout == "stacked"), "minFirst:220 minSecond:220"},
			}
			for i, split := range splits {
				settings := fmt.Sprintf("%+v", split.Behavior())
				for _, want := range wants[i] {
					if !strings.Contains(settings, want) {
						t.Errorf("split %d lacks %q: %s", i, want, settings)
					}
				}
			}
		})
	}
}
