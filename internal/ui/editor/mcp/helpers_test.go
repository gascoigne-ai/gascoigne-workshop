package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes the explorer is pinned at: a roomy response pane, and one so small
// the list and the open entry have to share it.
const (
	wideW, wideH     = 900, 600
	narrowW, narrowH = 400, 160
)

// shownDialog is a dialog the component asked the window to show.
type shownDialog struct {
	state   widget.DialogState
	content func(ctx *core.Ctx) Node
}

// button is the button of the dialog labelled label.
func (shown *shownDialog) button(t *testing.T, label string) widget.DialogButton {
	t.Helper()
	for _, button := range shown.state.Buttons {
		if button.Label == label {
			return button
		}
	}
	t.Fatalf("no %q button in %+v", label, shown.state.Buttons)
	return widget.DialogButton{}
}

// fake is the host of the component: real tabs, a queue standing for the
// UI goroutine, and a record of what the component asked for.
type fake struct {
	set *tabs.Set

	mu    sync.Mutex
	queue []func()

	composed []string          // what SetComposer put in a composer, in order
	called   []request.MCPTool // the tools ComposeCall was asked for
	dialog   *shownDialog      // the open dialog, nil for none
	dialogs  int               // how many dialogs were opened
}

func (fake *fake) IsOpen(tab *tabs.Tab) bool { return slices.Contains(fake.set.Tabs(), tab) }

// Run does work on a goroutine of its own; the answer waits in the queue
// until the test pumps it.
func (fake *fake) Run(work func() func()) {
	go func() {
		if answer := work(); answer != nil {
			fake.Dispatch(answer)
		}
	}()
}

func (fake *fake) Dispatch(run func()) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.queue = append(fake.queue, run)
}

// pump runs what waits in the queue, as the UI goroutine does once a frame.
func (fake *fake) pump() {
	fake.mu.Lock()
	waiting := fake.queue
	fake.queue = nil
	fake.mu.Unlock()
	for _, run := range waiting {
		run()
	}
}

// until pumps the queue until cond holds.
func (fake *fake) until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for range 400 {
		fake.pump()
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func (fake *fake) SetComposer(_ *tabs.Tab, text string) { fake.composed = append(fake.composed, text) }
func (fake *fake) ComposeCall(_ *tabs.Tab, tool request.MCPTool) {
	fake.called = append(fake.called, tool)
}

func (fake *fake) Open(state widget.DialogState, content func(ctx *core.Ctx) Node) func() {
	shown := &shownDialog{state: state, content: content}
	fake.dialog = shown
	fake.dialogs++
	return func() {
		if fake.dialog == shown {
			fake.dialog = nil
		}
	}
}

func (fake *fake) Busy() bool { return fake.dialog != nil }

// The looks below are the host's, in the small: the state names of the
// connection and plain lines, which is all the explorer's dumps show.
var stateTexts = map[tabs.ConnState]string{
	tabs.ConnIdle:       "Not connected",
	tabs.ConnConnecting: "Connecting...",
	tabs.ConnOpen:       "Connected",
	tabs.ConnClosing:    "Connected",
	tabs.ConnRetrying:   "Not connected",
}

func (fake *fake) StateText(state tabs.ConnState) string { return stateTexts[state] }
func (fake *fake) CodeLines(body []byte) []widget.Line   { return plainLines(body) }
func (fake *fake) RawLines(body []byte) []widget.Line    { return plainLines(body) }

func plainLines(body []byte) []widget.Line {
	var lines []widget.Line
	for _, text := range strings.Split(string(body), "\n") {
		lines = append(lines, widget.Line{Spans: []widget.Span{{Text: text}}})
	}
	return lines
}

// fakeSession is an open session that records what it is asked and
// answers with result and err.
type fakeSession struct {
	mu     sync.Mutex
	asked  []string
	result string
	err    error
}

func (session *fakeSession) Events() <-chan request.Event       { return nil }
func (session *fakeSession) Close(context.Context) error        { return nil }
func (session *fakeSession) Ping(context.Context) error         { return nil }
func (session *fakeSession) Send(context.Context, string) error { return nil }

func (session *fakeSession) Request(_ context.Context, text string) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.asked = append(session.asked, text)
	return session.result, session.err
}

