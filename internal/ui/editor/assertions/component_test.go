package assertions

import (
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// setVar is one captured value the component stored.
type setVar struct {
	req         *request.Request
	name, value string
}

// fake is the host of the component: real tabs, recorded variables and status.
type fake struct {
	set      *tabs.Set
	vars     []setVar
	status   string
	settings settings.Settings
}

func (fake *fake) Current() *tabs.Tab { return fake.set.Current() }
func (fake *fake) IsOpen(tab *tabs.Tab) bool {
	for _, open := range fake.set.Tabs() {
		if open == tab {
			return true
		}
	}
	return false
}
func (fake *fake) SetVar(req *request.Request, name, value string) {
	fake.vars = append(fake.vars, setVar{req, name, value})
}
func (fake *fake) Status() string        { return fake.status }
func (fake *fake) SetStatus(text string) { fake.status = text }
func (fake *fake) DefaultTimeout() time.Duration {
	return fake.settings.TimeoutFor(0)
}

// world is the component over a fake, with one open request.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
}

func newWorld(t *testing.T, req *request.Request) *world {
	t.Helper()
	prefs := settings.Default()
	ctx, err := core.NewCtx(theme.ThemeWith(prefs))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet(), settings: prefs}
	fake.set.Open(req)
	pane := New(Deps{Tabs: fake, Vars: fake, Status: fake, Defaults: fake})
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: fake.set.Current()}
}

func TestTheTestsStateIsMadeFromTheRequestOnFirstUse(t *testing.T) {
	cases := []struct {
		name        string
		req         request.Request
		wantTests   string
		wantTimeout string
	}{
		{"both set", request.Request{Name: "R", Tests: "status == 200", TimeoutSec: 30}, "status == 200", "30"},
		{"default timeout", request.Request{Name: "R"}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t, &tc.req)
			state := world.pane.state(world.tab)
			if state.tests.Text != tc.wantTests || state.timeout.Text != tc.wantTimeout {
				t.Fatalf("tests %q timeout %q, want %q and %q", state.tests.Text, state.timeout.Text, tc.wantTests, tc.wantTimeout)
			}
			if world.pane.state(world.tab) != state {
				t.Fatal("the state must keep its place between calls: views hold pointers into it")
			}
		})
	}
}

func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "first", Tests: "status == 200"})
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second"})
	world.pane.state(first).tests.Insert(" typed")
	world.fake.set.Apply(tabs.Close{Index: 0})
	if world.pane.Tick() {
		t.Fatal("dropping a state changes nothing on screen")
	}
	if len(world.pane.states) != 0 {
		t.Fatalf("%d states left, want none", len(world.pane.states))
	}
}

func TestTypingCountsOnlyTheCurrentTabsFields(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "first"})
	if world.pane.Typing() {
		t.Fatal("nothing is focused at the start")
	}
	if len(world.pane.states) != 0 {
		t.Fatal("asking whether the user types must not make state")
	}
	cases := []struct {
		name  string
		focus func(*state)
	}{
		{"tests area", func(state *state) { state.tests.Focused = true }},
		{"timeout field", func(state *state) { state.timeout.Focused = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := world.pane.state(world.tab)
			tc.focus(state)
			if !world.pane.Typing() {
				t.Fatal("a focused field of the current tab is typing")
			}
			state.tests.Focused, state.timeout.Focused = false, false
		})
	}
	world.pane.state(world.tab).tests.Focused = true
	world.fake.set.Open(&request.Request{Name: "second"})
	if world.pane.Typing() {
		t.Fatal("the focus of a hidden tab's field does not count")
	}
}

func TestResultsOfATabNeverCheckedAreNoneAndMakeNoState(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Tests: "status == 200"})
	if got := world.pane.Results(world.tab); got != nil {
		t.Fatalf("results %+v, want none", got)
	}
	if len(world.pane.states) != 0 {
		t.Fatal("reading the results must not make state")
	}
}

func TestRunChecksTheResponseKeepsTheCapturesAndSumsUp(t *testing.T) {
	ok := &request.Response{Status: 200, StatusText: "OK", Body: []byte(`{"token":"abc"}`)}
	cases := []struct {
		name       string
		tests      string
		resp       *request.Response
		wantOK     []bool
		wantVars   []string
		wantStatus string
	}{
		{"good response", "status == 200\nstatus == 404\nset token = json token", ok, []bool{true, false, true}, []string{"token=abc"}, "sent  Tests: 2/3 passed"},
		{"response with an error", "status == 200", &request.Response{Err: "refused"}, []bool{false}, nil, "sent"},
		{"no assertions", "", ok, nil, nil, "sent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t, &request.Request{Name: "R", Tests: tc.tests})
			world.fake.status = "sent"
			world.tab.Resp = tc.resp
			world.pane.Run(world.tab)
			var gotOK []bool
			for _, result := range world.pane.Results(world.tab) {
				gotOK = append(gotOK, result.OK)
			}
			if len(gotOK) != len(tc.wantOK) {
				t.Fatalf("results %v, want %v", gotOK, tc.wantOK)
			}
			for i := range gotOK {
				if gotOK[i] != tc.wantOK[i] {
					t.Fatalf("results %v, want %v", gotOK, tc.wantOK)
				}
			}
			var gotVars []string
			for _, set := range world.fake.vars {
				if set.req != world.tab.Req {
					t.Fatalf("variable stored for another request")
				}
				gotVars = append(gotVars, set.name+"="+set.value)
			}
			if len(gotVars) != len(tc.wantVars) || (len(gotVars) > 0 && gotVars[0] != tc.wantVars[0]) {
				t.Fatalf("variables %v, want %v", gotVars, tc.wantVars)
			}
			if world.fake.status != tc.wantStatus {
				t.Fatalf("status %q, want %q", world.fake.status, tc.wantStatus)
			}
		})
	}
	_ = assert.Summary
}
