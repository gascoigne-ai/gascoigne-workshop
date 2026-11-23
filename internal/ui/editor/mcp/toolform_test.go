package mcp

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func searchTool() request.MCPTool {
	return request.MCPTool{Name: "search", Schema: schemaOf(`{"type":"object","required":["query"],"properties":{
	"query":{"type":"string","description":"What to find"},
	"limit":{"type":"integer","default":10},
	"safe":{"type":"boolean"},
	"sort":{"type":"string","enum":["asc","desc"]}}}`)}
}

// schemaOf decodes the JSON text of a tool's input schema.
func schemaOf(text string) request.JSONSchema {
	schema, err := request.DecodeJSONSchema([]byte(text))
	if err != nil {
		panic(err)
	}
	return schema
}

func TestPickingAToolWithPropertiesOpensAFormWithTheDefaults(t *testing.T) {
	world := newWorld(t)
	world.pane.OpenToolForm(world.tab, searchTool())
	if world.fake.dialog == nil || world.fake.dialog.content == nil {
		t.Fatal("no form opened")
	}
	if world.fake.dialog.state.Title != "Call search" {
		t.Errorf("title %q", world.fake.dialog.state.Title)
	}
	dump := Dump(world.dialogContent())
	for _, want := range []string{`text="query *"`, `text="limit"`, `text="not sent"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the form lacks %s:\n%s", want, dump)
		}
	}
	// the defaults show in what the untouched form composes: the limit
	// fills in, an optional boolean and enum stay out
	world.fake.dialog.button(t, "Fill composer").Run()
	if got := world.fake.composed; len(got) != 1 || got[0] != `call search {"query":"","limit":10}` {
		t.Errorf("composed %q", got)
	}
}

func TestFillingTheFormComposesATypedCall(t *testing.T) {
	world := newWorld(t)
	world.pane.OpenToolForm(world.tab, searchTool())
	world.typeInDialog(0, "go")
	world.clickInDialog("not sent")
	world.clickInDialog("desc")
	world.fake.dialog.button(t, "Fill composer").Run()
	if world.fake.dialog != nil {
		t.Fatal("the form closes once the call is composed")
	}
	if got, want := world.fake.composed, `call search {"query":"go","limit":10,"safe":true,"sort":"desc"}`; len(got) != 1 || got[0] != want {
		t.Fatalf("composed %q, want %q", got, want)
	}
}

func TestAFormWithAnInvalidValueStaysOpenAndSaysWhich(t *testing.T) {
	world := newWorld(t)
	world.pane.OpenToolForm(world.tab, searchTool())
	world.typeInDialog(0, "go")
	world.typeInDialog(1, "lots")
	world.fake.dialog.button(t, "Fill composer").Run()
	if world.fake.dialog == nil {
		t.Fatal("the form closed")
	}
	if dump := Dump(world.fake.dialog.content(world.ctx)); !strings.Contains(dump, "limit must be a number") {
		t.Fatalf("the form does not say which value is wrong:\n%s", dump)
	}
	if len(world.fake.composed) != 0 {
		t.Fatalf("the composer changed to %q", world.fake.composed)
	}
}

func TestPickingAToolWithoutPropertiesComposesItRightAway(t *testing.T) {
	world := newWorld(t)
	world.pane.OpenToolForm(world.tab, request.MCPTool{Name: "ping"})
	if world.fake.dialog != nil {
		t.Fatal("a tool with no arguments needs no form")
	}
	if len(world.fake.called) != 1 || world.fake.called[0].Name != "ping" {
		t.Fatalf("composed calls %+v", world.fake.called)
	}
}

func TestCancellingTheFormClosesItAndComposesNothing(t *testing.T) {
	world := newWorld(t)
	world.pane.OpenToolForm(world.tab, searchTool())
	world.fake.dialog.button(t, "Cancel").Run()
	if world.fake.dialog != nil || len(world.fake.composed) != 0 {
		t.Fatalf("dialog %v composed %q", world.fake.dialog, world.fake.composed)
	}
}

// Enter in a field of the dialog's form fills the composer, as the Fill
// composer button does.
func TestEnterInTheDialogFormFillsTheComposer(t *testing.T) {
	world := newWorld(t)
	world.pane.OpenToolForm(world.tab, searchTool())
	world.typeInDialog(0, "go")
	world.pressEnterInDialog()
	if world.fake.dialog != nil {
		t.Fatal("the form closes once the call is composed")
	}
	if got := world.fake.composed; len(got) != 1 || got[0] != `call search {"query":"go","limit":10}` {
		t.Fatalf("composed %q", got)
	}
}
