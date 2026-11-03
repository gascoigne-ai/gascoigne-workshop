package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
)

func TestNext(t *testing.T) {
	tests := []struct {
		name, cur, want string
	}{
		{"advances", "rest", "websocket"},
		{"wraps", "mcp", "rest"},
		{"unknown starts over", "carrier-pigeon", "rest"},
	}
	for _, tt := range tests {
		if got := next(protocolOrder, tt.cur); got != tt.want {
			t.Errorf("%s: next(%q) = %q, want %q", tt.name, tt.cur, got, tt.want)
		}
	}
}

func TestSplitVars(t *testing.T) {
	tests := []struct {
		in   string
		want []urlSeg
	}{
		{"", nil},
		{"http://x", []urlSeg{{text: "http://x"}}},
		{"{{baseUrl}}/a", []urlSeg{{text: "{{baseUrl}}", isVar: true}, {text: "/a"}}},
		{"a{{x}}b{{y}}", []urlSeg{{text: "a"}, {text: "{{x}}", isVar: true}, {text: "b"}, {text: "{{y}}", isVar: true}}},
		{"half {{open", []urlSeg{{text: "half {{open"}}},
	}
	for _, tt := range tests {
		if got := splitVars(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitVars(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

// Every pane must carry what its kind needs, or drawing it would panic.
func TestPaneTablesAreComplete(t *testing.T) {
	for name, panes := range map[string][]pane{"request": requestPanes, "response": responsePanes} {
		for _, p := range panes {
			ok := map[paneKind]bool{
				paneTable:    p.rows != nil,
				paneCode:     p.code != nil,
				paneNote:     p.note != nil,
				paneBody:     true, // the body pane is built from the request itself
				paneSettings: true, // so is the settings pane
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
	tab := &reqTab{req: &model.Request{}}
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
	tab := newTab(&model.Request{
		URL: "http://x/s?q=a&sort=stars",
		Params: []model.KV{
			{Name: "q", Value: "a", Enabled: true},
			{Name: "sort", Value: "stars", Enabled: false},
		},
	})
	paramsChanged(tab)
	if tab.req.URL != "http://x/s?q=a" || tab.url.Text != tab.req.URL {
		t.Fatalf("url = %q field = %q", tab.req.URL, tab.url.Text)
	}
}

func TestRequestBadge(t *testing.T) {
	tests := []struct{ protocol, method, want string }{
		{"rest", "GET", "GET"}, {"", "POST", "POST"}, {"rest", "DELETE", "DEL"},
		{"websocket", "", "WS"}, {"mcp", "GET", "MCP"},
	}
	for _, tt := range tests {
		if got := requestBadge(tt.protocol, tt.method); got != tt.want {
			t.Errorf("requestBadge(%q,%q) = %q, want %q", tt.protocol, tt.method, got, tt.want)
		}
	}
}

func TestSampleCollectionIsConsistent(t *testing.T) {
	col := SampleCollection()
	reg := proto.NewRegistry()
	seen := map[string]bool{}
	for _, row := range treeRows(col.Folder, allExpanded(col.Folder)) {
		if seen[row.Path] {
			t.Errorf("duplicate tree path %q", row.Path)
		}
		seen[row.Path] = true
		if row.Req == nil {
			continue
		}
		if _, ok := reg[row.Req.Protocol]; !ok {
			t.Errorf("%s uses unknown protocol %q", row.Path, row.Req.Protocol)
		}
		if row.Req.Protocol == "rest" && row.Req.Method == "" {
			t.Errorf("%s is REST without a method", row.Path)
		}
	}
	if len(col.Vars) == 0 {
		t.Error("the sample must define baseUrl")
	}
}

func allExpanded(f model.Folder) map[string]bool {
	open := map[string]bool{f.Name: true}
	var walk func(model.Folder, string)
	walk = func(f model.Folder, path string) {
		for _, sub := range f.Folders {
			p := path + "/" + sub.Name
			open[p] = true
			walk(sub, p)
		}
	}
	walk(f, f.Name)
	return open
}

func TestKVRows(t *testing.T) {
	rows := []model.KV{
		{Name: "q", Value: "x", Description: "d", Enabled: true},
		{Name: "off", Value: "y"},
	}
	two := kvRows(rows, false)
	three := kvRows(rows, true)
	if len(two[0].Cells) != 2 || len(three[0].Cells) != 3 || three[0].Cells[2] != "d" {
		t.Fatalf("cells: %v / %v", two[0].Cells, three[0].Cells)
	}
	if !two[0].Checked || two[0].Muted || two[1].Checked || !two[1].Muted {
		t.Fatalf("enabled rows are checked and bright, disabled ones muted: %+v", two)
	}
}

var (
	respErr = model.Response{Err: "boom"}
	respOK  = model.Response{Status: 200, StatusText: "OK"}
)

func TestJSONStatus(t *testing.T) {
	tests := []struct {
		name, in  string
		wantValid bool
		wantMsg   string // substring; "" means no message at all
	}{
		{"empty body has nothing to report", "", false, ""},
		{"whitespace only too", "  \n", false, ""},
		{"valid object", `{"a": [1, 2]}`, true, "Valid JSON"},
		{"valid scalar", `42`, true, "Valid JSON"},
		{"invalid", `{"a": }`, false, "Invalid JSON"},
		{"truncated", `{"a": 1`, false, "Invalid JSON"},
	}
	for _, tt := range tests {
		msg, valid := jsonStatus(tt.in)
		if valid != tt.wantValid || (tt.wantMsg == "") != (msg == "") || !strings.Contains(msg, tt.wantMsg) {
			t.Errorf("%s: got %q %v; want %q %v", tt.name, msg, valid, tt.wantMsg, tt.wantValid)
		}
	}
}
