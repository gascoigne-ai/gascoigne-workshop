package ui

import "testing"

func paneLabels(panes []pane) []string {
	labels := make([]string, len(panes))
	for i, p := range panes {
		labels[i] = p.label
	}
	return labels
}

func TestRequestPanesFollowTheProtocol(t *testing.T) {
	tests := []struct {
		protocol string
		hidden   []string // REST-only tabs a socket has no use for
		shown    []string
	}{
		{"rest", nil, []string{"Params", "Body", "Headers", "Auth", "Vars", "Tests", "Settings"}},
		{"grpc", nil, []string{"Body", "Headers"}},
		{"websocket", []string{"Params", "Body", "Tests"}, []string{"Headers", "Auth", "Vars", "Settings"}},
		{"sse", []string{"Params", "Body", "Tests"}, []string{"Headers", "Auth", "Vars", "Settings"}},
		{"mcp", []string{"Params", "Body", "Tests"}, []string{"Headers", "Auth", "Vars", "Settings"}},
	}
	a, _ := newTestApp(t)
	tab := a.cur()
	for _, test := range tests {
		t.Run(test.protocol, func(t *testing.T) {
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
	a, _ := newTestApp(t)
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
