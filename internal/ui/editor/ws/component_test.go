package ws

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var sentAt = time.Date(2026, 11, 14, 10, 30, 0, 0, time.UTC)

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

// The initialize result is already in the log as the response to the
// initialize request; the copy that feeds the server info is not a line.
func TestLogSkipsTheServerInfoEvent(t *testing.T) {
	world := newWorld(t, request.ProtocolMCP)
	world.pane.Log(world.tab, request.Event{Dir: request.Info, Kind: request.KindInitialize, Text: `{"protocolVersion":"2025-06-18"}`})
	if got := len(world.pane.Lines(world.tab)); got != 0 {
		t.Fatalf("log has %d lines", got)
	}
}

// A log line's direction mark is an icon span: an arrow up for sent, an
// arrow down for received and a dot for a note, with the character kept
// as its text for copy.
func TestLogLineDirectionMarksAreIcons(t *testing.T) {
	cases := []struct {
		dir        request.Direction
		icon, text string
	}{
		{request.Sent, "arrow-up", "↑"},
		{request.Received, "arrow-down", "↓"},
		{request.Info, "dot", "•"},
	}
	for _, tc := range cases {
		world := newWorld(t, request.ProtocolWebSocket)
		world.pane.Log(world.tab, request.Event{Dir: tc.dir, Kind: request.KindText, Text: "x"})
		lines := world.pane.Lines(world.tab)
		if len(lines) == 0 {
			t.Fatalf("%v: nothing was logged", tc.dir)
		}
		mark := lines[len(lines)-1].Spans[1]
		if mark.Icon != tc.icon || mark.Text != tc.text {
			t.Errorf("%v: mark %+v, want icon %q and text %q", tc.dir, mark, tc.icon, tc.text)
		}
	}
}

func TestALogLineOfAMultiLineTextIsOneLinePerLine(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.pane.Log(world.tab, request.Event{Time: sentAt, Dir: request.Received, Kind: request.KindText, Text: "one\r\ntwo"})
	if got, want := world.logText(), "10:30:00 ↓ one\n           two\n"; got != want {
		t.Fatalf("log %q, want %q", got, want)
	}
}

func TestANoteOfAnErrorColorsItsMark(t *testing.T) {
	cases := []struct {
		level     severity.Level
		wantStyle w.Token
	}{
		{severity.Error, w.Error},
		{severity.Info, w.Muted},
	}
	for _, tc := range cases {
		world := newWorld(t, request.ProtocolWebSocket)
		world.pane.Note(world.tab, "boom", tc.level)
		lines := world.pane.Lines(world.tab)
		if len(lines) != 1 {
			t.Fatalf("level %v: %d lines, want 1", tc.level, len(lines))
		}
		if got := lines[0].Spans[1].Style; got != tc.wantStyle {
			t.Errorf("level %v: mark style %v, want %v", tc.level, got, tc.wantStyle)
		}
		if !strings.Contains(world.logText(), "boom") {
			t.Errorf("level %v: the note is not in the log: %q", tc.level, world.logText())
		}
	}
}

func TestClearEmptiesTheLog(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.pane.Note(world.tab, "one", severity.Info)
	world.pane.Clear(world.tab)
	if got := len(world.pane.Lines(world.tab)); got != 0 {
		t.Fatalf("the log keeps %d lines", got)
	}
}

func TestTheLogFollowsItsEndFromTheStart(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	if !world.pane.state(world.tab).log.Follow {
		t.Fatal("a new log must follow its end")
	}
}

func TestFormatMessageMakesTheComposerOneLineOfJSON(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.pane.Composer(world.tab).Text = "{\n  \"op\": \"ping\",\n  \"n\": 1\n}"
	world.pane.formatMessage(world.tab)
	if got := world.pane.Composer(world.tab).Text; got != `{"op":"ping","n":1}` {
		t.Fatalf("composer = %q", got)
	}
}

