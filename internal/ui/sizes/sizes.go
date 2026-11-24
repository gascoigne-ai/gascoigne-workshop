// Package sizes holds the fixed sizes of the app's chrome and the template
// scope that carries them.
package sizes

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// StatusH is the height of the status bar. It lives here, not in the
// statusbar package, because the toasts stack above the bar and may not
// import a sibling ui package.
const StatusH = 28

// Build builds template for own, which can read the fixed widths and the
// sizes of the faces ctx draws with. They are two more scopes the build looks
// at after own (see markup.Template.Build), not entries copied into a new map
// per component: one frame builds hundreds of templates and the merge was a
// quarter of the bytes the build allocated. own wins over both, and no name
// is in both (see TestFixedAndCtxSizeNamesAreDisjoint), so their order does
// not matter. The build only reads the three maps.
//
// Rejected: a merged map cached and cloned per build (the clone is the cost
// that went); a package-level cache of the merge (global mutable state).
// Revisit when a template needs a size scope other than ctx's, such as one
// per pane.
func Build(ctx *core.Ctx, template *markup.Template, own markup.Scope) view.Node {
	return template.MustBuild(own, fixedSizes, ctx.SizesScope())
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
	ModeChipW    = 36  // one icon chip of the sidebar's mode switch
	CompactChipW = 72  // the Compact chip of a response
	PrettyChipW  = 56  // the Pretty chip of a response
	RawChipW     = 44  // the Raw chip of a response
	HistoryBtnW  = 66  // the History button of a response
	SaveBtnW     = 56  // the Save button of an image response
	CountW       = 64  // the item count beside the response chips

	// The URL bar.
	ProtocolChipW = 64  // the protocol chip
	MethodChipW   = 84  // the HTTP method chip
	StreamChipW   = 56  // the Stream chip of a gRPC call
	SendBtnMinW   = 80  // the Send button
	ConnectMinW   = 96  // the Connect and Disconnect button
	CancelBtnW    = 80  // the Cancel button of a running request
	GRPCMethodW   = 200 // the method chip of a gRPC call: wide, as method names are long

	// The socket bar (WebSocket and MCP).
	ReconnectChipW = 116 // the Auto-reconnect chip
	ToolsBtnMinW   = 84  // the Tools button of an MCP bar
	ResourcesMinW  = 104 // the Resources button
	PromptsBtnMinW = 94  // the Prompts button
	SmallBtnMinW   = 56  // Ping and Clear
	FormatMsgMinW  = 40  // the { } button of the message composer
	SendMsgMinW    = 70  // the Send button of the message composer

	// Tabs, the header and the sidebar tree.
	TabsH         = 42  // the request tab row: a tab's height plus the room above it
	TabBadgeW     = 48  // the slot of a request tab's method badge, so editing never changes the tab's width
	TabDirtyW     = 12  // the slot of a request tab's unsaved dot
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
	"timeoutW": TimeoutW, "modeChipW": ModeChipW, "compactChipW": CompactChipW, "prettyChipW": PrettyChipW,
	"rawChipW": RawChipW, "historyBtnW": HistoryBtnW, "saveBtnW": SaveBtnW, "countW": CountW,
	"protocolChipW": ProtocolChipW, "methodChipW": MethodChipW, "streamChipW": StreamChipW,
	"sendBtnMinW": SendBtnMinW, "connectMinW": ConnectMinW, "cancelBtnW": CancelBtnW, "grpcMethodW": GRPCMethodW,
	"reconnectChipW": ReconnectChipW, "toolsBtnMinW": ToolsBtnMinW, "resourcesMinW": ResourcesMinW,
	"promptsBtnMinW": PromptsBtnMinW, "smallBtnMinW": SmallBtnMinW, "formatMsgMinW": FormatMsgMinW,
	"sendMsgMinW": SendMsgMinW, "tabStripeW": TabStripeW, "tabLabelInset": TabLabelInset,
	"tabCloseW": TabCloseW, "newTabW": NewTabW, "titleMinW": TitleMinW, "envMinW": EnvMinW, "envMaxW": EnvMaxW,
	"treeStripeW": TreeStripeW, "treeStripeH": TreeStripeH, "treeTriangleW": TreeTriangleW,
	"detailGap": DetailGap, "dialogButtonMinW": core.DialogButtonMinW,
}

// Sizes the sidebar shares with the window and the tabs.
const (
	HeaderH    = 44 // the banner: title, environment chip, settings; the sidebar's header row is as high
	URLBarH    = 56 // the URL bar: its controls plus padding above and below
	TreeIndent = 16 // how far each folder level of the tree moves right
	BadgeW     = 40 // the slot of a method or protocol badge in a tree row
)

// BadgeLabels are the method and protocol badges the tree and the tabs show.
var BadgeLabels = []string{"GET", "POST", "PUT", "PATCH", "DEL", "WS", "SSE", "MCP", "gRPC"}

// BadgeOverhang is how far a badge may stick out of its slot.
const BadgeOverhang = 1

// BadgeSlot is the width of a badge slot: base, or the widest badge in the
// small face when a large text size makes that wider, so no size clips a
// badge. It is one width for every row, so a changed method moves nothing.
// A badge may overhang its slot by BadgeOverhang px, as PATCH does in the
// tree at the default size, where the gap to the name absorbs it; that keeps
// the default size as it was.
func BadgeSlot(ctx *core.Ctx, base int) int {
	widest := base
	for _, label := range BadgeLabels {
		widest = max(widest, ctx.Width(ctx.F.Small, label)-BadgeOverhang)
	}
	return widest
}
