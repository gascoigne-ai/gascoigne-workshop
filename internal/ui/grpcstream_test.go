package ui

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func grpcTab(a *App) *reqTab {
	tab := a.cur()
	tab.req.Protocol, tab.req.Method = "grpc", "grpc.health.v1.Health/Watch"
	return tab
}

func TestAGRPCTabIsASocketOnlyWhileStreamIsOn(t *testing.T) {
	a, _ := newTestApp(t)
	tab := grpcTab(a)
	if isSocket(tab) || len(responsePanesFor(tab)) == 1 {
		t.Fatal("a gRPC tab shows the Messages pane before Stream is turned on")
	}
	a.toggleStream(tab)
	if !isSocket(tab) || len(responsePanesFor(tab)) != 1 {
		t.Fatal("Stream on should give the Messages pane and a Connect button")
	}
	a.toggleStream(tab)
	if isSocket(tab) {
		t.Fatal("Stream off should go back to single calls")
	}
}

func TestTheUnaryGRPCTabOffersAStreamChipThatTurnsConnectOn(t *testing.T) {
	a, _ := newTestApp(t)
	tab := grpcTab(a)
	if dump := view.Dump(a.urlBar(tab)); !strings.Contains(dump, "Stream") || !strings.Contains(dump, `"Send"`) {
		t.Fatalf("url bar:\n%s", dump)
	}
	a.toggleStream(tab)
	if dump := view.Dump(a.urlBar(tab)); !strings.Contains(dump, `"Connect"`) {
		t.Fatalf("url bar with Stream on:\n%s", dump)
	}
}

func TestOtherProtocolsHaveNoStreamChip(t *testing.T) {
	a, _ := newTestApp(t)
	if dump := view.Dump(a.urlBar(a.cur())); strings.Contains(dump, "Stream") {
		t.Fatalf("a REST tab has a Stream chip:\n%s", dump)
	}
}

func TestTurningStreamOffEndsTheOpenStream(t *testing.T) {
	a, _ := newTestApp(t)
	tab := grpcTab(a)
	a.toggleStream(tab)
	tab.wantOpen = true // a connection that is being opened
	a.toggleStream(tab)
	if tab.wantOpen || tab.connecting || tab.sess != nil {
		t.Fatal("the stream stayed open after Stream was turned off")
	}
}
