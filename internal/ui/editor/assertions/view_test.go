package assertions

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// assertSizes are the widths the two Tests panes are pinned at: the
// request pane of the test window and the narrowest main area.
var assertSizes = []struct {
	name  string
	width int
}{
	{"wide", wideW},
	{"narrow", narrowW},
}

// The request's Tests pane is pinned as text (element kinds, options,
// rects) at two widths, so moving it to a template cannot change it unseen.
func TestTheAssertionsPaneLaysOutAsPinned(t *testing.T) {
	for _, size := range assertSizes {
		name := "assertions-" + size.name
		t.Run(name, func(t *testing.T) {
			world := newWorld(t, &request.Request{Name: "R"})
			assertDumpPinned(t, world.ctx, world.pane.View(world.ctx, world.tab, &widget.ScrollState{}), name, size.width, paneH)
		})
	}
}

// What Dump does not show of the assertions area: it edits the tab's own
// tests area.
func TestTheAssertionsAreaEditsTheTabsTests(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R"})
	areas := behaviorStates(world.pane.View(world.ctx, world.tab, &widget.ScrollState{}), "*widget.areaB")
	if want := &world.pane.state(world.tab).tests; len(areas) != 1 || !strings.Contains(areas[0], fmt.Sprintf("%p", want)) {
		t.Fatalf("areas %q, want one editing %p", areas, want)
	}
}

// dumpedText is the quoted text on an element's own Dump line.
var dumpedText = regexp.MustCompile(`text=("(?:[^"\\]|\\.)*")`)

// The help above the assertions used to be one line cut off at the pane's
// edge; in the narrowest pane it now takes several lines that all fit.
func TestTheAssertionsHelpWrapsInsideANarrowPane(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R"})
	root := world.pane.View(world.ctx, world.tab, &widget.ScrollState{})
	Layout(world.ctx, root, narrowW, paneH)
	pane := root.Rect()
	lines := 0
	var walk func(element *El)
	walk = func(element *El) {
		for _, child := range element.Children() {
			walk(child)
		}
		line, _, _ := strings.Cut(Dump(element), "\n")
		match := dumpedText.FindStringSubmatch(line)
		if !strings.HasPrefix(strings.TrimSpace(line), "textB ") || match == nil {
			return
		}
		text, err := strconv.Unquote(match[1])
		if err != nil {
			t.Fatal(err)
		}
		face := world.ctx.F.UI
		if strings.Contains(line, "small=true") {
			face = world.ctx.F.Small
		}
		lines++
		rect := element.Rect()
		if width := world.ctx.Width(face, text); width > rect.W || rect.X < pane.X || rect.X+rect.W > pane.X+pane.W {
			t.Errorf("help line %q is %d px wide in a %d px rect at x=%d, pane %d px", text, width, rect.W, rect.X, pane.W)
		}
	}
	walk(root)
	if lines < 2 {
		t.Fatalf("the help is %d line(s), want several:\n%s", lines, Dump(root))
	}
}

// The wrapped help took five full-size lines and left the editor 158 of
// its 254 px in the narrowest pane; small lines give most of it back.
func TestTheAssertionsHelpLeavesTheEditorMostOfTheNarrowPane(t *testing.T) {
	const minEditorH = 190
	world := newWorld(t, &request.Request{Name: "R"})
	root := world.pane.View(world.ctx, world.tab, &widget.ScrollState{})
	Layout(world.ctx, root, narrowW, paneH)
	children := root.Children()
	editor := children[len(children)-1]
	if editor.Rect().H < minEditorH {
		t.Fatalf("the editor is %d px high, want at least %d:\n%s", editor.Rect().H, minEditorH, Dump(root))
	}
}

// pinnedResponse is the response the pinned results are checked against.
var pinnedResponse = &request.Response{Status: 200, StatusText: "OK", Body: []byte(`{"items":[{"name":"a"}]}`)}

// resultsPaneStates are the looks of the response's Tests pane: no
// assertions, all passed, and passes mixed with failures and a line that
// does not parse.
var resultsPaneStates = []struct {
	name  string
	tests string
}{
	{"none", ""},
	{"passed", "status == 200\njson items.length > 0"},
	{"mixed", "status == 200\nstatus == 404\njson items[0].name == \"b\"\nnot an assertion"},
}

