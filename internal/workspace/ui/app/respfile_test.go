package ui

import (
	"bytes"
	"encoding/json"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestHistoryKeepsEarlierResponsesNewestFirstAndSwapsOnPick(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	send := func(status int) {
		tab.Sending, tab.Seq = true, tab.Seq+1
		a.apply(sendResult{tab: tab, seq: tab.Seq, resp: model.Response{Status: status, StatusText: "S", Body: []byte("{}")}})
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
	items := a.historyMenu(tab)
	if len(items) != 2 || !strings.HasPrefix(items[0].Label, "201") {
		t.Fatalf("menu %+v", items)
	}
	items[1].Run() // pick the 200
	if tab.Resp.Status != 200 || tab.Earlier[1].Resp.Status != 202 {
		t.Fatalf("shown %d, history %+v", tab.Resp.Status, tab.Earlier)
	}
	for i := 0; i < tabs.PastLimit+5; i++ {
		send(300 + i)
	}
	if len(tab.Earlier) != tabs.PastLimit {
		t.Fatalf("history grew to %d", len(tab.Earlier))
	}
	tab.Sending, tab.Seq = true, tab.Seq+1
	n := len(tab.Earlier)
	a.apply(sendResult{tab: tab, seq: tab.Seq, resp: model.Response{Err: "boom"}})
	a.apply(sendResult{tab: tab, seq: tab.Seq + 1, resp: model.Response{Status: 200}}) // stale: ignored
	tab.Sending, tab.Seq = true, tab.Seq+1
	a.apply(sendResult{tab: tab, seq: tab.Seq, resp: model.Response{Status: 204}})
	if len(tab.Earlier) != n {
		t.Fatalf("a failed response must not enter the history: %d -> %d", n, len(tab.Earlier))
	}
}

func TestSaveBodyWritesTheFileAndReportsFailures(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no dialog program: the path is asked as text
	a, _ := newTestApp(t)
	tab := a.cur()
	a.saveBody(tab) // no response yet
	if a.dialog != nil {
		t.Fatal("no dialog without a response")
	}
	tab.Resp = &model.Response{Status: 200, Body: []byte(`{"a":1}`)}
	a.saveBody(tab)
	if a.dialog == nil || a.dialog.Input == nil || a.dialog.Input.Text != "response.txt" {
		t.Fatalf("the prompt did not open with a name: %+v", a.dialog)
	}
	a.closeDialog()
	path := filepath.Join(t.TempDir(), "out.json")
	a.writeBody(tab, path)
	if b, err := os.ReadFile(path); err != nil || string(b) != `{"a":1}` {
		t.Fatalf("%q %v", b, err)
	}
	a.writeBody(tab, filepath.Join(t.TempDir(), "no", "such", "dir", "x"))
	found := false
	for _, ts := range a.toasts {
		found = found || (ts.IsErr && strings.Contains(ts.Text, "Saving failed"))
	}
	if !found {
		t.Fatal("a failed save must be reported")
	}
}

func TestAssertionsRunWhenAResponseArrivesAndAfterEdits(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Tests = "status == 200\njson ok == true\nstatus == 404"
	tab.Sending, tab.Seq = true, tab.Seq+1
	a.apply(sendResult{tab: tab, seq: tab.Seq, resp: model.Response{Status: 200, StatusText: "OK", Body: []byte(`{"ok":true}`)}})
	if len(tab.testResults) != 3 || !tab.testResults[0].OK || !tab.testResults[1].OK || tab.testResults[2].OK {
		t.Fatalf("%+v", tab.testResults)
	}
	if !strings.Contains(a.status, "Tests: 2/3 passed") {
		t.Fatalf("status %q", a.status)
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
	if d := view.Dump(a.resultsPane(responsePanes[3], tab, nil)); !strings.Contains(d, "2/3 passed") || !strings.Contains(d, "status == 404") {
		t.Fatalf("results pane:\n%s", d)
	}
}

func TestAnImageResponseIsShownAsAPicture(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Sending, tab.Seq = true, tab.Seq+1
	a.apply(sendResult{tab: tab, seq: tab.Seq, resp: model.Response{Status: 200, StatusText: "OK", Body: buf.Bytes(),
		Headers: []model.KV{{Name: "Content-Type", Value: "text/plain", Enabled: true}}}}) // mislabeled on purpose
	if tab.picture == nil || !strings.HasPrefix(tab.pictureInfo, "PNG 40x20") {
		t.Fatalf("picture %v info %q", tab.picture, tab.pictureInfo)
	}
	if msg, _ := responseEmpty(tab); msg != "" {
		t.Fatalf("a picture must not be reported as binary: %q", msg)
	}
	if d := view.Dump(a.responsePane(responsePanes[0], tab, nil)); !strings.Contains(d, "PNG 40x20") || !strings.Contains(d, "Save") {
		t.Fatalf("pane:\n%s", d)
	}
	tab.Sending, tab.Seq = true, tab.Seq+1
	a.apply(sendResult{tab: tab, seq: tab.Seq, resp: model.Response{Status: 200, Body: []byte(`{"a":1}`)}})
	if tab.picture != nil {
		t.Fatal("the next, non-image response must clear the picture")
	}
}

func TestPickFileUsesTheDesktopDialogWhenThereIsOne(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "zenity"), []byte("#!/bin/sh\necho /chosen/by/dialog.bin\n"), 0o755)
	t.Setenv("PATH", dir)
	a, _ := newTestApp(t)
	a.files = filedialog.System{}
	var got string
	a.pickFile("Pick", false, "", func(p string) { got = p })
	if a.dialog != nil {
		t.Fatal("with a dialog program there is no text prompt")
	}
	until(t, a, "the chosen path", func() bool { return got != "" })
	if got != "/chosen/by/dialog.bin" {
		t.Fatalf("got %q", got)
	}

	t.Setenv("PATH", t.TempDir())
	got = ""
	a.pickFile("Pick", false, "start.txt", func(p string) { got = p })
	if a.dialog == nil || a.dialog.Input == nil || a.dialog.Input.Text != "start.txt" {
		t.Fatal("without one the path is asked as text")
	}
}

func TestFileBodyTypeKeepsItsPathSeparateFromTheTextBody(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.body = widget.NewTextArea("typed text")
	tab.Req.Body = "typed text"
	a.setBodyType(tab, model.BodyFile)
	if tab.Req.BodyType != model.BodyFile || tab.Req.Body != "" {
		t.Fatalf("type %q body %q", tab.Req.BodyType, tab.Req.Body)
	}
	tab.file = widget.NewTextField("/tmp/x.bin")
	tab.Req.Body = tab.file.Text
	a.setBodyType(tab, model.BodyText)
	if tab.Req.Body != "typed text" {
		t.Fatalf("the text body must come back: %q", tab.Req.Body)
	}
	if d := view.Dump(a.fileBody(tab)); !strings.Contains(d, "Choose...") {
		t.Fatalf("file editor:\n%s", d)
	}
}

func TestCapturedValuesBecomeVariablesInTheActiveScope(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Tests = "set token = json access_token"
	send := func(body string) {
		tab.Sending, tab.Seq = true, tab.Seq+1
		a.apply(sendResult{tab: tab, seq: tab.Seq, resp: model.Response{Status: 200, Body: []byte(body)}})
	}
	a.col.Vars = nil
	send(`{"access_token":"t1"}`)
	if got := model.Expand("{{token}}", a.varsFor(tab.Req)); got != "t1" {
		t.Fatalf("collection variable: %q", got)
	}
	send(`{"access_token":"t2"}`)
	if len(a.col.Vars) != 1 || a.col.Vars[0].Value != "t2" {
		t.Fatalf("an existing row is updated, not duplicated: %+v", a.col.Vars)
	}
	a.addEnv("Dev")
	send(`{"access_token":"t3"}`)
	if a.col.Vars[0].Value != "t2" || len(a.col.Envs[0].Vars) != 1 || a.col.Envs[0].Vars[0].Value != "t3" {
		t.Fatalf("with an environment active the value goes there: %+v %+v", a.col.Vars, a.col.Envs[0].Vars)
	}
}

func TestGraphQLBodyTypeSendsAQueryAndCanLoadTheSchema(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p struct{ Query string }
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &p)
		queries = append(queries, p.Query)
		w.Write([]byte(`{"data":{"__schema":{"types":[]}}}`))
	}))
	defer srv.Close()
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.URL, tab.Req.Method = srv.URL, "GET"
	tab.body = widget.NewTextArea("{ repos { name } }")
	a.setBodyType(tab, model.BodyGraphQL)
	if tab.Req.Method != "POST" || tab.Req.Body != "{ repos { name } }" {
		t.Fatalf("method %q body %q", tab.Req.Method, tab.Req.Body)
	}
	if d := view.Dump(a.graphQLBody(tab)); !strings.Contains(d, "Load schema") || !strings.Contains(d, "VARIABLES") {
		t.Fatalf("editor:\n%s", d)
	}
	a.send(tab)
	waitForResponse(t, a, tab)
	tab.Resp = nil
	a.loadSchema(tab)
	waitForResponse(t, a, tab)
	if len(queries) != 2 || queries[0] != "{ repos { name } }" || !strings.Contains(queries[1], "__schema") {
		t.Fatalf("queries %q", queries)
	}
	if tab.Req.Body != "{ repos { name } }" {
		t.Fatal("loading the schema must not touch the tab's own query")
	}
}

