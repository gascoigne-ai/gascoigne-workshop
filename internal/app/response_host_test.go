package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The host opens the in-app file browser, there being no desktop dialog in
// a test, with the name the component suggests.
func TestTheResponseHostAsksForTheFileToSaveInTheFileBrowser(t *testing.T) {
	a := testApp(t)
	var answer string
	responseHost{a}.PickSave("Save response body", "response.txt", func(path string) { answer = path })
	if dump := layoutDump(a); !strings.HasSuffix(a.over.Browser.Path(), "response.txt") || !strings.Contains(dump, `"Save response body"`) {
		t.Fatalf("the file browser did not open with the name %q:\n%s", a.over.Browser.Path(), dump)
	}
	a.folders = newFakeFolders()
	typePath(a, "/out/x")
	pressEnter(a)
	if answer != "/out/x" {
		t.Fatalf("answer %q, want the chosen path handed on", answer)
	}
}

// The Tests pane of the response is the assertions component's, drawn with
// the scroll the response area gives it.
func TestTheResponseHostDrawsTheResultsWithTheScrollItIsGiven(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Tests = "status == 200"
	tab.Resp = &request.Response{Status: 200, StatusText: "OK"}
	a.assertions.Run(tab.Tab)
	var scroll widget.ScrollState
	results := responseHost{a}.Results(tab.Tab, &scroll)
	if results == nil {
		t.Fatal("the host drew no results")
	}
	found := findWidget(results, "*widget.scrollB")
	if found == nil {
		t.Fatal("the results have no scroll")
	}
	if got, want := fmt.Sprintf("%+v", found.Behavior()), fmt.Sprintf("%p", &scroll); !strings.Contains(got, want) {
		t.Fatalf("the scroll lacks the given state %s: %s", want, got)
	}
}

// The column widths of the response headers are the key-value tables' own
// record, so they persist with the others.
func TestTheResponseHostGivesTheHeadersWidthsOfTheKeyValueTables(t *testing.T) {
	a := testApp(t)
	if got := (responseHost{a}).Widths(); got == nil || got != a.kv.Widths(kvlogic.ResponseHeaders) {
		t.Fatalf("widths %p, want the record of the response headers %p", got, a.kv.Widths(kvlogic.ResponseHeaders))
	}
}
