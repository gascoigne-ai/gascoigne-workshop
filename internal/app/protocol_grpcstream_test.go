package app

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func grpcTab(app *App) *reqTab {
	tab := app.cur()
	tab.Req.Protocol, tab.Req.Method = "grpc", "grpc.health.v1.Health/Watch"
	return tab
}

func TestAGRPCTabIsASocketOnlyWhileStreamIsOn(t *testing.T) {
	a := testApp(t)
	tab := grpcTab(a)
	if a.isSocket(tab) || len(a.responsePanesFor(tab)) == 1 {
		t.Fatal("a gRPC tab shows the Messages pane before Stream is turned on")
	}
	a.grpc.ToggleStream(tab.Tab)
	if !a.isSocket(tab) || len(a.responsePanesFor(tab)) != 1 {
		t.Fatal("Stream on should give the Messages pane and a Connect button")
	}
	a.grpc.ToggleStream(tab.Tab)
	if a.isSocket(tab) {
		t.Fatal("Stream off should go back to single calls")
	}
}

func TestTheUnaryGRPCTabOffersAStreamChipThatTurnsConnectOn(t *testing.T) {
	a := testApp(t)
	tab := grpcTab(a)
	if dump := view.Dump(a.urlbar.View(a.ctx)); !strings.Contains(dump, "Stream") || !strings.Contains(dump, `"Send"`) {
		t.Fatalf("url bar:\n%s", dump)
	}
	a.grpc.ToggleStream(tab.Tab)
	if dump := view.Dump(a.urlbar.View(a.ctx)); !strings.Contains(dump, `"Connect"`) {
		t.Fatalf("url bar with Stream on:\n%s", dump)
	}
}

func TestOtherProtocolsHaveNoStreamChip(t *testing.T) {
	a := testApp(t)
	if dump := view.Dump(a.urlbar.View(a.ctx)); strings.Contains(dump, "Stream") {
		t.Fatalf("a REST tab has a Stream chip:\n%s", dump)
	}
}

// Stream is a choice of the tab, not of the protocol: a tab that goes to
// another protocol and back still streams.
func TestAStreamingTabKeepsStreamAcrossAProtocolChange(t *testing.T) {
	a := testApp(t)
	tab := grpcTab(a)
	a.grpc.ToggleStream(tab.Tab)
	urlbarHost{a}.Choose(tab.Tab, "rest")
	urlbarHost{a}.Choose(tab.Tab, "grpc")
	if !a.isSocket(tab) {
		t.Fatal("the tab lost its stream across a protocol change")
	}
}
