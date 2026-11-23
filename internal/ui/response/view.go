package response

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// paneViews draws each pane, by its Pane. An array over the enum: the panes
// are fixed and each has exactly one view.
var paneViews = [PaneCount]func(c *Component, ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node{
	Body:     (*Component).bodyView,
	Headers:  (*Component).headersView,
	Timeline: (*Component).timelineView,
	Tests:    (*Component).testsView,
}

// View is the contents of pane for tab, scrolled with scroll where the pane
// scrolls itself. The application draws the strip above it and the message
// that replaces it while there is no response to show.
func (c *Component) View(ctx *w.Ctx, tab *tabs.Tab, pane Pane, scroll *widget.ScrollState) Node {
	return paneViews[pane](c, ctx, tab, scroll)
}

// Empty is the message that replaces the Body pane for a response that has
// no text to show, and whether it is an error. A picture and a body that is
// not text are told apart here; "no response" is the application's, which
// knows what is in flight.
func (c *Component) Empty(tab *tabs.Tab) (message string, isErr bool) {
	if tab.Resp == nil {
		return "", false
	}
	if state, ok := c.states[tab]; ok && state.picture != nil {
		return "", false
	}
	if respview.LooksBinary(tab.Resp.Body) {
		return fmt.Sprintf("Binary body, %s (not shown)", format.Size(len(tab.Resp.Body))), false
	}
	return "", false
}

// The text view and the picture are two roots, so two templates; Go
// picks one in bodyView.
var (
	responsePaneTemplate    = markup.Must("responsepane.vml", responsepaneSource)
	responsePictureTemplate = markup.Must("responsepicture.vml", responsepictureSource)
)

// bodyView is the response body with its toolbar and find bar, or
// the picture of an image body with its size and a Save button.
func (c *Component) bodyView(ctx *w.Ctx, tab *tabs.Tab, _ *widget.ScrollState) Node {
	state := c.state(tab)
	if state.picture != nil {
		return responsePictureTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
			"pictureInfo": state.pictureInfo,
			"picture":     state.picture,
			"save":        func() { c.saveBody(tab) },
		}))
	}
	var findBar Node // stays nil, so the slot is left out, while the bar is closed
	if state.look.FindOpen {
		findBar = c.findBar(ctx, tab)
	}
	return responsePaneTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"compact":     state.look.Compact(),
		"pretty":      state.look.Mode == viewer.Pretty,
		"raw":         state.look.Raw(),
		"wrap":        state.view.Wrap,
		"history":     len(tab.Earlier) > 0,
		"findTip":     c.deps.Hints.Hint("Find in response", "find"),
		"findBar":     findBar,
		"view":        &state.view,
		"showCompact": func() { c.SetMode(tab, viewer.Compact) },
		"showPretty":  func() { c.SetMode(tab, viewer.Pretty) },
		"showRaw":     func() { c.SetMode(tab, viewer.Raw) },
		"toggleWrap":  func() { c.toggleWrap(tab) },
		"openHistory": func(posX, posY int) { c.deps.Window.Open(posX, posY, c.historyMenu(tab)) },
		"openFind":    c.OpenFind,
		"copyBody":    func() { c.copyBody(ctx, tab) },
		"copyCurl":    func() { c.copyCurl(ctx, tab) },
		"openMenu":    func(posX, posY int) { c.deps.Window.Open(posX, posY, c.menuItems(ctx, tab)) },
	}))
}

var findBarTemplate = markup.Must("findbar.vml", findbarSource)

// findBar is the search row above the response: the query, the hit
// counter and the previous, next and close buttons.
func (c *Component) findBar(ctx *w.Ctx, tab *tabs.Tab) Node {
	state := c.state(tab)
	return findBarTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"find":     &state.find,
		"count":    viewer.Count(state.find.Text, len(state.view.Matches()), state.look.Match),
		"search":   func() { c.runFind(tab) },
		"enter":    func() { c.findEnter(ctx, tab) },
		"previous": func() { c.findStep(tab, -1) },
		"next":     func() { c.findStep(tab, 1) },
		"close":    func() { c.CloseFind(tab) },
	}))
}

var headersTemplate = markup.Must("responseheaders.vml", responseheadersSource)

// headersView is the response headers as a table to read: no checkboxes and
// no edit, a menu that copies a row.
func (c *Component) headersView(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	var rows []request.KV
	if tab.Resp != nil {
		rows = tab.Resp.Headers
	}
	spec := widget.TableSpec{
		Styler:  c.deps.Headers.CellStyler(tab.Req),
		State:   c.deps.Headers.Widths(),
		Columns: []string{"Name", "Value"},
		Weights: []float64{35, 65},
		Rows:    headerRows(rows),
		OnRowMenu: func(row, posX, posY int) {
			c.deps.Window.Open(posX, posY, copyRowItems(ctx, rows, row))
		},
	}
	return headersTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"pad":    w.Gutter,
		"scroll": scroll,
		"spec":   spec,
	}))
}

// headerRows turns the headers into table rows of a name and a value.
func headerRows(headers []request.KV) []widget.TableRow {
	out := make([]widget.TableRow, len(headers))
	for i, header := range headers {
		out[i] = widget.TableRow{Cells: []string{header.Name, header.Value}, Checked: header.Enabled, Muted: !header.Enabled}
	}
	return out
}

