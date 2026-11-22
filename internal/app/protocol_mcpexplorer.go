package app

import (
	"strings"

	"context"
	"encoding/json"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/ui/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/ui/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// mcpCallTimeout is how long the explorer waits for the answer of a call.
const mcpCallTimeout = 60 * time.Second

// exploreChoose shows another list of the server.
func (a *App) exploreChoose(tab *reqTab, kind mcpexplorer.Kind) {
	tab.explorer.Choose(kind)
	tab.form = nil
	tab.resultView.SetLines(nil)
}

// exploreOpen opens entry index of the shown list: tools and prompts get a
// form for their arguments, a resource has nothing to fill in.
func (a *App) exploreOpen(tab *reqTab, index int) {
	tab.explorer.Select(index)
	tab.resultView.SetLines(nil)
	tab.form = nil
	switch tab.explorer.Kind {
	case mcpexplorer.Tools:
		tab.form = mcpform.New(tab.Tools[index])
	case mcpexplorer.Prompts:
		prompt := tab.Prompts[index]
		tab.form = mcpform.New(request.MCPTool{Name: prompt.Name, Schema: prompt.Schema()})
	}
}

// exploreCommand is the command the open entry sends, or the error that
// says which value of its form is wrong.
func exploreCommand(tab *reqTab) (string, error) {
	if tab.explorer.Kind == mcpexplorer.Resources {
		return "read " + tab.Resources[tab.explorer.Selected].URI, nil
	}
	arguments, err := tab.form.Arguments()
	if err != nil {
		return "", err
	}
	verb := "call "
	if tab.explorer.Kind == mcpexplorer.Prompts {
		verb = "prompt "
	}
	return verb + tab.form.Tool.Name + " " + arguments, nil
}

// exploreCall sends the open entry and shows the answer when it comes.
func (a *App) exploreCall(tab *reqTab) {
	requester, ok := tab.Session().(request.Requester)
	if !ok || tab.explorer.Selected == mcpexplorer.None || tab.explorer.IsCalling() {
		return
	}
	command, err := exploreCommand(tab)
	if err != nil {
		tab.form.Problem = err.Error() // only a form can hold a wrong value
		return
	}
	if tab.form != nil {
		tab.form.Problem = ""
	}
	tab.explorer.Begin()
	a.bus.Run(context.Background(), func(context.Context) bus.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), mcpCallTimeout)
		defer cancel()
		result, err := requester.Request(ctx, command)
		return func() { // on the UI goroutine
			tab.explorer.Finish(result, err)
			tab.resultView.SetLines(resultLines(tab.explorer))
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
func (a *App) explorerPane(_ pane, tab *reqTab, _ *ScrollState) Node {
	return explorerTemplate.MustBuild(a.scope(markup.Scope{
		"looseGap": w.LooseGap,
		"header":   newExplorerHeader(tab),
		"list":     a.explorerList(tab),
		"detail":   a.explorerDetail(tab),
		"headerH":  a.ctx.Sizes.Row + w.Gutter,
		"gutter":   w.Gutter,
	}))
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
func newExplorerHeader(tab *reqTab) explorerHeader {
	server := tab.Server
	if server == nil {
		return explorerHeader{State: connState(tab)}
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
func (a *App) explorerList(tab *reqTab) Node {
	state := tab.explorer
	items := []TabItem{{Label: "Tools", Badge: len(tab.Tools)}, {Label: "Resources", Badge: len(tab.Resources)}, {Label: "Prompts", Badge: len(tab.Prompts)}}
	entries := []explorerEntry{}
	for index, entry := range mcpexplorer.Entries(state.Kind, tab.Tools, tab.Resources, tab.Prompts) {
		entries = append(entries, explorerEntry{
			Title:    entry.Title,
			Detail:   entry.Detail,
			Selected: index == state.Selected,
			Open:     func() { a.exploreOpen(tab, index) },
		})
	}
	return explorerListTemplate.MustBuild(a.scope(markup.Scope{
		"tabs":    items,
		"active":  int(state.Kind),
		"choose":  func(index int) { a.exploreChoose(tab, mcpexplorer.Kind(index)) },
		"scroll":  &tab.listScroll,
		"entries": entries,
		"listW":   explorerListW,
		"gutter":  w.Gutter,
	}))
}

// explorerListW is the width of the list next to the open entry.
const explorerListW = 260

// openEntry is what the detail template shows of the open entry.
type openEntry struct {
	Title        string
	Form         Node // the argument form, nil for a resource
	Label        string
	Call         func()
	NotConnected string // a note while there is no session, else ""
	Waiting      string // a note while a call is in flight, else ""
	Result       Node   // the last answer, nil before one came
}

// explorerDetail is the open entry: its form, the button that sends it and
// the answer.
func (a *App) explorerDetail(tab *reqTab) Node {
	state := tab.explorer
	entries := mcpexplorer.Entries(state.Kind, tab.Tools, tab.Resources, tab.Prompts)
	if state.Selected == mcpexplorer.None || state.Selected >= len(entries) {
		return explorerNoneTemplate.MustBuild(markup.Scope{"gutter": w.Gutter})
	}
	detail := openEntry{
		Title: entries[state.Selected].Title,
		Form:  formNode(a, tab),
		Label: callLabel(state),
		Call:  func() { a.exploreCall(tab) },
	}
	if !a.connected(tab) {
		detail.NotConnected = "Connect first"
	}
	if state.IsCalling() {
		detail.Waiting = "Waiting for the server..."
	}
	if state.Result != "" {
		detail.Result = a.resultNode(tab)
	}
	return explorerDetailTemplate.MustBuild(a.scope(markup.Scope{
		"detail": detail,
		"scroll": &tab.detailScroll,
	}))
}

// formNode is the argument form of the open entry, nil for a resource.
func formNode(app *App, tab *reqTab) Node {
	if tab.form == nil {
		return nil
	}
	return tab.form.View(app.ctx, app)
}

// connected reports whether t has a session to send to.
func (a *App) connected(tab *reqTab) bool { return tab.Session() != nil }

// callLabel is the caption of the send button of the open entry.
func callLabel(state mcpexplorer.State) string {
	if state.Kind == mcpexplorer.Resources {
		return "Read"
	}
	return "Call"
}

// resultNode is the last answer, headed by whether it succeeded.
func (a *App) resultNode(tab *reqTab) Node {
	state := tab.explorer
	heading, color := "Result", w.OK
	if state.IsFailed() {
		heading, color = "Error", w.Error
	}
	return explorerResultTemplate.MustBuild(a.scope(markup.Scope{
		"heading": heading,
		"color":   color,
		"view":    &tab.resultView,
		"resultH": explorerResultH,
	}))
}

// explorerResultH is the height of the answer's view.
const explorerResultH = 240

// Templates, parsed once above.
const (
	explorerSource = `<col grow="1">
  <row if="!header.Known" h="{{headerH}}" pad-x="{{gutter}}" align-items="center" border-b="border">
    <text muted>{{header.State}}: connect to see what the server offers</text>
  </row>
  <col if="header.Known" pad-x="{{gutter}}" pad-y="{{looseGap}}" gap="{{nearGap}}" border-b="border">
    <row gap="{{looseGap}}" align-items="baseline">
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
    <list-row for="entry in entries" selected="{{entry.Selected}}" pad-x="{{gutter}}" pad-y="{{tightGap}}" on-click="entry.Open">
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

	explorerdetailSource = `<scroll state="scroll" grow="1" pad="{{gutter}}" gap="{{detailGap}}">
  <text bold>{{detail.Title}}</text>
  <slot if="detail.Form" name="detail.Form"/>
  <!-- a resource has no form; the empty column keeps the gap a form takes -->
  <col if="!detail.Form"/>
  <row gap="{{looseGap}}" align-items="center">
    <primary-button h="{{rowH}}" pad-x="{{gutter}}" on-click="detail.Call" tooltip="Send it to the server (Enter in a field)">{{detail.Label}}</primary-button>
    <text if="detail.NotConnected" small muted>{{detail.NotConnected}}</text>
    <text if="detail.Waiting" small muted>{{detail.Waiting}}</text>
  </row>
  <slot if="detail.Result" name="detail.Result"/>
</scroll>
`

	explorerresultSource = `<col gap="{{nearGap}}">
  <text small bold color="{{color}}">{{heading}}</text>
  <code state="view" h="{{resultH}}" bg="sunken"/>
</col>
`
)
