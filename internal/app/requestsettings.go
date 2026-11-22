package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strconv"
	"strings"
	"time"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
)

// timeoutText is a timeout as shown in its field; 0 (the default) is blank.
func timeoutText(sec int) string {
	if sec <= 0 {
		return ""
	}
	return strconv.Itoa(sec)
}

var settingsPaneTemplate = markup.Must("settingspane.vml", settingspaneSource)

// settingsPane holds the per-request options: timeout, redirects and
// certificate checks.
func (a *App) settingsPane(_ pane, tab *reqTab, scroll *ScrollState) Node {
	return settingsPaneTemplate.MustBuild(a.scope(markup.Scope{
		"scroll":            scroll,
		"timeout":           &tab.timeout,
		"timeoutHint":       fmt.Sprintf("blank = default (%d s)", a.timeoutFor(request.Request{})/time.Second),
		"followRedirects":   !tab.Req.NoRedirects,
		"verifyCertificate": !tab.Req.InsecureTLS,
		"timeoutEdited":     func() { a.timeoutEdited(tab) },
		"toggleRedirects":   func() { editRequest(tab, editor.NoRedirectsToggled{}) },
		"toggleVerify":      func() { editRequest(tab, editor.InsecureTLSToggled{}) },
	}))
}

// timeoutEdited applies the typed timeout; text that is not a whole
// number of seconds is undone.
func (a *App) timeoutEdited(tab *reqTab) {
	text := strings.TrimSpace(tab.timeout.Text)
	if text == "" {
		editRequest(tab, editor.TimeoutSet{Seconds: 0})
		return
	}
	n, err := strconv.Atoi(text)
	if err != nil || n < 0 {
		tab.timeout = NewTextField(timeoutText(tab.Req.TimeoutSec))
		tab.timeout.Focused = true
		return
	}
	editRequest(tab, editor.TimeoutSet{Seconds: n})
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
