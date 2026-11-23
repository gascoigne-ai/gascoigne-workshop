package grpc

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// The picker lists the methods in a menu at the click, and the chosen one
// becomes the method of the request.
func TestPickingAMethodListsTheServersMethodsAndSetsIt(t *testing.T) {
	world := newWorld(t)
	world.ctx.In = core.Input{X: 30, Y: 40}
	world.pane.PickMethod(world.ctx, world.tab)
	world.ctx.In = core.Input{X: 500, Y: 600} // the pointer moves while the server answers
	world.answer([]string{"pkg.A/One", "pkg.A/Two"}, nil)
	if len(world.fake.menus) != 1 {
		t.Fatalf("menus opened: %d, want 1", len(world.fake.menus))
	}
	menu := world.fake.menus[0]
	if menu.posX != 30 || menu.posY != 40 {
		t.Errorf("menu at %d,%d, want where the click was", menu.posX, menu.posY)
	}
	var labels []string
	for _, item := range menu.items {
		labels = append(labels, item.Label)
	}
	if want := []string{"pkg.A/One", "pkg.A/Two"}; !reflect.DeepEqual(labels, want) {
		t.Fatalf("menu %v, want %v", labels, want)
	}
	menu.items[1].Run()
	if world.tab.Req.Method != "pkg.A/Two" {
		t.Fatalf("Method = %q", world.tab.Req.Method)
	}
}

// The server is asked as the send would: {{variables}} replaced.
func TestThePickerAsksForTheRequestWithItsVariablesFilledIn(t *testing.T) {
	world := newWorld(t)
	world.fake.vars = []request.KV{{Name: "host", Value: "example.test:1", Enabled: true}}
	world.tab.Req.URL = "grpc://{{host}}"
	world.pane.PickMethod(world.ctx, world.tab)
	if len(world.fake.listed) != 1 || world.fake.listed[0].URL != "grpc://example.test:1" {
		t.Fatalf("listed %+v, want the expanded URL", world.fake.listed)
	}
	if want := []string{"Listing methods of grpc://example.test:1"}; !reflect.DeepEqual(world.fake.statuses, want) {
		t.Errorf("status %v, want %v", world.fake.statuses, want)
	}
}

func TestAFailedListingIsToastedAndOpensNoMenu(t *testing.T) {
	world := newWorld(t)
	world.pane.PickMethod(world.ctx, world.tab)
	world.answer(nil, errors.New("connection refused"))
	if want := []toast{{"Listing methods failed: connection refused", severity.Error}}; !reflect.DeepEqual(world.fake.toasts, want) {
		t.Errorf("toasts %v, want %v", world.fake.toasts, want)
	}
	if len(world.fake.menus) != 0 {
		t.Errorf("a menu opened after a failure: %v", world.fake.menus)
	}
}

func TestAGRPCPickerWithoutAServerSaysSo(t *testing.T) {
	world := newWorld(t)
	world.tab.Req.URL = "  "
	world.pane.PickMethod(world.ctx, world.tab)
	if len(world.fake.toasts) != 1 || world.fake.toasts[0].level != severity.Error {
		t.Fatalf("toasts %v, want one error that a server is needed", world.fake.toasts)
	}
	if len(world.fake.listed) != 0 || len(world.fake.menus) != 0 {
		t.Fatalf("asked %d times, menus %d, want neither", len(world.fake.listed), len(world.fake.menus))
	}
}

// Proto files are read without a server: no URL has to be entered.
func TestPickingAMethodFromProtoFilesNeedsNoServer(t *testing.T) {
	world := newWorld(t)
	world.tab.Req.URL = ""
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("health.proto")})
	world.pane.PickMethod(world.ctx, world.tab)
	if len(world.fake.toasts) != 0 || len(world.fake.listed) != 1 {
		t.Fatalf("toasts %v, listed %d, want the files read", world.fake.toasts, len(world.fake.listed))
	}
}

// The stream of a tab ends before the mode flips: it belongs to the mode
// that opened it.
func TestTurningStreamOffEndsTheOpenStream(t *testing.T) {
	world := newWorld(t)
	world.pane.ToggleStream(world.tab)
	world.pane.ToggleStream(world.tab)
	if want := []bool{false, true}; !reflect.DeepEqual(world.fake.streamingAtDisconnect, want) {
		t.Fatalf("streaming when the stream was ended: %v, want %v (ended before each flip)", world.fake.streamingAtDisconnect, want)
	}
	if world.pane.Streaming(world.tab) {
		t.Fatal("Stream is still on")
	}
}

// Streaming is a choice of each tab.
func TestStreamingIsPerTab(t *testing.T) {
	world := newWorld(t)
	world.fake.set.Open(&request.Request{Name: "other", Protocol: request.ProtocolGRPC})
	other := world.fake.set.Current()
	world.pane.ToggleStream(world.tab)
	if !world.pane.Streaming(world.tab) || world.pane.Streaming(other) {
		t.Fatalf("streaming: first %v, second %v, want only the first", world.pane.Streaming(world.tab), world.pane.Streaming(other))
	}
}

// Asking about a tab makes no state for it: only drawing the pane and
// acting on it do.
func TestAskingAboutATabMakesNoState(t *testing.T) {
	world := newWorld(t)
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("a.proto")})
	if world.pane.Streaming(world.tab) {
		t.Error("a tab nobody switched streams")
	}
	if !world.pane.ProtoPending(world.tab) {
		t.Error("a tab with proto files nobody read is not pending")
	}
	world.pane.Tick()
	if len(world.pane.states) != 0 {
		t.Fatalf("states %d, want none", len(world.pane.states))
	}
}

func TestTheProtoFilesArePendingUntilTheyAreRead(t *testing.T) {
	world := newWorld(t)
	if world.pane.ProtoPending(world.tab) {
		t.Error("a request without proto files has nothing to read")
	}
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("a.proto")})
	world.view()
	if !world.pane.ProtoPending(world.tab) {
		t.Error("the files are being read, not pending")
	}
	world.answer(nil, nil)
	if world.pane.ProtoPending(world.tab) {
		t.Error("the files were read and the pane still waits")
	}
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("b.proto")})
	if !world.pane.ProtoPending(world.tab) {
		t.Error("an edited source is not read yet")
	}
}

func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t)
	world.pane.ToggleStream(world.tab)
	if changed := world.pane.Tick(); changed || len(world.pane.states) != 1 {
		t.Fatalf("changed %v, states %d, want the open tab's state kept", changed, len(world.pane.states))
	}
	world.fake.set.Close(0)
	if changed := world.pane.Tick(); changed || len(world.pane.states) != 0 {
		t.Fatalf("changed %v, states %d, want the closed tab's state dropped", changed, len(world.pane.states))
	}
}
