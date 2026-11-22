package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// bodyChoice is one choice of the body type row.
type bodyChoice struct {
	id    request.BodyType
	label string
	width int
}

// bodyTypes are the choices of the body type row.
var bodyTypes = []bodyChoice{
	{request.BodyNone, "None", 64},
	{request.BodyJSON, "JSON", 64},
	{request.BodyText, "Text", 64},
	{request.BodyForm, "Form", 64},
	{request.BodyMultipart, "Multipart", 84},
	{request.BodyGraphQL, "GraphQL", 80},
	{request.BodyFile, "File", 64},
}

// formPane is the table of form and multipart fields.
var formPane = pane{label: "Body", kind: paneTable, title: "Fields", editable: true,
	rows: func(_ *App, tab *reqTab) *[]request.KV { return &tab.Req.Form }}

// The pane and its two simplest editors are templates; the editor is
// picked in Go (bodyEditor) and enters the pane through a slot.
var (
	bodyPaneTemplate = markup.Must("bodypane.vml", bodypaneSource)
	bodyAreaTemplate = markup.Must("bodyarea.vml", bodyareaSource)
	bodyNoneTemplate = markup.Must("bodynone.vml", bodynoneSource)
)

// bodyChip is one chip of the body type row, as bodypane.vml shows it.
type bodyChip struct {
	Label    string
	Width    int
	Selected bool // the request's body type; also kept in view
	Choose   func()
}

// bodyPane is the body editor: a type chooser, then an editor that fits the type.
func (a *App) bodyPane(panel pane, tab *reqTab, scroll *ScrollState) Node {
	kind := tab.Req.BodyType
	isJSON := kind == request.BodyJSON
	canWrap := isJSON || kind == request.BodyText // the editors that are a text area
	status, valid := textfmt.JSONStatus(tab.body.Text)
	statusColor := w.Error
	if valid {
		statusColor = w.OK
	}
	chips := make([]bodyChip, len(bodyTypes))
	for i, bodyType := range bodyTypes {
		id := bodyType.id
		chips[i] = bodyChip{Label: bodyType.label, Width: bodyType.width, Selected: id == kind,
			Choose: func() { a.setBodyType(tab, id) }}
	}
	return bodyPaneTemplate.MustBuild(a.scope(markup.Scope{
		"typesX":      &tab.bodyTypesX,
		"chips":       chips,
		"showStatus":  isJSON && status != "",
		"status":      status,
		"statusColor": statusColor,
		"isJSON":      isJSON,
		"canWrap":     canWrap,
		"wrap":        tab.body.Wrap,
		"toggleWrap":  func() { a.toggleBodyWrap(tab) },
		"format":      func() { a.formatBody(tab) },
		"editor":      a.bodyEditor(tab, scroll),
	}))
}

// bodyEditor is the part of the body pane under the type chooser.
func (a *App) bodyEditor(tab *reqTab, scroll *ScrollState) Node {
	switch tab.Req.BodyType {
	case request.BodyJSON, request.BodyText:
		base := plainStyler
		if tab.Req.BodyType == request.BodyJSON {
			base = jsonStyler
		}
		tab.body.Styler = a.withVariables(base, tab.Req) // {{variables}} stand out, as in the URL
		return bodyAreaTemplate.MustBuild(markup.Scope{
			"body":     &tab.body,
			"wrap":     tab.body.Wrap,
			"bodyText": &tab.body.TextField,
			"sync":     func() { editRequest(tab, editor.BodyTyped{Text: tab.body.Text}) },
		})
	case request.BodyForm, request.BodyMultipart:
		return a.tableView(formPane, tab, scroll, 0)
	case request.BodyFile:
		return a.fileBody(tab)
	case request.BodyGraphQL:
		return a.graphQLBody(tab)
	}
	return bodyNoneTemplate.MustBuild(markup.Scope{})
}

// toggleBodyWrap switches the body editor between scrolling sideways and
// breaking long lines at the width of the pane, per tab like the
// response's Wrap.
func (a *App) toggleBodyWrap(tab *reqTab) { tab.body.Wrap = !tab.body.Wrap }

func (a *App) setBodyType(tab *reqTab, id request.BodyType) {
	a.commitEdit()
	editRequest(tab, editor.BodyTypeChosen{Type: id, Text: tab.body.Text, FilePath: tab.file.Text})
}

// formatBody re-indents a JSON body. Invalid JSON is left alone.
func (a *App) formatBody(tab *reqTab) {
	out, err := textfmt.IndentJSON(tab.body.Text)
	if err != nil {
		a.status = "Cannot format body: " + err.Error()
		return
	}
	tab.body.SelectAll()
	tab.body.Insert(out) // as an edit, so it can be undone
	editRequest(tab, editor.BodyTyped{Text: tab.body.Text})
}

// jsonStyler colors one line of a JSON body.
func jsonStyler(line string) []Span {
	src := jsonview.Tokenize(line)
	spans := make([]Span, len(src))
	for i, s := range src {
		spans[i] = Span{Text: s.Text, Style: respview.Tokens[s.Kind]}
	}
	return spans
}

// Templates, parsed once above.
const (
	bodypaneSource = `<!-- The body pane: the type chips with, for JSON, whether it parses;
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

	bodyareaSource = `<area state="body" grow="1" bg="sunken" wrap="{{wrap}}" on-change="sync" text-menu="bodyText"/>
`

	bodynoneSource = `<lines color="muted">This request has no body</lines>
`
)
