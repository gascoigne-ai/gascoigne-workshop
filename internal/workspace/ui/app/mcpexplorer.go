package ui

import (
	"strings"

	"context"
	"encoding/json"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
	return Col(Grow(1),
		a.serverInfo(t),
		Row(Grow(1),
			a.explorerList(t),
			a.explorerDetail(t),
		),
	)
}

// serverInfo is the header of the explorer.
func (a *App) serverInfo(t *reqTab) Node {
	server := t.Server
	if server == nil {
		return Row(H(w.RowH+w.Gutter), PadX(w.Gutter), AlignItems(AlignCenter), BorderB(w.Border),
			Text(connState(t)+": connect to see what the server offers", Muted()))
	}
	return Col(PadX(w.Gutter), PadY(8), Gap(4), BorderB(w.Border),
		Row(Gap(8), AlignItems(AlignBaseline),
			Text(server.Name+" "+server.Version, Bold()),
			Text("protocol "+server.Protocol, Small(), Muted()),
		),
		If(len(server.Capabilities) > 0, Text("Offers: "+strings.Join(server.Capabilities, " · "), Small(), Muted())),
		If(server.Instructions != "", Lines(server.Instructions, w.Muted)),
	)
}

// explorerList is the strip of lists and the entries of the shown one.
func (a *App) explorerList(t *reqTab) Node {
	state := t.explorer
	items := []TabItem{{Label: "Tools", Badge: len(t.Tools)}, {Label: "Resources", Badge: len(t.Resources)}, {Label: "Prompts", Badge: len(t.Prompts)}}
	entries := mcpexplorer.Entries(state.Kind, t.Tools, t.Resources, t.Prompts)
	return Col(W(explorerListW), BorderR(w.Border),
		TabBar( // as the strips of the panes: a line under the inactive tabs
			Tabs(items, int(state.Kind), OnSelect(func(index int) { a.exploreChoose(t, mcpexplorer.Kind(index)) })),
		),
		Scroll(&t.listScroll, Grow(1),
			If(len(entries) == 0, Col(Pad(w.Gutter), Text("Nothing announced", Small(), Muted()))),
			For(entries, func(index int, entry mcpexplorer.Entry) Node {
				return ListRow(index == state.Selected, PadX(w.Gutter), PadY(6), OnClick(func() { a.exploreOpen(t, index) }),
					Col(Grow(1),
						Text(entry.Title, Ellipsis()),
						If(entry.Detail != "", Text(entry.Detail, Small(), Muted(), Ellipsis())),
					),
				)
			}),
		),
	)
}

// explorerListW is the width of the list next to the open entry.
const explorerListW = 260

// explorerDetail is the open entry: its form, the button that sends it and
// the answer.
func (a *App) explorerDetail(t *reqTab) Node {
	state := t.explorer
	entries := mcpexplorer.Entries(state.Kind, t.Tools, t.Resources, t.Prompts)
	if state.Selected == mcpexplorer.None || state.Selected >= len(entries) {
		return Col(Grow(1), Pad(w.Gutter), Text("Pick a tool, a resource or a prompt", Muted()))
	}
	return Scroll(&t.detailScroll, Grow(1), Pad(w.Gutter), Gap(10),
		Text(entries[state.Selected].Title, Bold()),
		formNode(a, t),
		Row(Gap(8), AlignItems(AlignCenter),
			PrimaryButton(callLabel(state), H(w.RowH), PadX(w.Gutter), OnClick(func() { a.exploreCall(t) }),
				Tooltip("Send it to the server (Enter in a field)")),
			If(!a.connected(t), Text("Connect first", Small(), Muted())),
			If(state.Calling, Text("Waiting for the server...", Small(), Muted())),
		),
		If(state.Result != "", resultNode(t)),
	)
}

// formNode is the argument form of the open entry, nothing for a resource.
func formNode(a *App, t *reqTab) Node {
	if t.form == nil {
		return Col()
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
	return Col(Gap(4),
		Text(heading, Small(), Bold(), Color(color)),
		Code(&t.resultView, H(explorerResultH), Bg(w.Sunken)),
	)
}

// explorerResultH is the height of the answer's view.
const explorerResultH = 240