func TestADragRestingOverAClosedFolderOpensIt(t *testing.T) {
	a, _ := newTestApp(t)
	closed := a.col.Folders[0]
	a.expanded[closed] = false
	req := a.col.Folders[1].Requests[0]
	a.dragNode, a.dropTarget = req, closed
	for i := 0; i < dragExpandTicks-1; i++ {
		a.dragHover()
	}
	if a.expanded[closed] {
		t.Fatal("opened too early")
	}
	a.dragHover()
	if !a.expanded[closed] {
		t.Fatal("the folder did not open")
	}
	other := a.col.Folders[2]
	a.expanded[other] = false
	a.dropTarget = other
	a.dragHover()
	a.expanded[closed] = false
	a.dropTarget = closed // moved away before the time was up
	a.dragHover()
	if a.hoverFolder != closed || a.hoverTicks != 1 {
		t.Fatalf("the rest time must start over on a new folder: %v %d", a.hoverFolder, a.hoverTicks)
	}
	a.dragNode = nil
	a.dragHover()
	if a.hoverFolder != nil {
		t.Fatal("without a drag nothing hovers")
	}
}

func TestFormFieldsCanBeEditedAsText(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.BodyType = model.BodyForm
	tab.Req.Form = []model.KV{{Name: "user", Value: "ann", Enabled: true}}
	for i, p := range requestPanes {
		if p.kind == paneBody {
			tab.paneTab = i
		}
	}
	a.step(core.Input{})
	p, _, ok := a.visibleTable()
	if !ok || p.label != formPane.label || p.captionOf(a) != "Fields" {
		t.Fatalf("the form is a table with a caption: %v %q", ok, p.captionOf(a))
	}
	a.toggleBulk(p, tab)
	if a.bulk == nil || a.bulk.area.Text != "user: ann" {
		t.Fatalf("bulk text %q", a.bulk.area.Text)
	}
	a.bulk.area = widget.NewTextArea("user: bob\nrole: admin")
	a.closeBulk()
	if len(tab.Req.Form) != 2 || tab.Req.Form[0].Value != "bob" || tab.Req.Form[1].Name != "role" {
		t.Fatalf("form %+v", tab.Req.Form)
	}
}

func TestToggleWrapSwitchesTheResponseViewAndSurvivesANewResponse(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	tab := a.cur()
	if tab.view.Wrap {
		t.Fatal("lines are not wrapped by default")
	}
	a.toggleWrap(tab)
	if !tab.view.Wrap {
		t.Fatal("toggling turns wrapping on")
	}
	a.refreshView(tab) // a new response or view mode rebuilds the lines
	if !tab.view.Wrap {
		t.Fatal("wrapping is a choice of the view, not of one response")
	}
	a.toggleWrap(tab)
	if tab.view.Wrap {
		t.Fatal("toggling again turns it off")
	}
}
