package ui

import (
	"strings"

	"context"
	"encoding/json"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// mcpCallTimeout is how long the explorer waits for the answer of a call.
const mcpCallTimeout = 60 * time.Second

// exploreChoose shows another list of the server.
func (a *App) exploreChoose(t *reqTab, kind mcpexplorer.Kind) {
	t.explorer.Choose(kind)
	t.form = nil
	t.resultView.SetLines(nil)
}

// exploreOpen opens entry index of the shown list: tools and prompts get a
// form for their arguments, a resource has nothing to fill in.
func (a *App) exploreOpen(t *reqTab, index int) {
	t.explorer.Select(index)
	t.resultView.SetLines(nil)
	t.form = nil
	switch t.explorer.Kind {
	case mcpexplorer.Tools:
		t.form = mcpform.New(t.Tools[index])
	case mcpexplorer.Prompts:
		prompt := t.Prompts[index]
		t.form = mcpform.New(model.MCPTool{Name: prompt.Name, Schema: prompt.Schema()})
	}
}

// exploreCommand is the command the open entry sends, or the error that
// says which value of its form is wrong.
func exploreCommand(t *reqTab) (string, error) {
	if t.explorer.Kind == mcpexplorer.Resources {
		return "read " + t.Resources[t.explorer.Selected].URI, nil
	}
	arguments, err := t.form.Arguments()
	if err != nil {
		return "", err
	}
	verb := "call "
	if t.explorer.Kind == mcpexplorer.Prompts {
		verb = "prompt "
	}
	return verb + t.form.Tool.Name + " " + arguments, nil
}

// exploreCall sends the open entry and shows the answer when it comes.
func (a *App) exploreCall(t *reqTab) {
	requester, ok := t.Session.(reqport.Requester)
	if !ok || t.explorer.Selected == mcpexplorer.None || t.explorer.Calling {
		return
	}
	command, err := exploreCommand(t)
	if err != nil {
		t.form.Problem = err.Error() // only a form can hold a wrong value
		return
	}
	if t.form != nil {
		t.form.Problem = ""
	}
	t.explorer.Begin()
	a.bus.Run(context.Background(), func(context.Context) bus.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), mcpCallTimeout)
		defer cancel()
		result, err := requester.Request(ctx, command)
		return func() { // on the UI goroutine
			t.explorer.Finish(result, err)
			t.resultView.SetLines(resultLines(t.explorer))
		}
	})
}

// resultLines are the lines of an answer: indented JSON, or the plain
// message of an error the server sent.
func resultLines(state mcpexplorer.State) []Line {
	if !json.Valid([]byte(state.Result)) {
		return respview.RawLines([]byte(state.Result))
	}
	return respview.CodeLines([]byte(state.Result))
}

// explorerPane is the MCP view: what the server says about itself, the
// lists it offers on the left and the open entry with its form and its
// answer on the right.
func (a *App) explorerPane(_ pane, t *reqTab, _ *ScrollState) Node {
	return explorerTemplate.MustBuild(markup.Scope{
		"header":  newExplorerHeader(t),
		"list":    a.explorerList(t),
		"detail":  a.explorerDetail(t),
		"headerH": w.RowH + w.Gutter,
		"gutter":  w.Gutter,
	})
}

// explorerHeader is what the explorer template shows of the server: its
// connection state until it answered, then what it said about itself.
type explorerHeader struct {
	Known        bool   // the server answered initialize
	State        string // the connection state, shown until then
	Name         string
	Version      string
	Protocol     string
	Offers       string // the capabilities, left out when there are none
	Instructions string // left out when empty
}

// newExplorerHeader is the header of t's explorer.
func newExplorerHeader(t *reqTab) explorerHeader {
	server := t.Server
	if server == nil {
		return explorerHeader{State: connState(t)}
	}
	return explorerHeader{
		Known:        true,
		Name:         server.Name,
		Version:      server.Version,
		Protocol:     server.Protocol,
		Offers:       strings.Join(server.Capabilities, " · "),
		Instructions: server.Instructions,
	}
}

var (
	explorerTemplate       = markup.Must("explorer.vml", explorerSource)
	explorerListTemplate   = markup.Must("explorerlist.vml", explorerlistSource)
	explorerNoneTemplate   = markup.Must("explorernone.vml", explorernoneSource)
	explorerDetailTemplate = markup.Must("explorerdetail.vml", explorerdetailSource)
	explorerResultTemplate = markup.Must("explorerresult.vml", explorerresultSource)
)

// explorerEntry is what the list template shows of one entry.
type explorerEntry struct {
	Title    string
	Detail   string // muted under the title, left out when empty
	Selected bool
	Open     func()
}

// explorerList is the strip of lists and the entries of the shown one.
func (a *App) explorerList(t *reqTab) Node {
	state := t.explorer
	items := []TabItem{{Label: "Tools", Badge: len(t.Tools)}, {Label: "Resources", Badge: len(t.Resources)}, {Label: "Prompts", Badge: len(t.Prompts)}}
	entries := []explorerEntry{}
	for index, entry := range mcpexplorer.Entries(state.Kind, t.Tools, t.Resources, t.Prompts) {
		entries = append(entries, explorerEntry{
			Title:    entry.Title,
			Detail:   entry.Detail,
			Selected: index == state.Selected,
			Open:     func() { a.exploreOpen(t, index) },
		})
	}
	return explorerListTemplate.MustBuild(markup.Scope{
		"tabs":    items,
		"active":  int(state.Kind),
		"choose":  func(index int) { a.exploreChoose(t, mcpexplorer.Kind(index)) },
		"scroll":  &t.listScroll,
		"entries": entries,
		"listW":   explorerListW,
		"gutter":  w.Gutter,
	})
}