// questions is what was asked, in order.
func (session *fakeSession) questions() []string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return slices.Clone(session.asked)
}

// world is the component over a fake, with one open MCP request.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet()}
	fake.set.Open(&request.Request{Name: "R", Protocol: request.ProtocolMCP, URL: "http://localhost:3000/mcp"})
	pane := New(Deps{Tabs: fake, Bus: fake, Composer: fake, Dialogs: fake, Looks: fake})
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: fake.set.Current()}
}

// state is what the component keeps of the tab, which must be there.
func (world *world) state() *state {
	world.t.Helper()
	made, ok := world.pane.states[world.tab]
	if !ok {
		world.t.Fatal("the component keeps no state of the tab")
	}
	return made
}

// connect puts the tab in the open state with session.
func (world *world) connect(session request.Session) {
	world.tab.Begin()
	world.tab.Opened(session)
}

// announce puts a server's lists into the tab, as its events would.
func (world *world) announce() {
	for _, list := range []string{
		`{"result":{"tools":[{"name":"search","description":"Find","inputSchema":{"type":"object","properties":{"q":{"type":"string"},"limit":{"type":"integer","default":5}},"required":["q"]}},{"name":"ping"}]}}`,
		`{"result":{"resources":[{"uri":"file:///a.txt","name":"A"}]}}`,
		`{"result":{"prompts":[{"name":"review","arguments":[{"name":"code","required":true}]}]}}`,
	} {
		world.tab.Received(listEvent(list))
	}
}

// announceLong puts a list of many tools into the tab, the first with a
// long name and description.
func (world *world) announceLong() {
	tools := []string{fmt.Sprintf(`{"name":%q,"description":%q}`, longToolName, longToolDetail)}
	for i := range 12 {
		tools = append(tools, fmt.Sprintf(`{"name":"tool_%d","description":"Tool number %d"}`, i, i))
	}
	world.tab.Received(listEvent(`{"result":{"tools":[` + strings.Join(tools, ",") + `]}}`))
}

// longToolName and longToolDetail do not fit the list.
const (
	longToolName   = "a_tool_whose_name_is_much_longer_than_the_list_is_wide"
	longToolDetail = "A description that goes on well past the right edge of the list next to the open entry"
)

// initialize delivers the server's answer to initialize, which the header
// of the explorer shows.
func (world *world) initialize(result string) {
	world.tab.Received(request.Event{Dir: request.Info, Kind: request.KindInitialize, Text: result})
}

// listEvent is the event the MCP session emits for a JSON-RPC response to
// a list request: the text of the message and the typed payload.
func listEvent(text string) request.Event {
	var message struct{ Result json.RawMessage }
	_ = json.Unmarshal([]byte(text), &message) // the callers pass fixed, valid messages
	event := request.Event{Dir: request.Received, Kind: request.KindResponse, Text: text}
	switch {
	case strings.Contains(text, `"tools":`):
		tools, _ := request.DecodeTools(message.Result)
		event.Payload = request.ToolsList{Tools: tools}
	case strings.Contains(text, `"resources":`):
		resources, _ := request.DecodeResources(message.Result)
		event.Payload = request.ResourcesList{Resources: resources}
	case strings.Contains(text, `"prompts":`):
		prompts, _ := request.DecodePrompts(message.Result)
		event.Payload = request.PromptsList{Prompts: prompts}
	}
	return event
}

// view builds the explorer of the tab.
func (world *world) view() Node { return world.pane.View(world.ctx, world.tab) }

