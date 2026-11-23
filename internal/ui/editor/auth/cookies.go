package auth

import (
	"net/url"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// ShowCookies lists the cookies the jar holds for the request's URL and
// offers to clear all of them.
func (c *Component) ShowCookies(tab *tabs.Tab) {
	u, err := url.Parse(tab.Req.Expanded(c.deps.Vars.VarsFor(tab.Req)).URL)
	if err != nil || u.Host == "" {
		c.deps.Window.Toast("The URL has no host to look up cookies for", severity.Error)
		return
	}
	var lines []string
	for _, cookie := range c.deps.Cookies.Cookies(u) {
		lines = append(lines, cookie.Name+"="+cookie.Value)
	}
	body := "No cookies for " + u.Host + "."
	if len(lines) > 0 {
		body = "Sent to " + u.Host + ":\n" + strings.Join(lines, "\n")
	}
	c.deps.Window.Confirm("Cookies", body, "Clear all cookies", c.clearCookies)
}

// clearCookies forgets every cookie the servers set.
func (c *Component) clearCookies() {
	if err := c.deps.Cookies.Clear(); err != nil {
		c.deps.Window.Toast(err.Error(), severity.Error)
		return
	}
	c.deps.Window.Toast("Cookies cleared", severity.Info)
}