// The response's Tests pane is pinned as text for each state at two
// widths, so moving it to a template cannot change it unseen.
func TestTheResultsPaneLaysOutAsPinned(t *testing.T) {
	for _, state := range resultsPaneStates {
		for _, size := range assertSizes {
			name := "results-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t, &request.Request{Name: "R"})
				world.pane.state(world.tab).results = assert.Run(state.tests, pinnedResponse)
				assertDumpPinned(t, world.ctx, world.pane.ResultsView(world.ctx, world.tab, &widget.ScrollState{}), name, size.width, paneH)
			})
		}
	}
}

// What Dump does not show of the results list: it scrolls with the scroll
// the host hands over, the tab's own response scroll.
func TestTheResultsListScrollsWithTheGivenScroll(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Tests: "status == 200"})
	world.tab.Resp = pinnedResponse
	world.pane.Run(world.tab)
	var scroll widget.ScrollState
	root := world.pane.ResultsView(world.ctx, world.tab, &scroll)
	if got := fmt.Sprintf("%+v", root.Behavior()); !strings.Contains(got, fmt.Sprintf("%p", &scroll)) {
		t.Fatalf("scroll %s, want the state %p", got, &scroll)
	}
}

// The request's Settings pane is pinned as text (element kinds, options,
// rects) in two states: the defaults and a settings file with another
// default timeout.
func TestTheRequestSettingsPaneLaysOutAsPinned(t *testing.T) {
	states := []struct {
		name  string
		setUp func(world *world)
	}{
		{"defaults", func(world *world) {}},
		{"default-timeout", func(world *world) { world.fake.settings.TimeoutSec = 45 }},
	}
	for _, state := range states {
		for _, size := range assertSizes {
			name := "settingspane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t, &request.Request{Name: "R"})
				state.setUp(world)
				assertDumpPinned(t, world.ctx, world.pane.SettingsView(world.ctx, world.tab, &widget.ScrollState{}), name, size.width, paneH)
			})
		}
	}
}

// What Dump does not show of the timeout field: it edits the tab's timeout,
// not a copy, and has neither a placeholder nor an Enter action.
func TestTheTimeoutFieldEditsTheTabsTimeout(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R"})
	root := world.pane.SettingsView(world.ctx, world.tab, &widget.ScrollState{})
	field := findWidget(root, "*widget.fieldB")
	if field == nil {
		t.Fatalf("no field in:\n%s", Dump(root))
	}
	settings := fmt.Sprintf("%+v", field.Behavior())
	for _, want := range []string{"field:" + fmt.Sprintf("%p", &world.pane.state(world.tab).timeout), "placeholder: ", "onEnter:<nil>"} {
		if !strings.Contains(settings, want) {
			t.Errorf("the field lacks %q: %s", want, settings)
		}
	}
}

// Text that is not a whole number of seconds is undone in the field, which
// stays focused so the user can correct it.
func TestAnInvalidTimeoutIsUndoneInTheField(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R"})
	world.typeInto(0, "7")
	world.typeInto(0, "x")
	field := &world.pane.state(world.tab).timeout
	if field.Text != "7" || !field.Focused {
		t.Fatalf("field %q focused=%v, want %q still focused", field.Text, field.Focused, "7")
	}
	if world.tab.Req.TimeoutSec != 7 {
		t.Fatalf("timeout %d, want the valid 7 kept", world.tab.Req.TimeoutSec)
	}
}

// Dump does not show whether a checkbox is ticked: each one is ticked while
// the request follows redirects and verifies certificates.
func TestTheSettingsChecksFollowTheRequest(t *testing.T) {
	cases := []struct {
		name                     string
		noRedirects, insecureTLS bool
		want                     []string
	}{
		{"defaults", false, false, []string{"{Base:{} on:true}", "{Base:{} on:true}"}},
		{"redirects off", true, false, []string{"{Base:{} on:false}", "{Base:{} on:true}"}},
		{"verification off", false, true, []string{"{Base:{} on:true}", "{Base:{} on:false}"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t, &request.Request{Name: "R", NoRedirects: tc.noRedirects, InsecureTLS: tc.insecureTLS})
			got := behaviorStates(world.pane.SettingsView(world.ctx, world.tab, &widget.ScrollState{}), "widget.checkB")
			if !slices.Equal(got, tc.want) {
				t.Fatalf("checkboxes %q, want %q", got, tc.want)
			}
		})
	}
}
