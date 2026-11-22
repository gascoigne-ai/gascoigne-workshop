package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strings"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// methodOrder is cycled by clicking the method chip of the URL bar; the
// protocol chip cycles protocolViews.
var methodOrder = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// next returns the entry after cur in order, wrapping; the first entry
// when cur is not listed.
func next[T comparable](order []T, cur T) T {
	for i, s := range order {
		if s == cur {
			return order[(i+1)%len(order)]
		}
	}
	return order[0]
}

// protocolsFor are the protocols the chip cycles through for a request on
// current: the experimental ones only when settings switch them on, except
// the one the request already uses, so it never loses its protocol.
func (a *App) protocolsFor(current model.Protocol) []model.Protocol {
	var offered []model.Protocol
	for _, view := range protocolViews {
		if view.offered == nil || view.offered(a.settings) || view.protocol == current {
			offered = append(offered, view.protocol)
		}
	}
	return offered
}

func (a *App) cycleProtocol(tab *reqTab) {
	a.disconnect(tab) // a connection belongs to the protocol that opened it
	editRequest(tab, editor.ProtocolChosen{Protocol: next(a.protocolsFor(tab.Req.Protocol), tab.Req.Protocol)})
	tab.paneTab, tab.respTab = 0, 0 // the indexes point into the panes of the protocol
}

func (a *App) cycleMethod(tab *reqTab) {
	editRequest(tab, editor.MethodChosen{Method: next(methodOrder, tab.Req.Method)})
}

// urlEdited keeps the request and its Params table in step with the URL field.
func (a *App) urlEdited(tab *reqTab) { editRequest(tab, editor.URLTyped{Text: tab.url.Text}) }

var urlBarTemplate = markup.Must("urlbar.vml", urlbarSource)

// urlBar is [protocol | method | URL field ........] [Send]. The button
// is one of three: Send, Connect or Disconnect on a socket, and Cancel
// while a request runs.
func (a *App) urlBar(tab *reqTab) Node {
	info, socket := tab.Req.Protocol.Info(), isSocket(tab)
	httpMethod, rpcMethod := info.Method == model.MethodHTTP, info.Method == model.MethodRPC
	sending := !socket && tab.Sending()
	connectionLabel, connectionHint := "Connect", "Open the connection"
	if tab.Active() {
		connectionLabel, connectionHint = "Disconnect", "Close the connection"
	}
	grpcMethod, grpcMethodColor := grpcMethodLabel(tab)
	return urlBarTemplate.MustBuild(a.scope(markup.Scope{
		"looseGap":         w.LooseGap,
		"urlH":             urlH,
		"gutter":           w.Gutter,
		"urlTextGap":       w.Gutter - w.NearGap,
		"urlTip":           a.urlTip(tab),
		"protocol":         info.Label,
		"divided":          httpMethod || rpcMethod,
		"httpMethod":       httpMethod,
		"method":           tab.Req.Method,
		"methodColor":      theme.MethodToken(tab.Req.Method),
		"rpcMethod":        rpcMethod,
		"streamable":       info.Connection == model.OnStream,
		"grpcMethodW":      grpcMethodW,
		"grpcMethod":       grpcMethod,
		"grpcMethodColor":  grpcMethodColor,
		"streamColor":      streamColor(tab),
		"streamBg":         streamBg(tab),
		"url":              &tab.url,
		"segments":         a.varSegments,
		"idle":             !socket && !sending,
		"socket":           socket,
		"sending":          sending,
		"sendTip":          a.hint("Send the request", "send"),
		"connectionLabel":  connectionLabel,
		"connectionTip":    a.hint(connectionHint, "send"),
		"cycleProtocol":    func() { a.cycleProtocol(tab) },
		"cycleMethod":      func() { a.cycleMethod(tab) },
		"pickMethod":       func() { a.pickGRPCMethod(tab) },
		"toggleStream":     func() { a.toggleStream(tab) },
		"edited":           func() { a.urlEdited(tab) },
		"send":             func() { a.send(tab) },
		"toggleConnection": func() { a.toggleConnection(tab) },
		"cancel":           func() { a.cancelSend(tab) },
	}))
}

// streamColor shows the Stream chip lit while the tab streams.
func streamColor(tab *reqTab) w.Token {
	if tab.grpcStream {
		return w.Accent
	}
	return w.Muted
}

// streamBg tints the Stream chip while the tab streams; an empty token
// applies no background (see the markup package doc).
func streamBg(tab *reqTab) w.Token {
	if tab.grpcStream {
		return w.Selected
	}
	return ""
}

// hint is a tooltip with the action's shortcut, when it has one.
func (a *App) hint(text, shortcutID string) string {
	if key := a.shortcutHint(shortcutID); key != "" {
		return text + " (" + key + ")"
	}
	return text
}

// varSegments colors {{variables}} so they stand out from the literal
// URL: defined ones in the success color, unresolved ones as errors.
func (a *App) varSegments(raw string) []Span {
	var segs []Span
	vars := a.varsFor(a.cur().Req)
	for _, seg := range model.SplitVars(raw) {
		style := w.Text
		if seg.IsVar {
			style = w.OK
			if !model.Defined(strings.TrimSuffix(strings.TrimPrefix(seg.Text, "{{"), "}}"), vars) {
				style = w.Error
			}
		}
		segs = append(segs, Span{Text: seg.Text, Style: style})
	}
	return segs
}

// Templates, parsed once above.
const (
	urlbarSource = `<row h="{{urlH}}" pad-x="{{gutter}}" pad-y="{{looseGap}}" gap="{{looseGap}}" bg="bg">
  <row grow="1" bg="sunken" control="field" tooltip="{{urlTip}}">
    <row w="{{protocolChipW}}" on-click="cycleProtocol" tooltip="Protocol: click to change">
      <text text-center grow="1" color="muted">{{protocol}}</text>
    </row>
    <!-- a divider between the chips, none after the last one -->
    <vrule-inset if="divided" inset="8"/>
    <row if="httpMethod" w="{{methodChipW}}" on-click="cycleMethod" tooltip="Method: click to change">
      <text text-center grow="1" color="{{methodColor}}">{{method}}</text>
    </row>
    <row if="rpcMethod" w="{{grpcMethodW}}" pad-x="{{labelPad}}" on-click="pickMethod" tooltip="Method: click to list the server's methods">
      <text ellipsis grow="1" color="{{grpcMethodColor}}">{{grpcMethod}}</text>
    </row>
    <vrule-inset if="streamable" inset="8"/>
    <row if="streamable" w="{{streamChipW}}" bg="{{streamBg}}" on-click="toggleStream" tooltip="Stream: keep the call open and show every message in the Messages pane">
      <text text-center grow="1" color="{{streamColor}}">Stream</text>
    </row>
    <!-- breathing room between the divider and the URL text -->
    <spacer w="{{urlTextGap}}"/>
    <field state="url" grow="1" placeholder="Enter URL" styled="segments" on-change="edited" on-enter="send" text-menu="url"/>
  </row>
  <primary-button if="idle" min-w="{{sendBtnMinW}}" on-click="send" tooltip="{{sendTip}}">Send</primary-button>
  <primary-button if="socket" min-w="{{connectMinW}}" on-click="toggleConnection" tooltip="{{connectionTip}}">{{connectionLabel}}</primary-button>
  <primary-button if="sending" w="{{cancelBtnW}}" on-click="cancel" tooltip="Cancel the running request">Cancel</primary-button>
</row>
`
)
