package app

import (
	"path/filepath"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func TestNext(t *testing.T) {
	chipOrder := []request.Protocol{"rest", "websocket", "mcp", "sse", "grpc"}
	tests := []struct {
		name      string
		cur, want request.Protocol
	}{
		{"advances", "rest", "websocket"},
		{"wraps", "grpc", "rest"},
		{"unknown starts over", "carrier-pigeon", "rest"},
	}
	for _, tt := range tests {
		if got := next(chipOrder, tt.cur); got != tt.want {
			t.Errorf("%s: next(%q) = %q, want %q", tt.name, tt.cur, got, tt.want)
		}
	}
}

// Every pane must carry what its kind needs, or drawing it would panic.
func TestPaneTablesAreComplete(t *testing.T) {
	for name, panes := range map[string][]pane{"request": requestPanes, "response": responsePanes} {
		for _, p := range panes {
			ok := map[paneKind]bool{
				paneTable:      p.rows != nil,
				paneNote:       p.note != nil,
				paneBody:       true, // the body pane is built from the request itself
				paneSettings:   true, // so is the settings pane
				paneAuth:       true,
				paneResponse:   true,
				paneAssertions: true,
				paneResults:    true,
				paneMessages:   true,
			}[p.kind]
			if paneBuilders[p.kind] == nil {
				t.Errorf("%s pane %q has a kind with no builder", name, p.label)
			}
			if !ok {
				t.Errorf("%s pane %q is missing the field its kind needs", name, p.label)
			}
		}
	}
}

func TestResponsePanesNeverTouchNilResponse(t *testing.T) {
	a := &App{}
	tab := &reqTab{Tab: &tabs.Tab{Req: &request.Request{}}}
	for _, p := range responsePanes {
		msg, _ := paneEmpty(p, tab)
		if p.kind == paneNote && msg == "" && p.label != "Tests" {
			t.Errorf("pane %q would read a nil response", p.label)
		}
		if p.kind == paneTable {
			_ = p.badge(a, tab) // must not panic with no response
		}
	}
}

func TestParamsChangedRewritesURL(t *testing.T) {
	tab := newTab(&request.Request{
		URL: "http://x/s?q=a&sort=stars",
		Params: []request.KV{
			{Name: "q", Value: "a", Enabled: true},
			{Name: "sort", Value: "stars", Enabled: false},
		},
	})
	paramsChanged(tab)
	if tab.Req.URL != "http://x/s?q=a" || tab.url.Text != tab.Req.URL {
		t.Fatalf("url = %q field = %q", tab.Req.URL, tab.url.Text)
	}
}

func TestRequestBadge(t *testing.T) {
	tests := []struct {
		protocol     request.Protocol
		method, want string
	}{
		{"rest", "GET", "GET"}, {"", "POST", "POST"}, {"rest", "DELETE", "DEL"},
		{"websocket", "", "WS"}, {"mcp", "GET", "MCP"},
	}
	for _, tt := range tests {
		if got := requestBadge(tt.protocol, tt.method); got != tt.want {
			t.Errorf("requestBadge(%q,%q) = %q, want %q", tt.protocol, tt.method, got, tt.want)
		}
	}
}

func folderNames(folder *request.Folder) (out []string) {
	for _, s := range folder.Folders {
		out = append(out, s.Name)
	}
	return
}

func requestNames(folder *request.Folder) (out []string) {
	for _, r := range folder.Requests {
		out = append(out, r.Name)
	}
	return
}

func TestKVRows(t *testing.T) {
	rows := []request.KV{
		{Name: "q", Value: "x", Description: "d", Enabled: true},
		{Name: "off", Value: "y"},
	}
	two := kvRows(rows, false, false)
	three := kvRows(rows, true, false)
	if len(two[0].Cells) != 2 || len(three[0].Cells) != 3 || three[0].Cells[2] != "d" {
		t.Fatalf("cells: %v / %v", two[0].Cells, three[0].Cells)
	}
	if !two[0].Checked || two[0].Muted || two[1].Checked || !two[1].Muted {
		t.Fatalf("enabled rows are checked and bright, disabled ones muted: %+v", two)
	}
}

var (
	respErr = request.Response{Err: "boom"}
	respOK  = request.Response{Status: 200, StatusText: "OK"}
)

func TestSampleCollectionIsConsistent(t *testing.T) {
	col := collection.GitHubSample()
	reg := proto.NewRegistry()
	var check func(folder *request.Folder, path string)
	check = func(folder *request.Folder, path string) {
		seen := map[string]bool{}
		for _, name := range append(folderNames(folder), requestNames(folder)...) {
			if seen[name] {
				t.Errorf("%s: duplicate name %q among siblings", path, name)
			}
			seen[name] = true
		}
		for _, r := range folder.Requests {
			if _, ok := reg[r.Protocol]; !ok {
				t.Errorf("%s/%s uses unknown protocol %q", path, r.Name, r.Protocol)
			}
			if r.Protocol == "rest" && r.Method == "" {
				t.Errorf("%s/%s is REST without a method", path, r.Name)
			}
		}
		for _, sub := range folder.Folders {
			check(sub, path+"/"+sub.Name)
		}
	}
	check(&col.Folder, col.Name)
	if len(col.Vars) == 0 {
		t.Error("the sample must define baseUrl")
	}
}

func TestTheProtocolChipOffersGRPCAndSSEOnlyWhenTheyAreSwitchedOn(t *testing.T) {
	tests := []struct {
		name      string
		grpc, sse bool
		current   request.Protocol
		want      []request.Protocol
	}{
		{"both off", false, false, "rest", []request.Protocol{"rest", "websocket", "mcp"}},
		{"sse on", false, true, "rest", []request.Protocol{"rest", "websocket", "mcp", "sse"}},
		{"grpc on", true, false, "rest", []request.Protocol{"rest", "websocket", "mcp", "grpc"}},
		{"both on", true, true, "rest", []request.Protocol{"rest", "websocket", "mcp", "sse", "grpc"}},
		{"a request already on grpc keeps it", false, false, "grpc", []request.Protocol{"rest", "websocket", "mcp", "grpc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := testApp(t)
			a.settings.GRPC, a.settings.SSE = tt.grpc, tt.sse
			if got := a.protocolsFor(tt.current); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("protocolsFor(%q) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}

// fakeKeyRepeat is a desktop with a fixed key repeat setting.
type fakeKeyRepeat struct{ delay, interval time.Duration }

func (fake fakeKeyRepeat) Timing() (time.Duration, time.Duration) { return fake.delay, fake.interval }

func TestTheWindowRepeatsKeysAtTheDesktopsSpeed(t *testing.T) {
	want := core.KeyRepeat{Delay: 600 * time.Millisecond, Interval: 25 * time.Millisecond}
	app, err := NewApp(withAdapters(Options{
		SettingsPath: filepath.Join(t.TempDir(), "settings.json"),
		KeyRepeat:    fakeKeyRepeat{want.Delay, want.Interval},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if app.sampler.Repeat != want {
		t.Fatalf("key repeat = %+v, want %+v", app.sampler.Repeat, want)
	}
}