func TestFormatMessageLeavesBrokenTextAloneAndSaysWhy(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.pane.Composer(world.tab).Text = `{"op": }`
	world.pane.formatMessage(world.tab)
	if got := world.pane.Composer(world.tab).Text; got != `{"op": }` {
		t.Fatalf("composer = %q, broken text must stay as typed", got)
	}
	toasts := world.fake.toasts
	if len(toasts) != 1 || !strings.Contains(toasts[0].text, "not JSON") || toasts[0].level != severity.Error {
		t.Fatalf("toasts = %+v", toasts)
	}
}

func TestMCPResourcesAndPromptsFillTheComposer(t *testing.T) {
	world := newWorld(t, request.ProtocolMCP)
	world.tab.Received(listEvent(`{"result":{"resources":[{"uri":"file:///a.txt","name":"A"}]}}`))
	world.tab.Received(listEvent(`{"result":{"prompts":[{"name":"review","arguments":[{"name":"code","required":true}]}]}}`))
	world.pane.resourceMenu(world.tab)[0].Run()
	if got := world.pane.Composer(world.tab).Text; got != "read file:///a.txt" {
		t.Fatalf("composer %q", got)
	}
	world.pane.promptMenu(world.tab)[0].Run()
	if got := world.pane.Composer(world.tab).Text; got != `prompt review {"code":""}` {
		t.Fatalf("composer %q", got)
	}
	if !world.pane.Composer(world.tab).Focused {
		t.Fatal("a filled composer takes the keyboard")
	}
}