var timelineTemplate = markup.Must("notepane.vml", notepaneSource)

// timelineView is the timing of the response as text.
func (c *Component) timelineView(_ *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return timelineTemplate.MustBuild(markup.Scope{
		"gutter": w.Gutter,
		"scroll": scroll,
		"note":   respview.Timeline(tab.Resp),
	})
}

// testsView is the Tests pane, which the assertions component draws.
func (c *Component) testsView(_ *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	return c.deps.Slots.Results(tab, scroll)
}

var statusTemplate = markup.Must("responsestatus.vml", responsestatusSource)

// StatusView is "200 OK  1.4s  156.67KB" in a pill colored by the status,
// nil while there is no answer to show; the window drops it when the tabs
// need the room.
func (c *Component) StatusView(ctx *w.Ctx, tab *tabs.Tab) Node {
	if tab.Resp == nil || tab.Resp.Err != "" {
		return nil
	}
	fg, bg := statusPill(tab.Resp.Status)
	return statusTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"pillFg": fg,
		"pillBg": bg,
		"status": fmt.Sprintf("%d %s", tab.Resp.Status, tab.Resp.StatusText),
		"timing": fmt.Sprintf("%s  %s", format.Duration(tab.Resp.Duration), format.Size(tab.Resp.Size)),
	}))
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

// Templates, parsed once above.
const (
	responsepaneSource = `<col grow="1">
  <!-- same top inset as the request pane, so first rows line up -->
  <spacer h="{{gutter}}"/>
  <!-- written out: markup has no toolbar element; this is widget.Toolbar, copied in responsepicture.vml -->
  <row h="{{toolbarH}}" pad-x="{{gutter}}" gap="{{tightGap}}" align-items="center" border-b="border">
    <segment-chip selected="{{compact}}" w="{{compactChipW}}" h="{{compactH}}" on-click="showCompact">Compact</segment-chip>
    <segment-chip selected="{{pretty}}" w="{{prettyChipW}}" h="{{compactH}}" on-click="showPretty">Pretty</segment-chip>
    <segment-chip selected="{{raw}}" w="{{rawChipW}}" h="{{compactH}}" on-click="showRaw">Raw</segment-chip>
    <row w="{{gutter}}"/>
    <segment-chip selected="{{wrap}}" w="{{wrapChipW}}" h="{{compactH}}" on-click="toggleWrap">Wrap</segment-chip>
    <spacer/>
    <button if="history" w="{{historyBtnW}}" h="{{compactH}}" on-click-at="openHistory" tooltip="Earlier responses of this request">History</button>
    <icon-button icon="search" w="{{iconBtnW}}" h="{{compactH}}" on-click="openFind" tooltip="{{findTip}}"/>
    <icon-button icon="copy" w="{{iconBtnW}}" h="{{compactH}}" on-click="copyBody" tooltip="Copy the response body"/>
    <icon-button icon="prompt" w="{{iconBtnW}}" h="{{compactH}}" on-click="copyCurl" tooltip="Copy the request as a cURL command"/>
  </row>
  <slot if="findBar" name="findBar"/>
  <code state="view" grow="1" bg="sunken" wrap="{{wrap}}" on-right-click="openMenu"/>
</col>
`

	responsepictureSource = `<col grow="1">
  <spacer h="{{gutter}}"/>
  <!-- written out: markup has no toolbar element; this is widget.Toolbar, copied from responsepane.vml -->
  <row h="{{toolbarH}}" pad-x="{{gutter}}" gap="{{tightGap}}" align-items="center" border-b="border">
    <text small muted>{{pictureInfo}}</text>
    <spacer/>
    <button w="{{saveBtnW}}" h="{{compactH}}" on-click="save" tooltip="Save the image to a file">Save</button>
  </row>
  <image img="picture" grow="1" bg="sunken" tooltip="The image, scaled down to fit"/>
</col>
`

	findbarSource = `<row h="{{rowH}}" pad-x="{{gutter}}" gap="{{tightGap}}" border-b="border">
  <field state="find" grow="1" placeholder="Find" on-change="search" on-enter="enter" text-menu="find"/>
  <text small muted w="{{countW}}">{{count}}</text>
  <icon-button icon="chevron-up" w="{{iconBtnW}}" h="{{compactH}}" on-click="previous" tooltip="Previous match"/>
  <icon-button icon="chevron-down" w="{{iconBtnW}}" h="{{compactH}}" on-click="next" tooltip="Next match"/>
  <icon-button icon="cross" w="{{iconBtnW}}" h="{{compactH}}" on-click="close" tooltip="Close search (Esc)"/>
</row>
`

	responseheadersSource = `<scroll state="scroll" pad="{{pad}}" gap="{{tightGap}}">
  <table spec="spec"/>
</scroll>
`

	notepaneSource = `<scroll state="scroll" pad="{{gutter}}">
  <lines color="text">{{note}}</lines>
</scroll>
`

	responsestatusSource = `<row gap="{{gutter}}" pad-r="{{gutter}}" optional align-items="center">
  <row pad-x="{{labelPad}}" h="{{slimH}}" bg="{{pillBg}}" control="pill" align-items="center">
    <text small bold color="{{pillFg}}">{{status}}</text>
  </row>
  <text>{{timing}}</text>
</row>
`
)
