package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/mcpexplorer"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Sizes the explorer is pinned at: a roomy response pane, and one so small
// that the list scrolls and long names are cut.
const (
	explorerWideW, explorerWideH     = 900, 600
	explorerNarrowW, explorerNarrowH = 400, 160
)

// longToolName and longToolDetail do not fit the list.
const (
	longToolName   = "a_tool_whose_name_is_much_longer_than_the_list_is_wide"
	longToolDetail = "A description that goes on well past the right edge of the list next to the open entry"
)

// mcpTab is the test app's current tab, switched to MCP with its explorer
// shown on the response side.
func mcpTab(t *testing.T) (*App, *reqTab) {
	t.Helper()
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	tab.respTab = 0
	return a, tab
}

// announceLong puts a list of many tools into the tab, the first with a
// long name and description.
func announceLong(a *App, tab *reqTab) {
	tools := []string{fmt.Sprintf(`{"name":%q,"description":%q}`, longToolName, longToolDetail)}
	for i := range 12 {
		tools = append(tools, fmt.Sprintf(`{"name":"tool_%d","description":"Tool number %d"}`, i, i))
	}
	list := `{"result":{"tools":[` + strings.Join(tools, ",") + `]}}`
	a.applyWS(wsEvent{tab: tab, hasEv: true, ev: reqport.Event{Dir: reqport.Received, Kind: "response", Text: list}})
}

// The laid-out explorer is pinned as text (testdata/explorer-*.dump,
// captured from the Go-built view), so moving it to a template cannot
// change a rectangle, a color or a handler.
func TestTheMCPExplorerLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(a *App, tab *reqTab)
	}{
		{"not-announced", func(*App, *reqTab) {}},
		{"tools", announce},
		{"tool-open-not-connected", func(a *App, tab *reqTab) {
			announce(a, tab)
			a.exploreOpen(tab, 0)
		}},
		{"tool-calling", func(a *App, tab *reqTab) {
			announce(a, tab)
			tab.Session = &requesterSession{}
			a.exploreOpen(tab, 1)
			tab.explorer.Begin()
		}},
		{"tool-result", func(a *App, tab *reqTab) {
			announce(a, tab)
			tab.Session = &requesterSession{}
			a.exploreOpen(tab, 1)
			tab.explorer.Finish(`{"content":[{"type":"text","text":"pong"}]}`, nil)
			tab.resultView.SetLines(resultLines(tab.explorer))
		}},
		{"tool-error", func(a *App, tab *reqTab) {
			announce(a, tab)
			tab.Session = &requesterSession{}
			a.exploreOpen(tab, 1)
			tab.explorer.Finish("", errors.New("no such tool"))
			tab.resultView.SetLines(resultLines(tab.explorer))
		}},
		{"resources-open", func(a *App, tab *reqTab) {
			announce(a, tab)
			tab.Session = &requesterSession{}
			a.exploreChoose(tab, mcpexplorer.Resources)
			a.exploreOpen(tab, 0)
		}},
		{"prompts-open", func(a *App, tab *reqTab) {
			announce(a, tab)
			tab.Session = &requesterSession{}
			a.exploreChoose(tab, mcpexplorer.Prompts)
			a.exploreOpen(tab, 0)
		}},
		{"prompts-empty", func(a *App, tab *reqTab) {
			announceLong(a, tab)
			a.exploreChoose(tab, mcpexplorer.Prompts)
		}},
		{"long-names", func(a *App, tab *reqTab) {
			announceLong(a, tab)
			a.exploreOpen(tab, 0)
		}},
	}
	sizes := []struct {
		name          string
		width, height int
	}{
		{"wide", explorerWideW, explorerWideH},
		{"narrow", explorerNarrowW, explorerNarrowH},
	}
	for _, tc := range cases {
		for _, size := range sizes {
			name := "explorer-" + tc.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, tab := mcpTab(t)
				tc.prepare(a, tab)
				root := a.explorerPane(pane{}, tab, nil)
				Layout(a.ctx, root, size.width, size.height)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
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
			a, tab := mcpTab(t)
			announce(a, tab)
			if tc.want == mcpexplorer.Tools {
				a.exploreChoose(tab, mcpexplorer.Prompts)
			}
			a.exploreOpen(tab, 0)
			clickText(t, a, tc.label, 0)
			if tab.explorer.Kind != tc.want || tab.explorer.Selected != mcpexplorer.None || tab.form != nil {
				t.Fatalf("after clicking %s: state %+v, form %v", tc.label, tab.explorer, tab.form)
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
			a, tab := mcpTab(t)
			announce(a, tab)
			a.exploreChoose(tab, tc.kind)
			clickText(t, a, tc.title, 0)
			if tab.explorer.Selected != tc.index || (tab.form != nil) != tc.wantForm {
				t.Fatalf("after clicking %s: state %+v, form %v", tc.title, tab.explorer, tab.form)
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
			a, tab := mcpTab(t)
			announce(a, tab)
			session := &requesterSession{result: `{}`}
			tab.Session = session
			a.exploreChoose(tab, tc.kind)
			index := 0
			if tc.kind == mcpexplorer.Tools {
				index = 1
			}
			a.exploreOpen(tab, index)
			clickText(t, a, tc.button, 0)
			until(t, a, "the answer", func() bool { return !tab.explorer.Calling })
			if len(session.asked) != 1 || session.asked[0] != tc.want {
				t.Fatalf("asked %q, want %q", session.asked, tc.want)
			}
		})
	}
}
