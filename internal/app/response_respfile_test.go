package app

import (
	"encoding/json"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestHistoryKeepsEarlierResponsesNewestFirstAndSwapsOnPick(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	send := func(status int) {
		answer(a, tab, request.Response{Status: status, StatusText: "S", Body: []byte("{}")})
	}
	send(200)
	if len(tab.Earlier) != 0 {
		t.Fatal("the first response has no predecessor")
	}
	send(201)
	send(202)
	if len(tab.Earlier) != 2 || tab.Earlier[0].Resp.Status != 201 || tab.Earlier[1].Resp.Status != 200 {
		t.Fatalf("history %+v", tab.Earlier)
	}
	for i := 0; i < tabs.PastLimit+5; i++ {
		send(300 + i)
	}
	if len(tab.Earlier) != tabs.PastLimit {
		t.Fatalf("history grew to %d", len(tab.Earlier))
	}
	number, _ := a.sends.Begin(tab.Req, func() {})
	n := len(tab.Earlier)
	a.apply(sends.Result{Owner: tab.Req, Number: number, Response: request.Response{Err: "boom"}})
	a.apply(sends.Result{Owner: tab.Req, Number: number, Response: request.Response{Status: 200}}) // stale: ignored
	answer(a, tab, request.Response{Status: 204})
	if len(tab.Earlier) != n {
		t.Fatalf("a failed response must not enter the history: %d -> %d", n, len(tab.Earlier))
	}
}

func TestAssertionsRunWhenAResponseArrivesAndAfterEdits(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Tests = "status == 200\njson ok == true\nstatus == 404"
	answer(a, tab, request.Response{Status: 200, StatusText: "OK", Body: []byte(`{"ok":true}`)})
	if results := a.assertions.Results(tab.Tab); len(results) != 3 || !results[0].OK || !results[1].OK || results[2].OK {
		t.Fatalf("%+v", results)
	}
	if !strings.Contains(a.notices.Status(), "Tests: 2/3 passed") {
		t.Fatalf("status %q", a.notices.Status())
	}
	if got := responsePanes[len(responsePanes)-1].badge(a, tab); got != 1 {
		t.Fatalf("the response Tests tab counts failures: %d", got)
	}
	var reqTests pane
	for _, p := range requestPanes {
		if p.kind == paneAssertions {
			reqTests = p
		}
	}
	if got := reqTests.badge(a, tab); got != 3 {
		t.Fatalf("the request Tests tab counts assertions: %d", got)
	}
	if d := view.Dump(a.responsePaneView(responsePanes[3], tab, nil)); !strings.Contains(d, "2/3 passed") || !strings.Contains(d, "status == 404") {
		t.Fatalf("results pane:\n%s", d)
	}
}

func TestPickFileUsesTheDesktopDialogWhenThereIsOne(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "zenity"), []byte("#!/bin/sh\necho /chosen/by/dialog.bin\n"), 0o755)
	t.Setenv("PATH", dir)
	a := testApp(t)
	a.files = filedialog.System{}
	var got string
	a.pickFile("Pick", filedialog.OpenFile, "", func(name string) { got = name })
	if a.browser.Shown() {
		t.Fatal("with a dialog program there is no in-app file browser")
	}
	until(t, a, "the chosen path", func() bool { return got != "" })
	if got != "/chosen/by/dialog.bin" {
		t.Fatalf("got %q", got)
	}

	t.Setenv("PATH", t.TempDir())
	got = ""
	a.pickFile("Pick", filedialog.OpenFile, "start.txt", func(name string) { got = name })
	if !a.browser.Shown() || !strings.HasSuffix(a.browser.Path(), "start.txt") {
		t.Fatal("without one the in-app file browser asks, with the name")
	}
}

func TestFileBodyTypeKeepsItsPathSeparateFromTheTextBody(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	*a.body.Area(tab.Tab) = widget.NewTextArea("typed text")
	tab.Req.Body = "typed text"
	a.body.ChooseType(tab.Tab, request.BodyFile)
	if tab.Req.BodyType != request.BodyFile || tab.Req.Body != "" {
		t.Fatalf("type %q body %q", tab.Req.BodyType, tab.Req.Body)
	}
	*a.body.File(tab.Tab) = widget.NewTextField("/tmp/x.bin")
	tab.Req.Body = a.body.File(tab.Tab).Text
	a.body.ChooseType(tab.Tab, request.BodyText)
	if tab.Req.Body != "typed text" {
		t.Fatalf("the text body must come back: %q", tab.Req.Body)
	}
	a.body.ChooseType(tab.Tab, request.BodyFile) // the file editor is shown for a File body only
	if d := view.Dump(a.body.View(a.ctx, tab.Tab, &widget.ScrollState{})); !strings.Contains(d, "Choose...") {
		t.Fatalf("file editor:\n%s", d)
	}
}

