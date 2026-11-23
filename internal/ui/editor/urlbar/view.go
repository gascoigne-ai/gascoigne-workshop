package urlbar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var template = markup.Must("urlbar.vml", source)

// MethodLabel is what the method chip of a gRPC tab shows, and in which
// color: the method, or a muted prompt to pick one.
func MethodLabel(req *request.Request) (string, w.Token) {
	if req.Method == "" {
		return "Pick method...", w.Muted
	}
	return req.Method, w.Accent
}

// View is the bar of the current tab.
func (c *Component) View(ctx *w.Ctx) Node {
	tab := c.deps.Tabs.Current()
	info, socket := tab.Req.Protocol.Info(), c.deps.Connections.Socket(tab)
	httpMethod, rpcMethod := info.Method == request.MethodHTTP, info.Method == request.MethodRPC
	sending := !socket && c.deps.Sends.Sending(tab)
	connectionLabel, connectionHint := "Connect", "Open the connection"
	if tab.Active() {
		connectionLabel, connectionHint = "Disconnect", "Close the connection"
	}
	grpcMethod, grpcMethodColor := MethodLabel(tab.Req)
	streaming := c.deps.GRPC.Streaming(tab)
	return sizes.Build(ctx, template, markup.Scope{
		"looseGap":         w.LooseGap,
		"urlH":             sizes.URLBarH,
		"gutter":           w.Gutter,
		"urlTextGap":       w.Gutter - w.NearGap,
		"urlTip":           c.deps.Styles.Tip(tab),
		"protocol":         info.Label,
		"divided":          httpMethod || rpcMethod,
		"httpMethod":       httpMethod,
		"method":           tab.Req.Method,
		"methodColor":      theme.MethodToken(tab.Req.Method),
		"rpcMethod":        rpcMethod,
		"streamable":       info.Connection == request.OnStream,
		"grpcMethod":       grpcMethod,
		"grpcMethodColor":  grpcMethodColor,
		"streamColor":      streamColor(streaming),
		"streamBg":         streamBg(streaming),
		"url":              c.Field(tab),
		"segments":         func(raw string) []widget.Span { return c.deps.Styles.Segments(tab.Req, raw) },
		"idle":             !socket && !sending,
		"socket":           socket,
		"sending":          sending,
		"sendTip":          c.deps.Hints.Hint("Send the request", "send"),
		"connectionLabel":  connectionLabel,
		"connectionTip":    c.deps.Hints.Hint(connectionHint, "send"),
		"cycleProtocol":    func() { c.CycleProtocol(tab) },
		"cycleMethod":      func() { c.CycleMethod(tab) },
		"pickMethod":       func() { c.deps.GRPC.PickMethod(tab) },
		"toggleStream":     func() { c.deps.GRPC.ToggleStream(tab) },
		"edited":           func() { c.Edited(tab) },
		"send":             func() { c.deps.Sends.Send(tab) },
		"toggleConnection": func() { c.deps.Connections.ToggleConnection(tab) },
		"cancel":           func() { c.deps.Sends.Cancel(tab) },
	})
}

// streamColor shows the Stream chip lit while the tab streams.
func streamColor(streaming bool) w.Token {
	if streaming {
		return w.Accent
	}
	return w.Muted
}

// streamBg tints the Stream chip while the tab streams; an empty token
// applies no background (see the markup package doc).
func streamBg(streaming bool) w.Token {
	if streaming {
		return w.Selected
	}
	return ""
}

// Template, parsed once above.
const source = `<row h="{{urlH}}" pad-x="{{gutter}}" pad-y="{{looseGap}}" gap="{{looseGap}}" bg="bg">
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
