package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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

func dialogButtonNamed(t *testing.T, app *App, label string) widget.DialogButton {
	t.Helper()
	for _, button := range app.dialog.Buttons {
		if button.Label == label {
			return button
		}
	}
	t.Fatalf("no %q button in %+v", label, app.dialog.Buttons)
	return widget.DialogButton{}
}

func TestPickingAToolWithPropertiesOpensAFormWithTheDefaults(t *testing.T) {
	a := testApp(t)
	a.openToolForm(a.cur(), searchTool())
	if a.dialog == nil || a.dialog.form == nil {
		t.Fatal("no form opened")
	}
	form := a.dialog.form
	if got := form.Inputs["limit"].Text; got != "10" {
		t.Errorf("the default fills the field, got %q", got)
	}
	if got := form.Inputs["query"].Text; got != "" {
		t.Errorf("query = %q", got)
	}
	if form.Choices["safe"] != "" || form.Choices["sort"] != "" {
		t.Errorf("an optional boolean and an optional enum start unset: %v", form.Choices)
	}
}

func TestFillingTheFormComposesATypedCall(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	a.openToolForm(tab, searchTool())
	form := a.dialog.form
	form.Inputs["query"].Text = "go"
	form.Inputs["limit"].Text = "5"
	form.Choices["safe"] = "true"
	form.Choices["sort"] = "desc"
	dialogButtonNamed(t, a, "Fill composer").Run()
	if a.dialog != nil {
		t.Fatal("the form closes once the call is composed")
	}
	if want := `call search {"query":"go","limit":5,"safe":true,"sort":"desc"}`; tab.msg.Text != want {
		t.Fatalf("composer = %q, want %q", tab.msg.Text, want)
	}
}

func TestAFormWithAnInvalidValueStaysOpenAndSaysWhich(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.msg.Text = "unchanged"
	a.openToolForm(tab, searchTool())
	form := a.dialog.form
	form.Inputs["query"].Text = "go"
	form.Inputs["limit"].Text = "lots"
	dialogButtonNamed(t, a, "Fill composer").Run()
	if a.dialog == nil || !strings.Contains(form.Problem, "limit must be a number") {
		t.Fatalf("dialog %v problem %q", a.dialog, form.Problem)
	}
	if tab.msg.Text != "unchanged" {
		t.Fatalf("the composer changed to %q", tab.msg.Text)
	}
}

func TestPickingAToolWithoutPropertiesComposesItRightAway(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	a.openToolForm(tab, request.MCPTool{Name: "ping"})
	if a.dialog != nil {
		t.Fatal("a tool with no arguments needs no form")
	}
	if tab.msg.Text != "call ping {}" {
		t.Fatalf("composer = %q", tab.msg.Text)
	}
}

func TestDialogsWithAndWithoutAFormDrawWithoutPanicking(t *testing.T) {
	a := testApp(t)
	a.prompt("Name", "x", "OK", func(string) {})
	a.step(core.Input{})
	a.closeDialog()
	a.openToolForm(a.cur(), searchTool())
	a.step(core.Input{})
	if a.dialog == nil {
		t.Fatal("the form closed by itself")
	}
}
