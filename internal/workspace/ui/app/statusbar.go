package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var responseStatusTemplate = markup.Must("responsestatus.vml", responsestatusSource)

// responseStatus is "200 OK  1.4s  156.67KB", dropped when the tabs need the room.
func (a *App) responseStatus(tab *reqTab) Node {
	if tab.Resp == nil || tab.Resp.Err != "" {
		return nil
	}
	fg, bg := statusPill(tab.Resp.Status)
	return responseStatusTemplate.MustBuild(markup.Scope{
		"gutter":   w.Gutter,
		"labelPad": w.LabelPad,
		"slimH":    w.SlimH,
		"pillFg":   fg,
		"pillBg":   bg,
		"status":   fmt.Sprintf("%d %s", tab.Resp.Status, tab.Resp.StatusText),
		"timing":   fmt.Sprintf("%s  %s", textfmt.Duration(tab.Resp.Duration), textfmt.Size(tab.Resp.Size)),
	})
}

// statusPill is the text and background colors of a response status:
// success, redirect or failure, as the dashboard's badges.
func statusPill(status int) (fg, bg w.Token) {
	switch {
	case status >= 400:
		return w.Error, w.ErrorSoft
	case status >= 300:
		return w.Warn, w.WarnSoft
	}
	return w.OK, w.OKSoft
}

// ── Status bar ─────────────────────────────────────────────────────────

var statusBarTemplate = markup.Must("statusbar.vml", statusbarSource)

// statusBar is the line under the window: the last status, the Log
// button and the version.
func (a *App) statusBar() Node {
	return statusBarTemplate.MustBuild(markup.Scope{
		"statusH":     statusH,
		"gutter":      w.Gutter,
		"slimH":       w.SlimH,
		"wordBtnW":    w.WordBtnW,
		"status":      a.status,
		"version":     version,
		"openLog":     a.calls.Open,
		"openWelcome": a.openWelcome,
	})
}

// Templates, parsed once above.
const (
	responsestatusSource = `<row gap="{{gutter}}" pad-r="{{gutter}}" optional align-items="center">
  <row pad-x="{{labelPad}}" h="{{slimH}}" bg="{{pillBg}}" control="pill" align-items="center">
    <text small bold color="{{pillFg}}">{{status}}</text>
  </row>
  <text>{{timing}}</text>
</row>
`

	statusbarSource = `<row h="{{statusH}}" pad-x="{{gutter}}" gap="{{gutter}}" bg="frame" control="statusbar" border-t="border">
  <text small muted ellipsis grow="1">{{status}}</text>
  <button w="{{wordBtnW}}" h="{{slimH}}" on-click="openLog" tooltip="Show the log of calls">Log</button>
  <row on-click="openWelcome" tooltip="What is in this version">
    <text small muted>{{version}}</text>
  </row>
</row>
`
)
