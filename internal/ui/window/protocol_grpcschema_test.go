package window

import (
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// grpcTabOnSchemaPane is grpcTabOnProtoPane with health.proto added and
// the Check method chosen, shown on the Schema pane once it has loaded.
func grpcTabOnSchemaPane(t *testing.T) (*Component, *reqTab) {
	t.Helper()
	a, tab, dir := grpcTabOnProtoPane(t)
	editRequest(tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: protoFileIn(t, dir, "health.proto", nil)})
	editRequest(tab, requestedit.MethodChosen{Method: "grpc.health.v1.Health/Check"})
	clickText(t, a, "Schema", 0)
	for range 400 {
		a.Step(core.Input{})
		if strings.Contains(layoutDump(a), `text="service: string"`) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	return a, tab
}

func TestTheSchemaPaneShowsTheFieldsOfTheChosenMethod(t *testing.T) {
	a, _ := grpcTabOnSchemaPane(t)
	dump := layoutDump(a)
	for _, want := range []string{`text="grpc.health.v1.HealthCheckRequest"`, `text="service: string"`, `text="Fill body"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the pane lacks %s", want)
		}
	}
}

func TestTheSchemaPaneAsksForAMethodFirst(t *testing.T) {
	a, _, _ := grpcTabOnProtoPane(t)
	clickText(t, a, "Schema", 0)
	dump := layoutDump(a)
	if !strings.Contains(dump, "Pick a method") {
		t.Errorf("no prompt to pick a method:\n%s", dump)
	}
	if strings.Contains(dump, `text="Fill body"`) {
		t.Error("Fill body shows with no method")
	}
}

func TestFillBodyWritesTheSkeletonIntoTheBodyAndShowsIt(t *testing.T) {
	a, tab := grpcTabOnSchemaPane(t)
	clickText(t, a, "Fill body", 0)
	if want := `{"service":""}`; tab.Req.Body != want || a.editor.Body.Area(tab.Tab).Text != want {
		t.Errorf("body = %q, editor = %q, want %q", tab.Req.Body, a.editor.Body.Area(tab.Tab).Text, want)
	}
	if !tab.Dirty() {
		t.Error("filling the body did not mark the request edited")
	}
	if got := a.editor.PaneLabel(tab.Tab); got != "Body" {
		t.Errorf("pane after Fill body = %q, want Body", got)
	}
}