// assertDumpPinned lays root out in a window of width by height and
// compares its dump with testdata/name.dump.
func assertDumpPinned(t *testing.T, ctx *core.Ctx, root Node, name string, width, height int) {
	t.Helper()
	Layout(ctx, root, width, height)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Dump(root); got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// textNode is the nth (from 0) element under root that shows text exactly.
func textNode(root Node, text string, nth int) Node {
	var found []Node
	var walk func(Node)
	walk = func(node Node) {
		if strings.Contains(strings.SplitN(Dump(node), "\n", 2)[0], fmt.Sprintf("text=%q", text)) {
			found = append(found, node)
		}
		for _, child := range node.Children() {
			walk(child)
		}
	}
	walk(root)
	if nth >= len(found) {
		return nil
	}
	return found[nth]
}

// frame builds the explorer, lays it out wide and runs one frame of input
// over the element pick finds, where the pointer stays when pick is nil.
func (world *world) frame(in core.Input, pick func(root Node) Node) {
	world.t.Helper()
	world.frameOver(world.view, in, pick)
}

// frameOver is frame over the tree build makes.
func (world *world) frameOver(build func() Node, in core.Input, pick func(root Node) Node) {
	world.t.Helper()
	root := build()
	Layout(world.ctx, root, wideW, wideH)
	if pick != nil {
		target := pick(root)
		if target == nil {
			world.t.Fatalf("nothing to point at in:\n%s", Dump(root))
		}
		at := target.Rect()
		in.X, in.Y = at.X+at.W/2, at.Y+at.H/2
	}
	world.ctx.In = in
	Update(world.ctx, root)
}

// clickOn presses the element pick finds and lets go.
func (world *world) clickOn(pick func(Node) Node) {
	world.t.Helper()
	world.frame(core.Input{Click: true, Held: true}, pick)
	world.frame(core.Input{}, nil)
}

// clickText presses the nth element that shows text and lets go.
func (world *world) clickText(text string, nth int) {
	world.t.Helper()
	world.clickOn(func(root Node) Node { return textNode(root, text, nth) })
}

// fieldAt is the pick of the nth text field of the explorer.
func fieldAt(nth int) func(Node) Node {
	return func(root Node) Node {
		var fields []Node
		var walk func(Node)
		walk = func(node Node) {
			if fmt.Sprintf("%T", node.Behavior()) == "*widget.fieldB" {
				fields = append(fields, node)
			}
			for _, child := range node.Children() {
				walk(child)
			}
		}
		walk(root)
		if nth >= len(fields) {
			return nil
		}
		return fields[nth]
	}
}

// dialogContent builds the form the open dialog shows.
func (world *world) dialogContent() Node {
	world.t.Helper()
	if world.fake.dialog == nil || world.fake.dialog.content == nil {
		world.t.Fatal("no form dialog is open")
	}
	return world.fake.dialog.content(world.ctx)
}

// typeInDialog clicks the nth text field of the dialog's form and types
// text in it.
func (world *world) typeInDialog(nth int, text string) {
	world.t.Helper()
	world.frameOver(world.dialogContent, core.Input{Click: true, Held: true}, fieldAt(nth))
	world.frameOver(world.dialogContent, core.Input{}, nil)
	world.frameOver(world.dialogContent, core.Input{Typed: []rune(text)}, nil)
}

// pressEnterInDialog presses Enter where the pointer and the focus are.
func (world *world) pressEnterInDialog() {
	world.t.Helper()
	world.frameOver(world.dialogContent, core.Input{Enter: true}, nil)
}

// widgetDialog is a dialog state with only a title.
func widgetDialog(title string) widget.DialogState { return widget.DialogState{Title: title} }

// clickInDialog presses the element of the dialog's form that shows text.
func (world *world) clickInDialog(text string) {
	world.t.Helper()
	pick := func(root Node) Node { return textNode(root, text, 0) }
	world.frameOver(world.dialogContent, core.Input{Click: true, Held: true}, pick)
	world.frameOver(world.dialogContent, core.Input{}, nil)
}
