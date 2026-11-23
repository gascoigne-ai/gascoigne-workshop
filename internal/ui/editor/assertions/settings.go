package assertions

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	// The view DSL is dot-imported so views read like markup.
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// timeoutText is a timeout as shown in its field; 0 (the default) is blank.
func timeoutText(sec int) string {
	if sec <= 0 {
		return ""
	}
	return strconv.Itoa(sec)
}

var settingsPaneTemplate = markup.Must("settingspane.vml", settingspaneSource)

// SettingsView is the request's Settings pane of tab: the per-request
// options: timeout, redirects and certificate checks.
func (c *Component) SettingsView(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return settingsPaneTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"scroll":            scroll,
		"timeout":           &c.state(tab).timeout,
		"timeoutHint":       fmt.Sprintf("blank = default (%d s)", c.deps.Defaults.DefaultTimeout()/time.Second),
		"followRedirects":   !tab.Req.NoRedirects,
		"verifyCertificate": !tab.Req.InsecureTLS,
		"timeoutEdited":     func() { c.timeoutEdited(tab) },
		"toggleRedirects":   func() { c.edit(tab, requestedit.NoRedirectsToggled{}) },
		"toggleVerify":      func() { c.edit(tab, requestedit.InsecureTLSToggled{}) },
	}))
}

// timeoutEdited applies the typed timeout; text that is not a whole
// number of seconds is undone in the same field, which keeps its address
// because the template holds the pointer across the frame.
func (c *Component) timeoutEdited(tab *tabs.Tab) {
	field := &c.state(tab).timeout
	text := strings.TrimSpace(field.Text)
	if text == "" {
		c.edit(tab, requestedit.TimeoutSet{Seconds: 0})
		return
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 0 {
		*field = widget.NewTextField(timeoutText(tab.Req.TimeoutSec))
		field.Focused = true
		return
	}
	c.edit(tab, requestedit.TimeoutSet{Seconds: n})
}

// Templates, parsed once above.
const (
	settingspaneSource = `<scroll state="scroll" pad="{{gutter}}" gap="{{looseGap}}">
  <form-row label="Timeout (seconds)" label-width="160">
    <field state="timeout" w="{{timeoutW}}" bg="sunken" on-change="timeoutEdited" text-menu="timeout"/>
    <text small muted>{{timeoutHint}}</text>
  </form-row>
  <check-row on="{{followRedirects}}" on-click="toggleRedirects">
    <text>Follow redirects</text>
  </check-row>
  <check-row on="{{verifyCertificate}}" on-click="toggleVerify">
    <text>Verify the server certificate</text>
    <text small muted>off accepts self-signed certificates</text>
  </check-row>
</scroll>
`
)
