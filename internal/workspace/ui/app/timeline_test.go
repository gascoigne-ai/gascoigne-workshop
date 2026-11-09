package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

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
	tab.url = view.NewTextField("https://api.test/users/:id")
	a.urlEdited(tab)
	var row *model.KV
	for i := range tab.Req.Params {
		if tab.Req.Params[i].Name == ":id" {
			row = &tab.Req.Params[i]
		}
	}
	if row == nil {
		t.Fatalf("no :id row in %+v", tab.Req.Params)
	}
	row.Value = "42"
	if got := tab.Req.Expanded(a.varsFor(tab.Req)).URL; got != "https://api.test/users/42" {
		t.Fatalf("sent URL %q", got)
	}
}

func TestCookiesDialogListsWhatTheJarHoldsForTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.URL = "https://api.test/x"
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
	tab.Req.URL = "not a url"
	a.closeDialog()
	a.showCookies(tab)
	if a.dialog != nil {
		t.Fatal("no dialog for a URL without a host")
	}
}
