package mcp

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The laid-out explorer is pinned as text (testdata/explorer-*.dump,
// element kinds, options, rects) at two sizes; the goldens are the ones the
// pane had in the app before it moved here.
func TestTheMCPExplorerLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(world *world)
	}{
		{"not-announced", func(*world) {}},
		{"tools", func(world *world) { world.announce() }},
		{"tool-open-not-connected", func(world *world) {
			world.announce()
			world.pane.Open(world.tab, 0)
		}},
		{"tool-calling", func(world *world) {
			world.announce()
			world.connect(&fakeSession{})
			world.pane.Open(world.tab, 1)
			world.state().explorer.Begin()
		}},
		{"tool-result", func(world *world) {
			world.announce()
			world.connect(&fakeSession{})
			world.pane.Open(world.tab, 1)
			made := world.state()
			made.explorer.Finish(`{"content":[{"type":"text","text":"pong"}]}`, nil)
			made.result.SetLines(world.pane.resultLines(made.explorer))
		}},
		{"tool-error", func(world *world) {
			world.announce()
			world.connect(&fakeSession{})
			world.pane.Open(world.tab, 1)
			made := world.state()
			made.explorer.Finish("", errors.New("no such tool"))
			made.result.SetLines(world.pane.resultLines(made.explorer))
		}},
		{"resources-open", func(world *world) {
			world.announce()
			world.connect(&fakeSession{})
			world.pane.Choose(world.tab, mcpexplorer.Resources)
			world.pane.Open(world.tab, 0)
		}},
		{"prompts-open", func(world *world) {
			world.announce()
			world.connect(&fakeSession{})
			world.pane.Choose(world.tab, mcpexplorer.Prompts)
			world.pane.Open(world.tab, 0)
		}},
		{"prompts-empty", func(world *world) {
			world.announceLong()
			world.pane.Choose(world.tab, mcpexplorer.Prompts)
		}},
		{"long-names", func(world *world) {
			world.announceLong()
			world.pane.Open(world.tab, 0)
		}},
		{"server-full", func(world *world) {
			world.initialize(`{"protocolVersion":"2025-06-18","capabilities":{"tools":{},"resources":{}},"serverInfo":{"name":"mock","version":"1.2.0"},"instructions":"Use echo.\nThen call add with two numbers."}`)
			world.announce()
		}},
		{"server-bare", func(world *world) {
			world.initialize(`{"protocolVersion":"2025-06-18","serverInfo":{"name":"bare"}}`)
		}},
	}
	sizes := []struct {
		name          string
		width, height int
	}{
		{"wide", wideW, wideH},
		{"narrow", narrowW, narrowH},
	}
	for _, tc := range cases {
		for _, size := range sizes {
			name := "explorer-" + tc.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t)
				tc.prepare(world)
				assertDumpPinned(t, world.ctx, world.view(), name, size.width, size.height)
			})
		}
	}
}

// Clicking a list's tab in the window shows that list and closes the open
// entry.
func TestClickingAnExplorerTabShowsItsList(t *testing.T) {
	cases := []struct {
		label string
		want  mcpexplorer.Kind
	}{
		{"Resources", mcpexplorer.Resources},
		{"Prompts", mcpexplorer.Prompts},
		{"Tools", mcpexplorer.Tools},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			world := newWorld(t)
			world.announce()
			if tc.want == mcpexplorer.Tools {
				world.pane.Choose(world.tab, mcpexplorer.Prompts)
			}
			world.pane.Open(world.tab, 0)
			world.clickText(tc.label, 0)
			made := world.state()
			if made.explorer.Kind != tc.want || made.explorer.Selected != mcpexplorer.None || made.form != nil {
				t.Fatalf("after clicking %s: state %+v, form %v", tc.label, made.explorer, made.form)
			}
		})
	}
}

// Clicking an entry opens it: a tool and a prompt get a form, a resource
// none.
func TestClickingAnExplorerEntryOpensIt(t *testing.T) {
	cases := []struct {
		name     string
		kind     mcpexplorer.Kind
		title    string
		index    int
		wantForm bool
	}{
		{"second tool", mcpexplorer.Tools, "ping", 1, true},
		{"resource", mcpexplorer.Resources, "file:///a.txt", 0, false},
		{"prompt", mcpexplorer.Prompts, "review", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t)
			world.announce()
			world.pane.Choose(world.tab, tc.kind)
			world.clickText(tc.title, 0)
			made := world.state()
			if made.explorer.Selected != tc.index || (made.form != nil) != tc.wantForm {
				t.Fatalf("after clicking %s: state %+v, form %v", tc.title, made.explorer, made.form)
			}
		})
	}
}

// The open entry's button sends it: Call for a tool, Read for a resource.
func TestClickingTheExplorersButtonSendsTheOpenEntry(t *testing.T) {
	cases := []struct {
		name   string
		kind   mcpexplorer.Kind
		button string
		want   string
	}{
		{"tool", mcpexplorer.Tools, "Call", `call ping {}`},
		{"resource", mcpexplorer.Resources, "Read", `read file:///a.txt`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t)
			world.announce()
			session := &fakeSession{result: `{}`}
			world.connect(session)
			world.pane.Choose(world.tab, tc.kind)
			index := 0
			if tc.kind == mcpexplorer.Tools {
				index = 1
			}
			world.pane.Open(world.tab, index)
			world.clickText(tc.button, 0)
			world.fake.until(t, "the answer", func() bool { return !world.pane.Calling(world.tab) })
			if got := session.questions(); len(got) != 1 || got[0] != tc.want {
				t.Fatalf("asked %q, want %q", got, tc.want)
			}
		})
	}
}

// The empty list's note sat flush against the edge of the list and the
// tab strip had no line under it, unlike every other tab strip.
func TestExplorerListInsetsItsNoteAndRulesTheTabStrip(t *testing.T) {
	world := newWorld(t)
	root := world.view()
	Layout(world.ctx, root, wideW, wideH)
	dump := Dump(root)
	if !strings.Contains(dump, "borderB=border") {
		t.Errorf("the tab strip has no rule under it:\n%s", dump)
	}
	match := regexp.MustCompile(`text="Nothing announced".*rect=(\d+),`).FindStringSubmatch(dump)
	if match == nil {
		t.Fatalf("no note in:\n%s", dump)
	}
	if x, _ := strconv.Atoi(match[1]); x < core.Gutter {
		t.Errorf("the note starts at x=%d, want at least the gutter %d", x, core.Gutter)
	}
}
