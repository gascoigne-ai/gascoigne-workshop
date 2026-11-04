package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestTimelineListsOnlyThePhasesThatHappened(t *testing.T) {
	r := &model.Response{Duration: 1200 * time.Millisecond, Size: 2048,
		Timing: model.Timing{Connect: 30 * time.Millisecond, FirstByte: 900 * time.Millisecond, Transfer: 100 * time.Millisecond}}
	got := timelineText(r)
	for _, want := range []string{"Total      1.2s", "Connect    30ms", "First byte 900ms", "Transfer   100ms", "Size       2.00KB"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "DNS") || strings.Contains(got, "TLS") || strings.Contains(got, "cut") {
		t.Errorf("phases that did not happen must not show:\n%s", got)
	}
	r.Truncated = true
	if !strings.Contains(timelineText(r), "is cut here") {
		t.Error("a cut body needs a note")
	}
}

func TestRequestsShareACookieJarThatClearCookiesEmpties(t *testing.T) {
	a, _ := newTestApp(t)
	first := a.jar
	if first == nil {
		t.Fatal("no cookie jar")
	}
	rest, ok := a.reg["rest"].(proto.REST)
	if !ok || rest.Jar != first {
		t.Fatalf("the REST protocol must use the app's jar: %+v", a.reg["rest"])
	}
	a.clearCookies()
	rest, _ = a.reg["rest"].(proto.REST)
	if a.jar == first || rest.Jar != a.jar {
		t.Fatal("clearing cookies must install a new jar everywhere")
	}
}

func TestTypingAColonSegmentAddsAPathParameterRowThatFillsTheSentURL(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.url = vml.NewTextField("https://api.test/users/:id")
	a.urlEdited(tab)
	var row *model.KV
	for i := range tab.req.Params {
		if tab.req.Params[i].Name == ":id" {
			row = &tab.req.Params[i]
		}
	}
	if row == nil {
		t.Fatalf("no :id row in %+v", tab.req.Params)
	}
	row.Value = "42"
	if got := tab.req.Expanded(a.varsFor(tab.req)).URL; got != "https://api.test/users/42" {
		t.Fatalf("sent URL %q", got)
	}
}

func TestCookiesDialogListsWhatTheJarHoldsForTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.URL = "https://api.test/x"
	a.showCookies(tab)
	if a.dialog == nil || !strings.Contains(a.dialog.body, "No cookies for api.test") {
		t.Fatalf("dialog %+v", a.dialog)
	}
	a.closeDialog()
	u, _ := url.Parse("https://api.test/x")
	a.jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "abc", Path: "/"}})
	a.showCookies(tab)
	if !strings.Contains(a.dialog.body, "session=abc") {
		t.Fatalf("body %q", a.dialog.body)
	}
	first := a.jar
	a.dialog.buttons[len(a.dialog.buttons)-1].run() // Clear all cookies
	if a.jar == first || len(a.jar.Cookies(u)) != 0 {
		t.Fatal("the jar must be replaced and empty")
	}
	tab.req.URL = "not a url"
	a.closeDialog()
	a.showCookies(tab)
	if a.dialog != nil {
		t.Fatal("no dialog for a URL without a host")
	}
}
