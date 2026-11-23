package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestRequestPanesFollowTheProtocol(t *testing.T) {
	tests := []struct {
		protocol request.Protocol
		hidden   []string // REST-only tabs a socket has no use for
		shown    []string
	}{
		{"rest", nil, []string{"Params", "Body", "Headers", "Auth", "Vars", "Tests", "Settings"}},
		{"grpc", nil, []string{"Body", "Headers"}},
		{"websocket", []string{"Params", "Body", "Tests"}, []string{"Headers", "Auth", "Vars", "Settings"}},
		{"sse", []string{"Params", "Body", "Tests"}, []string{"Headers", "Auth", "Vars", "Settings"}},
		{"mcp", []string{"Params", "Body", "Tests"}, []string{"Headers", "Auth", "Vars", "Settings"}},
	}
	a := testApp(t)
	tab := a.cur()
	for _, test := range tests {
		t.Run(string(test.protocol), func(t *testing.T) {
			tab.Req.Protocol = test.protocol
			have := map[string]bool{}
			for _, label := range tabLabels(a.editor.PaneTabs(tab.Tab)) {
				have[label] = true
			}
			for _, label := range test.shown {
				if !have[label] {
					t.Errorf("%s has no %s tab", test.protocol, label)
				}
			}
			for _, label := range test.hidden {
				if have[label] {
					t.Errorf("%s shows the REST-only %s tab", test.protocol, label)
				}
			}
		})
	}
}

// The pane index of a tab points into the panes of its protocol, so
// choosing another protocol must not leave it past the end or on a
// different tab than the user was looking at.
func TestChoosingAProtocolShowsItsFirstRequestPane(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "rest"
	a.editor.ShowPane(tab.Tab, len(a.editor.PaneTabs(tab.Tab))-1)
	a.editor.URLBar.CycleProtocol(tab.Tab) // rest -> websocket
	if tab.Req.Protocol != "websocket" || a.editor.PaneOf(tab.Tab) != 0 {
		t.Fatalf("protocol %q pane %d", tab.Req.Protocol, a.editor.PaneOf(tab.Tab))
	}
	if got := a.editor.PaneLabel(tab.Tab); got != "Headers" {
		t.Fatalf("first websocket pane is %q", got)
	}
}