func TestCapturedValuesBecomeVariablesInTheActiveScope(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Tests = "set token = json access_token"
	send := func(body string) {
		answer(a, tab, request.Response{Status: 200, Body: []byte(body)})
	}
	a.col().Vars = nil
	send(`{"access_token":"t1"}`)
	if got := request.Expand("{{token}}", a.varsFor(tab.Req)); got != "t1" {
		t.Fatalf("collection variable: %q", got)
	}
	send(`{"access_token":"t2"}`)
	if len(a.col().Vars) != 1 || a.col().Vars[0].Value != "t2" {
		t.Fatalf("an existing row is updated, not duplicated: %+v", a.col().Vars)
	}
	a.addEnv("Dev")
	send(`{"access_token":"t3"}`)
	if a.col().Vars[0].Value != "t2" || len(a.col().Envs[0].Vars) != 1 || a.col().Envs[0].Vars[0].Value != "t3" {
		t.Fatalf("with an environment active the value goes there: %+v %+v", a.col().Vars, a.col().Envs[0].Vars)
	}
}

func TestGraphQLBodyTypeSendsAQueryAndCanLoadTheSchema(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		var p struct{ Query string }
		b, _ := io.ReadAll(call.Body)
		json.Unmarshal(b, &p)
		queries = append(queries, p.Query)
		writer.Write([]byte(`{"data":{"__schema":{"types":[]}}}`))
	}))
	defer srv.Close()
	a := openBodyPane(t)
	tab := a.cur()
	tab.Req.URL, tab.Req.Method = srv.URL, "GET"
	*a.body.Area(tab.Tab) = widget.NewTextArea("{ repos { name } }")
	a.body.ChooseType(tab.Tab, request.BodyGraphQL)
	a.Step(core.Input{})
	if tab.Req.Method != "POST" || tab.Req.Body != "{ repos { name } }" {
		t.Fatalf("method %q body %q", tab.Req.Method, tab.Req.Body)
	}
	if d := view.Dump(a.graphql.View(a.ctx, tab.Tab)); !strings.Contains(d, "Load schema") || !strings.Contains(d, "VARIABLES") {
		t.Fatalf("editor:\n%s", d)
	}
	a.send(tab)
	waitForResponse(t, a, tab)
	tab.Resp = nil
	click(a, gqlLoadSchemaX, gqlButtonsY)
	waitForResponse(t, a, tab)
	if len(queries) != 2 || queries[0] != "{ repos { name } }" || !strings.Contains(queries[1], "__schema") {
		t.Fatalf("queries %q", queries)
	}
	if tab.Req.Body != "{ repos { name } }" {
		t.Fatal("loading the schema must not touch the tab's own query")
	}
}

func TestADragRestingOverAClosedFolderOpensIt(t *testing.T) {
	a := testApp(t)
	closed := a.col().Folders[0]
	a.side.Tree().Expanded()[closed] = false
	req := a.col().Folders[1].Requests[0]
	holdDrag(a, request.Of(req), request.Of(closed))
	for i := 0; i < 29; i++ { // one tick short of opening the folder
		a.side.Tick()
	}
	if a.side.Tree().Expanded()[closed] {
		t.Fatal("opened too early")
	}
	a.side.Tick()
	if !a.side.Tree().Expanded()[closed] {
		t.Fatal("the folder did not open")
	}
	other := a.col().Folders[2]
	a.side.Tree().Expanded()[other] = false
	a.side.Tree().Over(request.Of(other))
	a.side.Tick()
	a.side.Tree().Expanded()[closed] = false
	a.side.Tree().Over(request.Of(closed)) // moved away before the time was up
	a.side.Tick()
	if drag := a.side.Tree().Drag(); drag.Hover != closed || drag.Ticks != 1 {
		t.Fatalf("the rest time must start over on a new folder: %v %d", drag.Hover, drag.Ticks)
	}
	holdDrag(a, request.Node{}, request.Of(closed))
	a.side.Tick()
	if a.side.Tree().Drag().Hover != nil {
		t.Fatal("without a drag nothing hovers")
	}
}

func TestFormFieldsCanBeEditedAsText(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.BodyType = request.BodyForm
	tab.Req.Form = []request.KV{{Name: "user", Value: "ann", Enabled: true}}
	for i, p := range requestPanes {
		if p.kind == paneBody {
			tab.paneTab = i
		}
	}
	a.Step(core.Input{})
	shown, ok := kvtableHost{a}.Visible()
	p := tablePanes[shown.Kind]
	if !ok || p.label != formPane.label || p.captionOf(a) != "Fields" {
		t.Fatalf("the form is a table with a caption: %v %q", ok, p.captionOf(a))
	}
	table := shown
	a.kv.ToggleBulk(table)
	if a.kv.BulkArea() == nil || a.kv.BulkArea().Text != "user: ann" {
		t.Fatalf("bulk text %v", a.kv.BulkArea())
	}
	*a.kv.BulkArea() = widget.NewTextArea("user: bob\nrole: admin")
	a.kv.ToggleBulk(table)
	if len(tab.Req.Form) != 2 || tab.Req.Form[0].Value != "bob" || tab.Req.Form[1].Name != "role" {
		t.Fatalf("form %+v", tab.Req.Form)
	}
}

// A failed send is still checked (a "status == 0" line can pass), but its
// status line stays "Request failed": no tests summary is added to it.
func TestAFailedSendIsCheckedButNotSummedUpInTheStatus(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Tests = "status == 200"
	answer(a, tab, request.Response{Err: "connection refused"})
	if results := a.assertions.Results(tab.Tab); len(results) != 1 || results[0].OK {
		t.Fatalf("results %+v, want one failed", results)
	}
	if got := a.notices.Status(); got != "Request failed" {
		t.Fatalf("status %q, want it unchanged by the tests", got)
	}
}