// explorerListW is the width of the list next to the open entry.
const explorerListW = 260

// openEntry is what the detail template shows of the open entry.
type openEntry struct {
	Title        string
	Form         Node // the argument form, nil for a resource
	Label        string
	Call         func()
	Disconnected bool
	Calling      bool
	Result       Node // the last answer, nil before one came
}

// explorerDetail is the open entry: its form, the button that sends it and
// the answer.
func (a *App) explorerDetail(t *reqTab) Node {
	state := t.explorer
	entries := mcpexplorer.Entries(state.Kind, t.Tools, t.Resources, t.Prompts)
	if state.Selected == mcpexplorer.None || state.Selected >= len(entries) {
		return explorerNoneTemplate.MustBuild(markup.Scope{"gutter": w.Gutter})
	}
	detail := openEntry{
		Title:        entries[state.Selected].Title,
		Form:         formNode(a, t),
		Label:        callLabel(state),
		Call:         func() { a.exploreCall(t) },
		Disconnected: !a.connected(t),
		Calling:      state.Calling,
	}
	if state.Result != "" {
		detail.Result = resultNode(t)
	}
	return explorerDetailTemplate.MustBuild(markup.Scope{
		"detail": detail,
		"scroll": &t.detailScroll,
		"gutter": w.Gutter,
		"rowH":   w.RowH,
	})
}

// formNode is the argument form of the open entry, nil for a resource.
func formNode(a *App, t *reqTab) Node {
	if t.form == nil {
		return nil
	}
	return t.form.View(a)
}

// connected reports whether t has a session to send to.
func (a *App) connected(t *reqTab) bool { return t.Session != nil }

// callLabel is the caption of the send button of the open entry.
func callLabel(state mcpexplorer.State) string {
	if state.Kind == mcpexplorer.Resources {
		return "Read"
	}
	return "Call"
}

// resultNode is the last answer, headed by whether it succeeded.
func resultNode(t *reqTab) Node {
	state := t.explorer
	heading, color := "Result", w.OK
	if state.Failed {
		heading, color = "Error", w.Error
	}
	return explorerResultTemplate.MustBuild(markup.Scope{
		"heading": heading,
		"color":   color,
		"view":    &t.resultView,
		"resultH": explorerResultH,
	})
}

// explorerResultH is the height of the answer's view.
const explorerResultH = 240

// Templates, parsed once above.
const (
	explorerSource = `<col grow="1">
  <row if="!header.Known" h="{{headerH}}" pad-x="{{gutter}}" align-items="center" border-b="border">
    <text muted>{{header.State}}: connect to see what the server offers</text>
  </row>
  <col if="header.Known" pad-x="{{gutter}}" pad-y="8" gap="4" border-b="border">
    <row gap="8" align-items="baseline">
      <text bold>{{header.Name}} {{header.Version}}</text>
      <text small muted>protocol {{header.Protocol}}</text>
    </row>
    <text if="header.Offers" small muted>Offers: {{header.Offers}}</text>
    <lines if="header.Instructions" color="muted">{{header.Instructions}}</lines>
  </col>
  <row grow="1">
    <slot name="list"/>
    <slot name="detail"/>
  </row>
</col>
`

	explorerlistSource = `<col w="{{listW}}" border-r="border">
  <tab-bar>
    <tabs items="tabs" active="{{active}}" on-select="choose"/>
  </tab-bar>
  <scroll state="scroll" grow="1">
    <col if="!entries" pad="{{gutter}}">
      <text small muted>Nothing announced</text>
    </col>
    <list-row for="entry in entries" selected="{{entry.Selected}}" pad-x="{{gutter}}" pad-y="6" on-click="entry.Open">
      <col grow="1">
        <text ellipsis>{{entry.Title}}</text>
        <text if="entry.Detail" small muted ellipsis>{{entry.Detail}}</text>
      </col>
    </list-row>
  </scroll>
</col>
`

	explorernoneSource = `<col grow="1" pad="{{gutter}}">
  <text muted>Pick a tool, a resource or a prompt</text>
</col>
`

	explorerdetailSource = `<scroll state="scroll" grow="1" pad="{{gutter}}" gap="10">
  <text bold>{{detail.Title}}</text>
  <slot if="detail.Form" name="detail.Form"/>
  <!-- a resource has no form; the empty column keeps the gap a form takes -->
  <col if="!detail.Form"/>
  <row gap="8" align-items="center">
    <primary-button h="{{rowH}}" pad-x="{{gutter}}" on-click="detail.Call" tooltip="Send it to the server (Enter in a field)">{{detail.Label}}</primary-button>
    <text if="detail.Disconnected" small muted>Connect first</text>
    <text if="detail.Calling" small muted>Waiting for the server...</text>
  </row>
  <slot if="detail.Result" name="detail.Result"/>
</scroll>
`

	explorerresultSource = `<col gap="4">
  <text small bold color="{{color}}">{{heading}}</text>
  <code state="view" h="{{resultH}}" bg="sunken"/>
</col>
`
)
