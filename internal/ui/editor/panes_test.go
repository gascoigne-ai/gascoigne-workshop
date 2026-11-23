package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func TestShowPaneLabeledAndPaneLabelRoundTrip(t *testing.T) {
	restLabels := []string{"Params", "Body", "Headers", "Auth", "Vars", "Tests", "Settings"}
	for _, label := range restLabels {
		t.Run(label, func(t *testing.T) {
			editor, fake := newEditor(t)
			tab := fake.set.Current()
			if !editor.ShowPaneLabeled(tab, label) {
				t.Fatalf("ShowPaneLabeled(%q) = false", label)
			}
			if got := editor.PaneLabel(tab); got != label {
				t.Errorf("PaneLabel = %q, want %q", got, label)
			}
			if got := editor.PaneTabs(tab)[editor.PaneOf(tab)].Label; got != label {
				t.Errorf("the shown pane is %q, want %q", got, label)
			}
		})
	}
}

func TestShowPaneLabeledKeepsThePaneForALabelTheTabLacks(t *testing.T) {
	editor, fake := newEditor(t)
	tab := fake.set.Current()
	editor.ShowPane(tab, 2)
	if editor.ShowPaneLabeled(tab, "Gone") {
		t.Error("ShowPaneLabeled of a label the tab lacks = true")
	}
	if got := editor.PaneOf(tab); got != 2 {
		t.Errorf("PaneOf = %d, want 2", got)
	}
}

func TestPaneLabelIsEmptyWhenTheIndexPointsNowhere(t *testing.T) {
	editor, fake := newEditor(t)
	tab := fake.set.Current()
	editor.ShowPane(tab, 99)
	if got := editor.PaneLabel(tab); got != "" {
		t.Errorf("PaneLabel = %q, want none", got)
	}
}

// The pane index points into the panes of the protocol: choosing another
// protocol closes its connection first, then starts both panes over, the
// response side through the host.
func TestChoosingAProtocolResetsBothPaneIndexes(t *testing.T) {
	editor, fake := newEditor(t)
	tab := fake.set.Current()
	editor.ShowPane(tab, 4)
	editor.Choose(tab, request.ProtocolWebSocket)
	if tab.Req.Protocol != request.ProtocolWebSocket {
		t.Errorf("protocol %q, want websocket", tab.Req.Protocol)
	}
	if got := editor.PaneOf(tab); got != 0 {
		t.Errorf("PaneOf = %d, want 0", got)
	}
	if want := []string{"disconnect", "chosen"}; !slices.Equal(fake.calls, want) {
		t.Errorf("host calls %v, want %v: the connection closes before the panes start over", fake.calls, want)
	}
}

func TestAClosedTabsPaneStateIsPrunedOnTick(t *testing.T) {
	editor, fake := newEditor(t)
	closing := fake.set.Tabs()[1]
	editor.ShowPane(closing, 3)
	fake.set.Close(1)
	editor.Tick()
	if got := editor.PaneOf(closing); got != 0 {
		t.Errorf("PaneOf a closed tab = %d, want the state dropped", got)
	}
}

func TestOfferedFollowsTheSettings(t *testing.T) {
	editor, _ := newEditor(t)
	got := editor.Offered(request.ProtocolREST)
	if want := []request.Protocol{request.ProtocolREST, request.ProtocolWebSocket, request.ProtocolMCP}; !slices.Equal(got, want) {
		t.Errorf("Offered(rest) = %v, want %v", got, want)
	}
	if got := editor.Offered(request.ProtocolGRPC); !slices.Contains(got, request.ProtocolGRPC) {
		t.Errorf("Offered(grpc) = %v, want the protocol the request already uses", got)
	}
}

func TestTheVarsPaneIsTheHostsTable(t *testing.T) {
	editor, fake := newEditor(t)
	table := kvlogic.Table{Tab: fake.set.Current(), Kind: kvlogic.Vars}
	if editor.TableRows(table) != &fake.rows {
		t.Error("the Vars rows are not the host's")
	}
	if got := editor.TableCaption(kvlogic.Vars); got != "Collection variables" {
		t.Errorf("caption %q", got)
	}
}