func TestComposeCallFillsTheComposerWithTheSchemasSkeleton(t *testing.T) {
	cases := []struct {
		name string
		tool request.MCPTool
		want string
	}{
		{"no schema", request.MCPTool{Name: "ping"}, "call ping {}"},
		{"with properties", request.MCPTool{Name: "search", Schema: request.JSONSchema{Type: "object", Properties: []request.SchemaProperty{{Name: "q", Required: true, Schema: request.JSONSchema{Type: "string"}}}}}, `call search {"q":""}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t, request.ProtocolMCP)
			world.pane.ComposeCall(world.tab, tc.tool)
			if got := world.pane.Composer(world.tab).Text; got != tc.want {
				t.Fatalf("composer %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSendMessageQueuesItAndEmptiesTheComposer(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	session := &recorder{}
	world.open(session)
	world.pane.Composer(world.tab).Text = "hello"
	world.pane.SendMessage(world.tab)
	if len(world.fake.queued) != 1 {
		t.Fatalf("%d sends queued, want 1", len(world.fake.queued))
	}
	world.fake.runAll(t)
	if got := session.log(); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("sent %q, want hello", got)
	}
	composer := world.pane.Composer(world.tab)
	if composer.Text != "" || !composer.Focused {
		t.Fatalf("composer %q focused %v, want empty and focused", composer.Text, composer.Focused)
	}
}

func TestSendWithoutSessionQueuesNothing(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.pane.Composer(world.tab).Text = "hello"
	world.pane.SendMessage(world.tab)
	if len(world.fake.queued) != 0 {
		t.Fatalf("%d sends queued without a session", len(world.fake.queued))
	}
	if got := world.pane.Composer(world.tab).Text; got != "hello" {
		t.Fatalf("composer %q, the draft must stay", got)
	}
}

func TestSendOfAnEmptyComposerQueuesNothing(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.open(&recorder{})
	world.pane.SendMessage(world.tab)
	if len(world.fake.queued) != 0 {
		t.Fatalf("%d sends queued for an empty composer", len(world.fake.queued))
	}
}

func TestPingQueuesThePing(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	session := &recorder{}
	world.open(session)
	world.pane.Ping(world.tab)
	world.fake.runAll(t)
	if got := session.log(); len(got) != 1 || got[0] != "ping" {
		t.Fatalf("sent %q, want ping", got)
	}
}

func TestPingWithoutSessionQueuesNothing(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.pane.Ping(world.tab)
	if len(world.fake.queued) != 0 {
		t.Fatalf("%d sends queued without a session", len(world.fake.queued))
	}
}

func TestHexGoesOutAsABinaryMessage(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	session := &binaryRecorder{}
	world.open(session)
	world.pane.Composer(world.tab).Text = "hex: de ad BE ef"
	world.pane.SendMessage(world.tab)
	world.fake.runAll(t)
	if len(session.binary) != 1 || string(session.binary[0]) != "\xde\xad\xbe\xef" {
		t.Fatalf("binary %x, want deadbeef", session.binary)
	}
	if len(session.log()) != 0 {
		t.Fatalf("a hex message also went out as text: %q", session.log())
	}
}

func TestBadHexIsRefusedBeforeAnythingIsQueued(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	world.open(&binaryRecorder{})
	world.pane.Composer(world.tab).Text = "hex: xyz"
	world.pane.SendMessage(world.tab)
	if len(world.fake.queued) != 0 {
		t.Fatalf("%d sends queued for bad hex", len(world.fake.queued))
	}
	if !strings.Contains(world.logText(), "hex:") {
		t.Fatalf("a bad hex message must be explained in the log:\n%s", world.logText())
	}
	if got := world.pane.Composer(world.tab).Text; got != "hex: xyz" {
		t.Fatalf("composer %q, the draft must stay", got)
	}
}

func TestTypingIsOnlyTheCurrentTabsComposer(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second", Protocol: request.ProtocolWebSocket})
	second := world.fake.set.Current()
	if world.pane.Typing() {
		t.Fatal("typing before any composer was made")
	}
	if len(world.pane.states) != 0 {
		t.Fatal("Typing must not make state")
	}
	world.pane.Composer(first).Focused = true
	if world.pane.Typing() {
		t.Fatal("the focused composer is on a tab in the background")
	}
	world.pane.Composer(second).Focused = true
	if !world.pane.Typing() {
		t.Fatal("the current tab's composer has the keyboard")
	}
}

func TestLinesMakeNoState(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	if world.pane.Lines(world.tab) != nil || len(world.pane.states) != 0 {
		t.Fatal("reading the lines of a tab made state")
	}
}

func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second", Protocol: request.ProtocolWebSocket})
	world.pane.Note(first, "kept until the tab closes", severity.Info)
	world.fake.set.Apply(tabs.Close{Index: 0})
	if world.pane.Tick() {
		t.Fatal("dropping a state changes nothing on screen")
	}
	if _, kept := world.pane.states[first]; kept {
		t.Fatal("the state of a closed tab is kept")
	}
}

func TestAnEventOfAHiddenTabLandsInItsLog(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second", Protocol: request.ProtocolWebSocket})
	world.pane.Log(first, request.Event{Time: sentAt, Dir: request.Received, Kind: request.KindText, Text: "late"})
	if got := len(world.pane.Lines(first)); got != 1 {
		t.Fatalf("hidden tab's log has %d lines, want 1", got)
	}
}

func TestEveryConnectionStateHasALabel(t *testing.T) {
	for state := tabs.ConnIdle; state <= tabs.ConnRetrying; state++ {
		if StateText(state) == "" {
			t.Errorf("state %d has no label", state)
		}
	}
}

func TestTheConnectionBarSaysWhereTheConnectionIs(t *testing.T) {
	cases := map[tabs.ConnState]string{
		tabs.ConnIdle:       "Not connected",
		tabs.ConnConnecting: "Connecting...",
		tabs.ConnOpen:       "Connected",
		tabs.ConnClosing:    "Connected",
		tabs.ConnRetrying:   "Not connected",
	}
	for state, want := range cases {
		if got := StateText(state); got != want {
			t.Errorf("state %d: %q, want %q", state, got, want)
		}
	}
}

func TestTheComposerHintFollowsTheProtocolAndTheConnection(t *testing.T) {
	cases := []struct {
		name      string
		composer  Composer
		connected bool
		want      string
	}{
		{"not connected", Composer{Shown: true, Hint: "own"}, false, "Connect to send messages"},
		{"connected, no hint of its own", Composer{Shown: true}, true, "Message"},
		{"connected, own hint", Composer{Shown: true, Hint: "own"}, true, "own"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.composer.HintFor(tc.connected); got != tc.want {
				t.Fatalf("hint %q, want %q", got, tc.want)
			}
		})
	}
}
