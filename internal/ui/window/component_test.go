package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol"
)

// Every pane must carry what its kind needs, or drawing it would panic. The
// request side is ui/editor's table, with its own test of the same name.
func TestPaneTablesAreComplete(t *testing.T) {
	for name, panes := range map[string][]pane{"response": responsePanes} {
		for _, p := range panes {
			ok := map[paneKind]bool{
				paneResponse: true,
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
	a := &Component{sends: sends.New()}
	tab := &reqTab{Tab: &tabs.Tab{Req: &request.Request{}}}
	for _, p := range responsePanes {
		if p.label == "Tests" {
			continue // it reads the assertions component, which this bare app has none of
		}
		if msg, _ := paneEmpty(a, p, tab); msg == "" {
			t.Errorf("pane %q would read a nil response", p.label)
		}
		_ = p.badge(a, tab) // must not panic with no response
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

var (
	respErr = request.Response{Err: "boom"}
	respOK  = request.Response{Status: 200, StatusText: "OK"}
)

func TestSampleCollectionIsConsistent(t *testing.T) {
	col := collection.GitHubSample()
	reg := protocol.NewRegistry()
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
			if got := a.editor.Offered(tt.current); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Offered(%q) = %v, want %v", tt.current, got, tt.want)
			}
		})
	}
}
