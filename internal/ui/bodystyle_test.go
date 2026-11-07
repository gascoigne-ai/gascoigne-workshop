package ui

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestBodyEditorColorsVariablesByWhetherTheyResolve(t *testing.T) {
	a, _ := newTestApp(t)
	a.col.Vars = []model.KV{{Name: "known", Value: "1", Enabled: true}}
	style := a.withVariables(jsonStyler, a.cur().req)
	spans := style(`{"a": "{{known}}", "b": "{{missing}}"}`)
	var known, missing w.Token
	var plain int
	for _, s := range spans {
		switch s.Text {
		case "{{known}}":
			known = s.Style
		case "{{missing}}":
			missing = s.Style
		default:
			plain++
		}
	}
	if known != w.OK || missing != w.Error {
		t.Fatalf("known %q missing %q in %+v", known, missing, spans)
	}
	if plain == 0 {
		t.Fatal("the text around the variables must stay")
	}
	joined := ""
	for _, s := range spans {
		joined += s.Text
	}
	if joined != `{"a": "{{known}}", "b": "{{missing}}"}` {
		t.Fatalf("styling must not change the text: %q", joined)
	}
	if got := a.withVariables(plainStyler, a.cur().req)("no variables here"); len(got) != 1 {
		t.Fatalf("a line without variables is one span: %+v", got)
	}
}

func TestTableCellsColorVariablesLikeTheURLField(t *testing.T) {
	a, _ := newTestApp(t)
	a.col.Vars = []model.KV{{Name: "known", Value: "1", Enabled: true}}
	style := a.cellStyler(a.cur().req)
	if style("plain text") != nil {
		t.Fatal("a cell without a variable is drawn as it is")
	}
	spans := style("Bearer {{known}} {{missing}}")
	if len(spans) != 4 || spans[0].Style != "" || spans[1].Style != w.OK || spans[2].Style != "" || spans[3].Style != w.Error {
		t.Fatalf("%+v", spans)
	}
}
