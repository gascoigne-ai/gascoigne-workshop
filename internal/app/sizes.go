package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
)

// scope is the scope of a template: the sizes of the faces the context draws
// with and the fixed widths of the app's controls, under the names a template
// reads, plus the view's own entries (which win). A template never holds a
// size as a number: heights follow ctx.Sizes, the rest is named here.
func (a *App) scope(own markup.Scope) markup.Scope {
	return sizeScope(a.ctx, own)
}

// sizeScope is scope for a view that has the context but not the App.
func sizeScope(ctx *core.Ctx, own markup.Scope) markup.Scope {
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
	wrapChipW    = 56  // the Wrap chip of a body and a response
	formatBtnW   = 70  // the Format button of the body pane
	chooseBtnW   = 90  // the Choose... button of the file body
	schemaBtnW   = 104 // the schema buttons of the GraphQL pane
	timeoutW     = 80  // the timeout field of the request settings
	modeBtnW     = 96  // the collections/requests mode button of the sidebar
	compactChipW = 72  // the Compact chip of a response
	prettyChipW  = 56  // the Pretty chip of a response
	rawChipW     = 44  // the Raw chip of a response
	historyBtnW  = 66  // the History button of a response
	saveBtnW     = 56  // the Save button of an image response
	countW       = 64  // the item count beside the response chips

	// The URL bar.
	protocolChipW = 64 // the protocol chip
	methodChipW   = 84 // the HTTP method chip
	streamChipW   = 56 // the Stream chip of a gRPC call
	sendBtnMinW   = 80 // the Send button
	connectMinW   = 96 // the Connect and Disconnect button
	cancelBtnW    = 80 // the Cancel button of a running request

	// The socket bar (WebSocket and MCP).
	reconnectChipW = 116 // the Auto-reconnect chip
	toolsBtnMinW   = 84  // the Tools button of an MCP bar
	resourcesMinW  = 104 // the Resources button
	promptsBtnMinW = 94  // the Prompts button
	smallBtnMinW   = 56  // Ping and Clear
	formatMsgMinW  = 40  // the { } button of the message composer
	sendMsgMinW    = 70  // the Send button of the message composer

	// Tabs, the header and the sidebar tree.
	tabStripeW    = 3   // the collection color stripe of a request tab
	tabLabelInset = 11  // room before the label of a request tab, past the stripe
	tabCloseW     = 24  // the close slot of a request tab
	newTabW       = 40  // the new request slot after the tabs
	titleMinW     = 48  // the least width of the header's title
	envMinW       = 24  // the least width of the environment chip
	envMaxW       = 200 // the chip stops here and cuts a long name
	treeStripeW   = 3   // the collection stripe of a tree row
	treeStripeH   = 14  // and its height
	treeTriangleW = 12  // the slot of a folder's triangle
	detailGap     = 10  // the gap between the sections of an MCP detail
)

// fixedSizes are the widths above under the names templates use.
var fixedSizes = map[string]any{
	"wrapChipW": wrapChipW, "formatBtnW": formatBtnW, "chooseBtnW": chooseBtnW, "schemaBtnW": schemaBtnW,
	"timeoutW": timeoutW, "modeBtnW": modeBtnW, "compactChipW": compactChipW, "prettyChipW": prettyChipW,
	"rawChipW": rawChipW, "historyBtnW": historyBtnW, "saveBtnW": saveBtnW, "countW": countW,
	"protocolChipW": protocolChipW, "methodChipW": methodChipW, "streamChipW": streamChipW,
	"sendBtnMinW": sendBtnMinW, "connectMinW": connectMinW, "cancelBtnW": cancelBtnW,
	"reconnectChipW": reconnectChipW, "toolsBtnMinW": toolsBtnMinW, "resourcesMinW": resourcesMinW,
	"promptsBtnMinW": promptsBtnMinW, "smallBtnMinW": smallBtnMinW, "formatMsgMinW": formatMsgMinW,
	"sendMsgMinW": sendMsgMinW, "tabStripeW": tabStripeW, "tabLabelInset": tabLabelInset,
	"tabCloseW": tabCloseW, "newTabW": newTabW, "titleMinW": titleMinW, "envMinW": envMinW, "envMaxW": envMaxW,
	"treeStripeW": treeStripeW, "treeStripeH": treeStripeH, "treeTriangleW": treeTriangleW,
	"detailGap": detailGap, "dialogButtonMinW": core.DialogButtonMinW,
}
