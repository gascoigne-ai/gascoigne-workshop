package ui

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
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
				paneTable: p.rows != nil,
				paneCode:  p.code != nil,
				paneNote:  p.note != nil,
			}[p.kind]
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
