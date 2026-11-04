package ui

import (
	"bytes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestHistoryKeepsEarlierResponsesNewestFirstAndSwapsOnPick(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	send := func(status int) {
		tab.sending, tab.seq = true, tab.seq+1
		a.apply(sendResult{tab: tab, seq: tab.seq, resp: model.Response{Status: status, StatusText: "S", Body: []byte("{}")}})
	}
	send(200)
	if len(tab.respHist) != 0 {
		t.Fatal("the first response has no predecessor")
	}
	send(201)
	send(202)
	if len(tab.respHist) != 2 || tab.respHist[0].resp.Status != 201 || tab.respHist[1].resp.Status != 200 {
		t.Fatalf("history %+v", tab.respHist)
	}
	items := a.historyMenu(tab)
	if len(items) != 2 || !strings.HasPrefix(items[0].label, "201") {
		t.Fatalf("menu %+v", items)
	}
	items[1].run() // pick the 200
	if tab.resp.Status != 200 || tab.respHist[1].resp.Status != 202 {
		t.Fatalf("shown %d, history %+v", tab.resp.Status, tab.respHist)
	}
	for i := 0; i < respHistoryLimit+5; i++ {
		send(300 + i)
	}
	if len(tab.respHist) != respHistoryLimit {
		t.Fatalf("history grew to %d", len(tab.respHist))
	}
	tab.sending, tab.seq = true, tab.seq+1
	n := len(tab.respHist)
	a.apply(sendResult{tab: tab, seq: tab.seq, resp: model.Response{Err: "boom"}})
	a.apply(sendResult{tab: tab, seq: tab.seq + 1, resp: model.Response{Status: 200}}) // stale: ignored
	tab.sending, tab.seq = true, tab.seq+1
	a.apply(sendResult{tab: tab, seq: tab.seq, resp: model.Response{Status: 204}})
	if len(tab.respHist) != n {
		t.Fatalf("a failed response must not enter the history: %d -> %d", n, len(tab.respHist))
	}
}

func TestSuggestedFileNameFollowsTheContentType(t *testing.T) {
	for ct, want := range map[string]string{
		"application/json; charset=utf-8": "response.json",
		"text/html":                       "response.html",
		"application/vnd.api+json":        "response.json",
		"image/png":                       "response.png",
		"":                                "response.txt",
	} {
		r := &model.Response{}
		if ct != "" {
			r.Headers = []model.KV{{Name: "content-type", Value: ct, Enabled: true}}
		}
		if got := suggestedFileName(r); got != want {
			t.Errorf("%q: %q, want %q", ct, got, want)
		}
	}
}

func TestSaveBodyWritesTheFileAndReportsFailures(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	a.saveBody(tab) // no response yet
	if a.dialog != nil {
		t.Fatal("no dialog without a response")
	}
	tab.resp = &model.Response{Status: 200, Body: []byte(`{"a":1}`)}
	a.saveBody(tab)
	if a.dialog == nil || a.dialog.input == nil || a.dialog.input.Text != "response.txt" {
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
		found = found || (ts.isErr && strings.Contains(ts.text, "Saving failed"))
	}
	if !found {
		t.Fatal("a failed save must be reported")
	}
}

func TestAssertionsRunWhenAResponseArrivesAndAfterEdits(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.Tests = "status == 200\njson ok == true\nstatus == 404"
	tab.sending, tab.seq = true, tab.seq+1
	a.apply(sendResult{tab: tab, seq: tab.seq, resp: model.Response{Status: 200, StatusText: "OK", Body: []byte(`{"ok":true}`)}})
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
	if d := vml.Dump(a.resultsPane(responsePanes[3], tab, nil)); !strings.Contains(d, "2/3 passed") || !strings.Contains(d, "status == 404") {
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
	tab.sending, tab.seq = true, tab.seq+1
	a.apply(sendResult{tab: tab, seq: tab.seq, resp: model.Response{Status: 200, StatusText: "OK", Body: buf.Bytes(),
		Headers: []model.KV{{Name: "Content-Type", Value: "text/plain", Enabled: true}}}}) // mislabeled on purpose
	if tab.picture == nil || !strings.HasPrefix(tab.pictureInfo, "PNG 40x20") {
		t.Fatalf("picture %v info %q", tab.picture, tab.pictureInfo)
	}
	if msg, _ := responseEmpty(tab); msg != "" {
		t.Fatalf("a picture must not be reported as binary: %q", msg)
	}
	if d := vml.Dump(a.responsePane(responsePanes[0], tab, nil)); !strings.Contains(d, "PNG 40x20") || !strings.Contains(d, "Save") {
		t.Fatalf("pane:\n%s", d)
	}
	tab.sending, tab.seq = true, tab.seq+1
	a.apply(sendResult{tab: tab, seq: tab.seq, resp: model.Response{Status: 200, Body: []byte(`{"a":1}`)}})
	if tab.picture != nil {
		t.Fatal("the next, non-image response must clear the picture")
	}
}
