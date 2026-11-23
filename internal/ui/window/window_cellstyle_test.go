package window

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestTableCellsColorVariablesLikeTheURLField(t *testing.T) {
	a := testApp(t)
	a.col().Vars = []request.KV{{Name: "known", Value: "1", Enabled: true}}
	style := kvtableHost{a}.CellStyler(a.cur().Req)
	if style("plain text") != nil {
		t.Fatal("a cell without a variable is drawn as it is")
	}
	spans := style("Bearer {{known}} {{missing}}")
	if len(spans) != 4 || spans[0].Style != "" || spans[1].Style != w.OK || spans[2].Style != "" || spans[3].Style != w.Error {
		t.Fatalf("%+v", spans)
	}
}
