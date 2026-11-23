// Package sizes holds the fixed sizes of the app's chrome and the template
// scope that carries them.
package sizes

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
)

// StatusH is the height of the status bar. It lives here, not in the
// statusbar package, because the toasts stack above the bar and may not
// import a sibling ui package.
const StatusH = 28

// Scope is the scope of a template: the sizes of the faces the context
// draws, the fixed widths below, then own, so a view can still override a
// name for itself.
func Scope(ctx *core.Ctx, own markup.Scope) markup.Scope {
	merged := markup.Scope{}
	for name, value := range ctx.Sizes.Scope() {
		merged[name] = value
	}
	for name, value := range fixedSizes {
		merged[name] = value
	}
	for name, value := range own {
		merged[name] = value
	}
	return merged
}

// Widths of the buttons, chips and slots of the app, by role. Each holds a
// label of the default face with its padding (a content size, not a rule over
// a line height), so it stays a fixed number; the layout lets a longer label
// grow where the element is a minimum (`min-w`).
const (
	// Chips and buttons of the editor panes.
	WrapChipW    = 56  // the Wrap chip of a body and a response
	FormatBtnW   = 70  // the Format button of the body pane
	ChooseBtnW   = 90  // the Choose... button of the file body
	SchemaBtnW   = 104 // the schema buttons of the GraphQL pane
	TimeoutW     = 80  // the timeout field of the request settings
	ModeBtnW     = 96  // the collections/requests mode button of the sidebar
	CompactChipW = 72  // the Compact chip of a response
	PrettyChipW  = 56  // the Pretty chip of a response
	RawChipW     = 44  // the Raw chip of a response
	HistoryBtnW  = 66  // the History button of a response
	SaveBtnW     = 56  // the Save button of an image response
	CountW       = 64  // the item count beside the response chips

	// The URL bar.
	ProtocolChipW = 64 // the protocol chip
	MethodChipW   = 84 // the HTTP method chip
	StreamChipW   = 56 // the Stream chip of a gRPC call
	SendBtnMinW   = 80 // the Send button
	ConnectMinW   = 96 // the Connect and Disconnect button
	CancelBtnW    = 80 // the Cancel button of a running request

	// The socket bar (WebSocket and MCP).
	ReconnectChipW = 116 // the Auto-reconnect chip
	ToolsBtnMinW   = 84  // the Tools button of an MCP bar
	ResourcesMinW  = 104 // the Resources button
	PromptsBtnMinW = 94  // the Prompts button
	SmallBtnMinW   = 56  // Ping and Clear
	FormatMsgMinW  = 40  // the { } button of the message composer
	SendMsgMinW    = 70  // the Send button of the message composer

	// Tabs, the header and the sidebar tree.
	TabStripeW    = 3   // the collection color stripe of a request tab
	TabLabelInset = 11  // room before the label of a request tab, past the stripe
	TabCloseW     = 24  // the close slot of a request tab
	NewTabW       = 40  // the new request slot after the tabs
	TitleMinW     = 48  // the least width of the header's title
	EnvMinW       = 24  // the least width of the environment chip
	EnvMaxW       = 200 // the chip stops here and cuts a long name
	TreeStripeW   = 3   // the collection stripe of a tree row
	TreeStripeH   = 14  // and its height
	TreeTriangleW = 12  // the slot of a folder's triangle
	DetailGap     = 10  // the gap between the sections of an MCP detail
)

// fixedSizes are the widths above under the names templates use.
var fixedSizes = map[string]any{
	"wrapChipW": WrapChipW, "formatBtnW": FormatBtnW, "chooseBtnW": ChooseBtnW, "schemaBtnW": SchemaBtnW,
	"timeoutW": TimeoutW, "modeBtnW": ModeBtnW, "compactChipW": CompactChipW, "prettyChipW": PrettyChipW,
	"rawChipW": RawChipW, "historyBtnW": HistoryBtnW, "saveBtnW": SaveBtnW, "countW": CountW,
	"protocolChipW": ProtocolChipW, "methodChipW": MethodChipW, "streamChipW": StreamChipW,
	"sendBtnMinW": SendBtnMinW, "connectMinW": ConnectMinW, "cancelBtnW": CancelBtnW,
	"reconnectChipW": ReconnectChipW, "toolsBtnMinW": ToolsBtnMinW, "resourcesMinW": ResourcesMinW,
	"promptsBtnMinW": PromptsBtnMinW, "smallBtnMinW": SmallBtnMinW, "formatMsgMinW": FormatMsgMinW,
	"sendMsgMinW": SendMsgMinW, "tabStripeW": TabStripeW, "tabLabelInset": TabLabelInset,
	"tabCloseW": TabCloseW, "newTabW": NewTabW, "titleMinW": TitleMinW, "envMinW": EnvMinW, "envMaxW": EnvMaxW,
	"treeStripeW": TreeStripeW, "treeStripeH": TreeStripeH, "treeTriangleW": TreeTriangleW,
	"detailGap": DetailGap, "dialogButtonMinW": core.DialogButtonMinW,
}
