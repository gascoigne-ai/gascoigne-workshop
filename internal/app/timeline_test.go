package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

func TestRequestsShareACookieJarThatClearCookiesEmpties(t *testing.T) {
	a := testApp(t)
	first := a.jar
	if first == nil {
		t.Fatal("no cookie jar")
	}
	rest, ok := a.reg["rest"].(protocol.REST)
	if !ok || rest.Jar != first {
		t.Fatalf("the REST protocol must use the app's jar: %+v", a.reg["rest"])
	}
	authHost{a}.Clear()
	rest, _ = a.reg["rest"].(protocol.REST)
	if a.jar == first || rest.Jar != a.jar {
		t.Fatal("clearing cookies must install a new jar everywhere")
	}
}

func TestTypingAColonSegmentAddsAPathParameterRowThatFillsTheSentURL(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	*a.urlbar.Field(tab.Tab) = widget.NewTextField("https://api.test/users/:id")
	a.urlbar.Edited(tab.Tab)
	var row *request.KV
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
	a := testApp(t)
	tab := a.cur()
	tab.Req.URL = "https://api.test/x"
	a.auth.ShowCookies(tab.Tab)
	if a.over.Dialog() == nil || !strings.Contains(a.over.Dialog().Body, "No cookies for api.test") {
		t.Fatalf("dialog %+v", a.over.Dialog())
	}
	a.over.CloseDialog()
	u, _ := url.Parse("https://api.test/x")
	a.jar.SetCookies(u, []*http.Cookie{{Name: "session", Value: "abc", Path: "/"}})
	a.auth.ShowCookies(tab.Tab)
	if !strings.Contains(a.over.Dialog().Body, "session=abc") {
		t.Fatalf("body %q", a.over.Dialog().Body)
	}
	first := a.jar
	a.over.Dialog().Buttons[len(a.over.Dialog().Buttons)-1].Run() // Clear all cookies
	if a.jar == first || len(a.jar.Cookies(u)) != 0 {
		t.Fatal("the jar must be replaced and empty")
	}
	tab.Req.URL = "not a url"
	a.over.CloseDialog()
	a.auth.ShowCookies(tab.Tab)
	if a.over.Dialog() != nil {
		t.Fatal("no dialog for a URL without a host")
	}
}

// Regression: the protocol registry was built at start and when cookies
// were cleared, so a proxy or certificate chosen in the settings did
// nothing until the next restart.
func TestChangingTheProxyInTheSettingsReachesTheProtocols(t *testing.T) {
	a := testApp(t)

	a.changeSettings(func(prefs *settings.Settings) { prefs.Proxy = "http://127.0.0.1:3128" })

	rest, ok := a.reg["rest"].(protocol.REST)
	if !ok || rest.Proxy != "http://127.0.0.1:3128" {
		t.Fatalf("REST protocol did not get the proxy: %+v", a.reg["rest"])
	}
}
