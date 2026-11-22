package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// clearCookies forgets every cookie the servers set. The jar is replaced,
// so connections made before keep nothing.
func (a *App) clearCookies() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		a.toast(err.Error(), severity.Error)
		return
	}
	a.jar = jar
	a.reg = a.newRegistry()
	a.toast("Cookies cleared", severity.Info)
}

// newRegistry builds the protocols from the cookie jar, the network
// settings (proxy, certificates) and the app's sampling dialog.
func (a *App) newRegistry() request.Registry {
	if a.protocols == nil {
		return request.Registry{}
	}
	s := a.settings
	return a.protocols(request.ProtocolConfig{Jar: a.jar, Proxy: s.Proxy, CACert: s.CACert, ClientCert: s.ClientCert, ClientKey: s.ClientKey, Sampler: a.sampleViaDialog})
}

// showCookies lists the cookies the jar holds for the request's URL and
// offers to clear all of them.
func (a *App) showCookies(tab *reqTab) {
	u, err := url.Parse(tab.Req.Expanded(a.varsFor(tab.Req)).URL)
	if err != nil || u.Host == "" {
		a.toast("The URL has no host to look up cookies for", severity.Error)
		return
	}
	var lines []string
	for _, c := range a.jar.Cookies(u) {
		lines = append(lines, c.Name+"="+c.Value)
	}
	body := "No cookies for " + u.Host + "."
	if len(lines) > 0 {
		body = "Sent to " + u.Host + ":\n" + strings.Join(lines, "\n")
	}
	a.confirm("Cookies", body, "Clear all cookies", a.clearCookies)
}
