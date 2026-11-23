package body

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// bodyChoice is one choice of the body type row.
type bodyChoice struct {
	id    request.BodyType
	label string
	width int
}

// bodyTypes are the choices of the body type row, in the order they are
// shown. A table, not a map: the order is the row's.
var bodyTypes = []bodyChoice{
	{request.BodyNone, "None", 64},
	{request.BodyJSON, "JSON", 64},
	{request.BodyText, "Text", 64},
	{request.BodyForm, "Form", 64},
	{request.BodyMultipart, "Multipart", 84},
	{request.BodyGraphQL, "GraphQL", 80},
	{request.BodyFile, "File", 64},
}

// The pane and its two simplest editors are templates; the editor is
// picked in Go (editor) and enters the pane through a slot.
var (
	paneTemplate = markup.Must("bodypane.vml", paneSource)
	areaTemplate = markup.Must("bodyarea.vml", areaSource)
	noneTemplate = markup.Must("bodynone.vml", noneSource)
)

// bodyChip is one chip of the body type row, as bodypane.vml shows it.
type bodyChip struct {
	Label    string
	Width    int
	Selected bool // the request's body type; also kept in view
	Choose   func()
}

// View is the body pane of tab: a type chooser, then an editor that fits
// the type. scroll is the scroll state of the table editors.
func (c *Component) View(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	state := c.state(tab)
	kind := tab.Req.BodyType
	isJSON := kind == request.BodyJSON
	canWrap := isJSON || kind == request.BodyText // the editors that are a text area
	status, valid := format.JSONStatus(state.area.Text)
	statusColor := w.Error
	if valid {
		statusColor = w.OK
	}
	chips := make([]bodyChip, len(bodyTypes))
	for i, bodyType := range bodyTypes {
		id := bodyType.id
		chips[i] = bodyChip{Label: bodyType.label, Width: bodyType.width, Selected: id == kind,
			Choose: func() { c.ChooseType(tab, id) }}
	}
	return paneTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"typesX":      &state.typesX,
		"chips":       chips,
		"showStatus":  isJSON && status != "",
		"status":      status,
		"statusColor": statusColor,
		"isJSON":      isJSON,
		"canWrap":     canWrap,
		"wrap":        state.area.Wrap,
		"toggleWrap":  func() { c.ToggleWrap(tab) },
		"format":      func() { c.format(tab) },
		"editor":      c.editor(ctx, tab, scroll),
	}))
}

// editor is the part of the body pane under the type chooser.
func (c *Component) editor(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	switch tab.Req.BodyType {
	case request.BodyJSON, request.BodyText:
		return c.areaEditor(tab)
	case request.BodyForm, request.BodyMultipart:
		return c.deps.Slots.FormTable(tab, scroll)
	case request.BodyFile:
		return c.fileBody(ctx, tab)
	case request.BodyGraphQL:
		return c.deps.Slots.GraphQL(tab)
	}
	return noneTemplate.MustBuild(markup.Scope{})
}

// areaEditor is the text area of a JSON or Text body, colored by the type
// with the {{variables}} standing out, as in the URL.
func (c *Component) areaEditor(tab *tabs.Tab) Node {
	state := c.state(tab)
	base := plainStyler
	if tab.Req.BodyType == request.BodyJSON {
		base = jsonStyler
	}
	state.area.Styler = withVariables(base, c.deps.Vars.VarsFor(tab.Req))
	return areaTemplate.MustBuild(markup.Scope{
		"body":     &state.area,
		"wrap":     state.area.Wrap,
		"bodyText": &state.area.TextField,
		"sync":     func() { c.edit(tab, requestedit.BodyTyped{Text: state.area.Text}) },
	})
}

// Templates, parsed once above.
const (
	paneSource = `<!-- The body pane: the type chips with, for JSON, whether it parses;
     for JSON and Text the Wrap chip; for JSON the Format button; the editor of the chosen type below. -->
<col grow="1" pad="{{gutter}}" gap="{{looseGap}}">
  <row h="{{rowH}}" gap="{{tightGap}}">
    <!-- the type chips scroll sideways when the pane is narrow -->
    <hscroll state="typesX" grow="1" gap="{{tightGap}}">
      <segment-chip for="chip in chips" selected="{{chip.Selected}}" w="{{chip.Width}}" keep-in-view="{{chip.Selected}}" on-click="chip.Choose">{{chip.Label}}</segment-chip>
    </hscroll>
    <text if="showStatus" small color="{{statusColor}}" ellipsis optional>{{status}}</text>
    <segment-chip if="canWrap" selected="{{wrap}}" w="{{wrapChipW}}" on-click="toggleWrap">Wrap</segment-chip>
    <button if="isJSON" w="{{formatBtnW}}" on-click="format">Format</button>
  </row>
  <slot name="editor"/>
</col>
`

	areaSource = `<area state="body" grow="1" bg="sunken" wrap="{{wrap}}" on-change="sync" text-menu="bodyText"/>
`

	noneSource = `<lines color="muted">This request has no body</lines>
`
)
