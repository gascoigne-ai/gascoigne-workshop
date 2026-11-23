package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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

func TestDialogsWithAndWithoutAFormDrawWithoutPanicking(t *testing.T) {
	a := testApp(t)
	a.prompt("Name", "x", "OK", func(string) {})
	a.Step(core.Input{})
	a.over.CloseDialog()
	a.editor.MCP.OpenToolForm(a.cur().Tab, searchTool())
	a.Step(core.Input{})
	if a.over.Dialog() == nil {
		t.Fatal("the form closed by itself")
	}
}
