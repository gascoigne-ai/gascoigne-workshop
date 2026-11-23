package mcp

import (
	"encoding/json"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// resultLines are the lines of an answer: indented JSON, or the plain
// message of an error the server sent.
func (c *Component) resultLines(state mcpexplorer.State) []widget.Line {
	if !json.Valid([]byte(state.Result)) {
		return c.deps.Looks.RawLines([]byte(state.Result))
	}
	return c.deps.Looks.CodeLines([]byte(state.Result))
}

// View is the MCP explorer: what the server says about itself, the lists it
// offers on the left and the open entry with its form and its answer on the
// right.
func (c *Component) View(ctx *w.Ctx, tab *tabs.Tab) Node {
	made := c.state(tab)
	return explorerTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"looseGap": w.LooseGap,
		"header":   c.header(tab),
		"list":     c.list(ctx, tab, made),
		"detail":   c.detail(ctx, tab, made),
		"headerH":  ctx.Sizes.Row + w.Gutter,
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

// header is the header of the explorer of tab.
func (c *Component) header(tab *tabs.Tab) explorerHeader {
	server := tab.Server
	if server == nil {
		return explorerHeader{State: c.deps.Looks.StateText(tab.State())}
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

// list is the strip of lists and the entries of the shown one.
func (c *Component) list(ctx *w.Ctx, tab *tabs.Tab, made *state) Node {
	shown := made.explorer
	items := []widget.TabItem{{Label: "Tools", Badge: len(tab.Tools)}, {Label: "Resources", Badge: len(tab.Resources)}, {Label: "Prompts", Badge: len(tab.Prompts)}}
	entries := []explorerEntry{}
	for index, entry := range mcpexplorer.Entries(shown.Kind, tab.Tools, tab.Resources, tab.Prompts) {
		entries = append(entries, explorerEntry{
			Title:    entry.Title,
			Detail:   entry.Detail,
			Selected: index == shown.Selected,
			Open:     func() { c.Open(tab, index) },
		})
	}
	return explorerListTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"tabs":    items,
		"active":  int(shown.Kind),
		"choose":  func(index int) { c.Choose(tab, mcpexplorer.Kind(index)) },
		"scroll":  &made.list,
		"entries": entries,
		"listW":   listW,
		"gutter":  w.Gutter,
	}))
}

// listW is the width of the list next to the open entry.
const listW = 260

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

// detail is the open entry: its form, the button that sends it and the
// answer.
func (c *Component) detail(ctx *w.Ctx, tab *tabs.Tab, made *state) Node {
	shown := made.explorer
	entries := mcpexplorer.Entries(shown.Kind, tab.Tools, tab.Resources, tab.Prompts)
	if shown.Selected == mcpexplorer.None || shown.Selected >= len(entries) {
		return explorerNoneTemplate.MustBuild(markup.Scope{"gutter": w.Gutter})
	}
	detail := openEntry{
		Title: entries[shown.Selected].Title,
		Form:  c.formNode(ctx, tab, made),
		Label: callLabel(shown),
		Call:  func() { c.Call(tab) },
	}
	if tab.Session() == nil {
		detail.NotConnected = "Connect first"
	}
	if shown.IsCalling() {
		detail.Waiting = "Waiting for the server..."
	}
	if shown.Result != "" {
		detail.Result = c.resultNode(ctx, made)
	}
	return explorerDetailTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"detail": detail,
		"scroll": &made.detail,
	}))
}

// formNode is the argument form of the open entry, nil for a resource.
func (c *Component) formNode(ctx *w.Ctx, tab *tabs.Tab, made *state) Node {
	if made.form == nil {
		return nil
	}
	return made.form.View(ctx, submit(func() { c.Call(tab) }))
}

// callLabel is the caption of the send button of the open entry.
func callLabel(shown mcpexplorer.State) string {
	if shown.Kind == mcpexplorer.Resources {
		return "Read"
	}
	return "Call"
}

// resultNode is the last answer, headed by whether it succeeded.
func (c *Component) resultNode(ctx *w.Ctx, made *state) Node {
	heading, color := "Result", w.OK
	if made.explorer.IsFailed() {
		heading, color = "Error", w.Error
	}
	return explorerResultTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"heading": heading,
		"color":   color,
		"view":    &made.result,
		"resultH": resultH,
	}))
}

// resultH is the height of the answer's view.
const resultH = 240

// submit is what Enter in a field of a form does, as the form's host.
type submit func()

// Submit runs the function.
func (run submit) Submit() { run() }

// command is the command the open entry sends, or the error that says which
// value of its form is wrong.
func command(tab *tabs.Tab, made *state) (string, error) {
	if made.explorer.Kind == mcpexplorer.Resources {
		return "read " + tab.Resources[made.explorer.Selected].URI, nil
	}
	arguments, err := made.form.Arguments()
	if err != nil {
		return "", err
	}
	verb := "call "
	if made.explorer.Kind == mcpexplorer.Prompts {
		verb = "prompt "
	}
	return verb + made.form.Tool.Name + " " + arguments, nil
}

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
