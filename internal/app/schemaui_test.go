package app

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestSchemaRowsShowEveryFieldWithItsNesting(t *testing.T) {
	address := &model.MessageSchema{Name: "Address", Fields: []model.SchemaField{{Name: "street", Type: "string"}}}
	node := &model.MessageSchema{Name: "Node", Cut: true}
	schema := model.MessageSchema{Name: "Person", Fields: []model.SchemaField{
		{Name: "name", Type: "string"},
		{Name: "tags", Type: "string", Repeated: true},
		{Name: "status", Type: "enum Status", EnumValues: []string{"UNKNOWN", "SERVING"}},
		{Name: "labels", Type: "map<string, string>", MapKey: "string"},
		{Name: "email", Type: "string", Oneof: "contact"},
		{Name: "home", Type: "message Address", Message: address},
		{Name: "next", Type: "message Node", Message: node},
	}}
	want := []schemaRow{
		{Depth: 0, Text: "name: string"},
		{Depth: 0, Text: "tags: repeated string"},
		{Depth: 0, Text: "status: enum Status (UNKNOWN, SERVING)"},
		{Depth: 0, Text: "labels: map<string, string>"},
		{Depth: 0, Text: "email: string, oneof contact"},
		{Depth: 0, Text: "home: message Address"},
		{Depth: 1, Text: "street: string"},
		{Depth: 0, Text: "next: message Node (recursive)"},
	}
	if got := schemaRows(schema); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %+v\nwant   %+v", got, want)
	}
}

func TestTheSchemaPaneFollowsTheProtoPane(t *testing.T) {
	labels := paneLabels(grpcRequestPanes)
	for i, label := range labels {
		if label == "Proto" && labels[i+1] != "Schema" {
			t.Errorf("the pane after Proto is %q, want Schema", labels[i+1])
			return
		}
	}
}

// grpcTabOnSchemaPane is grpcTabOnProtoPane with health.proto added and
// the Check method chosen, shown on the Schema pane once it has loaded.
func grpcTabOnSchemaPane(t *testing.T) (*App, *reqTab) {
	t.Helper()
	a, tab, dir := grpcTabOnProtoPane(t)
	editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: protoFileIn(t, dir, "health.proto", nil)})
	editRequest(tab, editor.MethodChosen{Method: "grpc.health.v1.Health/Check"})
	clickText(t, a, "Schema", 0)
	for range 400 {
		a.step(core.Input{})
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
	if want := `{"service":""}`; tab.Req.Body != want || tab.body.Text != want {
		t.Errorf("body = %q, editor = %q, want %q", tab.Req.Body, tab.body.Text, want)
	}
	if !tab.Dirty() {
		t.Error("filling the body did not mark the request edited")
	}
	if got := requestPanesFor(tab)[tab.paneTab].label; got != "Body" {
		t.Errorf("pane after Fill body = %q, want Body", got)
	}
}