func TestVisibleTableFollowsThePaneShown(t *testing.T) {
	editor, fake := newEditor(t)
	tab := fake.set.Current()
	tests := []struct {
		label string
		body  request.BodyType
		want  kvlogic.Kind
		ok    bool
	}{
		{"Params", request.BodyNone, kvlogic.Params, true},
		{"Headers", request.BodyNone, kvlogic.Headers, true},
		{"Auth", request.BodyNone, 0, false},
		{"Body", request.BodyJSON, 0, false},
		{"Body", request.BodyForm, kvlogic.Form, true},
		{"Body", request.BodyMultipart, kvlogic.Form, true},
	}
	for _, test := range tests {
		tab.Req.BodyType = test.body
		editor.ShowPaneLabeled(tab, test.label)
		table, ok := editor.VisibleTable()
		if ok != test.ok || (ok && (table.Kind != test.want || table.Tab != tab)) {
			t.Errorf("%s/%s: VisibleTable = %v %v, want kind %v %v", test.label, test.body, table, ok, test.want, test.ok)
		}
	}
}

// The protocol table has one row per protocol of the model, in its order.
func TestProtocolViewsCoverEveryProtocolInOrder(t *testing.T) {
	if len(protocolViews) != len(request.Protocols) {
		t.Fatalf("%d protocol views for %d protocols", len(protocolViews), len(request.Protocols))
	}
	for i, row := range request.Protocols {
		if got := protocolViews[i].protocol; got != row.Name {
			t.Errorf("view %d is %s, want %s", i, got, row.Name)
		}
		if len(protocolViews[i].requestPanes) == 0 {
			t.Errorf("%s has no request panes", row.Name)
		}
		if row.Connection != request.OneShot && len(protocolViews[i].streamPanes) == 0 {
			t.Errorf("%s connects but has no panes to show the connection", row.Name)
		}
	}
}

// Every pane must carry what its kind needs, or drawing it would panic.
func TestPaneTablesAreComplete(t *testing.T) {
	for _, p := range requestPanes {
		ok := map[paneKind]bool{
			paneTable:      p.rows != nil,
			paneBody:       true, // the body pane is built from the request itself
			paneSettings:   true, // so is the settings pane
			paneAuth:       true,
			paneAssertions: true,
			paneMessages:   true,
		}[p.kind]
		if builders[p.kind] == nil {
			t.Errorf("request pane %q has a kind with no builder", p.label)
		}
		if !ok {
			t.Errorf("request pane %q is missing the field its kind needs", p.label)
		}
	}
}

// The history of a table is keyed by the label of its pane, and the pane
// names its table by kind: the two must agree, or the undo history of one
// table would be shared with another's.
func TestEachTablePaneLabelMatchesItsKind(t *testing.T) {
	for kind := kvlogic.Params; kind < kvlogic.KindCount; kind++ {
		if kind == kvlogic.ResponseHeaders {
			continue // the response's table is drawn by ui/response and has no pane here
		}
		shown := tablePanes[kind]
		if shown.kind != paneTable || shown.table != kind || shown.label != kind.Label() {
			t.Errorf("kind %d: pane %q (kind %d, table %d), want label %q", kind, shown.label, shown.kind, shown.table, kind.Label())
		}
	}
	for _, list := range [][]pane{requestPanes, connectionRequestPanes, grpcRequestPanes} {
		for _, shown := range list {
			if (shown.kind == paneTable) != (shown.table != kvlogic.NoTable) {
				t.Errorf("pane %q: kind %d with table %d", shown.label, shown.kind, shown.table)
			}
		}
	}
}

// Widths the pane is pinned at: the main column of a 1280px window with the
// default sidebar, and the narrowest the dividers allow; the height the
// request pane has in the test window.
const paneH = 300

