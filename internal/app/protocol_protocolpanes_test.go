package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

func paneLabels(panes []pane) []string {
	labels := make([]string, len(panes))
	for i, p := range panes {
		labels[i] = p.label
	}
	return labels
}

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
			for _, label := range paneLabels(requestPanesFor(tab)) {
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
	tab.paneTab = len(requestPanes) - 1
	a.cycleProtocol(tab) // rest -> websocket
	if tab.Req.Protocol != "websocket" || tab.paneTab != 0 {
		t.Fatalf("protocol %q paneTab %d", tab.Req.Protocol, tab.paneTab)
	}
	if got := requestPanesFor(tab)[tab.paneTab].label; got != "Headers" {
		t.Fatalf("first websocket pane is %q", got)
	}
}

// The app shows every protocol the model knows, in the chip's order.
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