var paneSizes = []struct {
	name  string
	width int
}{
	{"wide", 1020},
	{"narrow", 420},
}

// The Form look of the body pane, which the body draws around the table the
// pane table supplies through its slots, is the one pinned in the whole
// app before the table moved here: the dumps are byte-identical copies.
func TestTheFormBodyPaneLaysOutAsPinned(t *testing.T) {
	for _, size := range paneSizes {
		name := "bodypane-form-" + size.name
		t.Run(name, func(t *testing.T) {
			editor, fake := newEditor(t)
			tab := fake.set.Current()
			editor.Body.ChooseType(tab, request.BodyForm)
			editor.ShowPaneLabeled(tab, "Body")
			root := editor.PaneBody(editor.deps.Ctx, tab, &widget.ScrollState{})
			Layout(editor.deps.Ctx, root, size.width, paneH)
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

// The Schema pane follows the Proto pane, which holds the files its message
// comes from.
func TestTheSchemaPaneFollowsTheProtoPane(t *testing.T) {
	for i, shown := range grpcRequestPanes {
		if shown.label == "Proto" && grpcRequestPanes[i+1].label != "Schema" {
			t.Errorf("the pane after Proto is %q, want Schema", grpcRequestPanes[i+1].label)
			return
		}
	}
}

// Pins what differs between the composers of the protocols that connect:
// only MCP's hint lists its commands.
func TestOnlyTheMCPComposerShowsTheCommandHint(t *testing.T) {
	for _, protocol := range []request.Protocol{"websocket", "mcp", "sse", "grpc"} {
		hint := viewOf(protocol).composer.HintFor(true)
		if mcp := strings.HasPrefix(hint, "call TOOL"); mcp != (protocol == "mcp") {
			t.Errorf("%s composer hint %q", protocol, hint)
		}
	}
}

func TestParamsChangedRewritesURL(t *testing.T) {
	editor, fake := newEditor(t)
	tab := fake.set.Current()
	*tab.Req = request.Request{
		URL: "http://x/s?q=a&sort=stars",
		Params: []request.KV{
			{Name: "q", Value: "a", Enabled: true},
			{Name: "sort", Value: "stars", Enabled: false},
		},
	}
	paramsChanged(editor, tab)
	if tab.Req.URL != "http://x/s?q=a" || editor.URLBar.Field(tab).Text != tab.Req.URL {
		t.Fatalf("url = %q field = %q", tab.Req.URL, editor.URLBar.Field(tab).Text)
	}
}

// findStrip is the sideways scroll of the pane's strip of tabs that keeps
// state, nil when the view has none.
func findStrip(root Node, state *widget.ScrollState) Node {
	if fmt.Sprintf("%T", root.Behavior()) == "*widget.hscrollB" &&
		strings.Contains(fmt.Sprintf("%+v", root.Behavior()), fmt.Sprintf("state:%p", state)) {
		return root
	}
	for _, child := range root.Children() {
		if found := findStrip(child, state); found != nil {
			return found
		}
	}
	return nil
}

// Each pane's tab strip scrolls with its own state on the tab, which the
// dump does not show, and selecting a tab there shows the pane from its
// top. The handler is called straight: after a click, layout clamps a
// short pane's scroll to the top whatever the handler did. The response
// side has its own test of the same name.
func TestThePaneTabStripsSelectFromTheTopWithTheirOwnState(t *testing.T) {
	editor, fake := newEditor(t)
	tab := fake.set.Current()
	state := editor.state(tab)
	strip := findStrip(editor.RequestPane(editor.deps.Ctx, tab), &state.tabsX)
	if strip == nil {
		t.Fatalf("no tab strip with the state %p", &state.tabsX)
	}
	state.scroll.Y = 40
	strip.SelectHandler()(1)
	if active := editor.PaneOf(tab); active != 1 {
		t.Errorf("active pane %d, want 1", active)
	}
	if y := state.scroll.Y; y != 0 {
		t.Errorf("pane scrolled to %d, want the top", y)
	}
}
