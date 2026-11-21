package app

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// schemaPane is the Schema tab of a gRPC request: the fields of the
// request message of its method, and a button that fills the Body with a
// JSON skeleton of them.
var schemaPane = pane{label: "Schema", kind: paneSchema}

const (
	schemaIndent  = 16 // pixels a nested message is moved right
	fillBodyW     = 92
	reloadButtonW = 76
)

// messageSchemaState is what a tab keeps for its Schema pane between
// frames: the last answer, and what it was asked for.
type messageSchemaState struct {
	asked   schemaKey
	running bool
	schema  *model.MessageSchema
	err     string
}

// schemaKey is what a schema depends on; another one asks again.
type schemaKey struct {
	URL, Method string
	Proto       model.ProtoSource
}

func schemaKeyOf(req *model.Request) schemaKey {
	return schemaKey{URL: req.URL, Method: req.Method, Proto: req.Proto}
}

// methodChosen says the request names a gRPC method, not the HTTP one a
// fresh tab still carries.
func methodChosen(req *model.Request) bool { return strings.Contains(req.Method, "/") }

type schemaRow struct {
	Depth int
	Text  string
}

// schemaRowView is a row as the template shows it.
type schemaRowView struct {
	Pad  int
	Text string
}

// schemaRows lists the fields of schema one to a row, a nested message's
// fields right below it one level deeper.
func schemaRows(schema model.MessageSchema) []schemaRow {
	var rows []schemaRow
	appendSchemaRows(&rows, schema, 0)
	return rows
}

func appendSchemaRows(rows *[]schemaRow, schema model.MessageSchema, depth int) {
	for _, field := range schema.Fields {
		*rows = append(*rows, schemaRow{Depth: depth, Text: fieldText(field)})
		if field.Message != nil && !field.Message.Cut {
			appendSchemaRows(rows, *field.Message, depth+1)
		}
	}
}

func fieldText(field model.SchemaField) string {
	text := field.Name + ": "
	if field.Repeated {
		text += "repeated "
	}
	text += field.Type
	if len(field.EnumValues) > 0 {
		text += " (" + strings.Join(field.EnumValues, ", ") + ")"
	}
	if field.Oneof != "" {
		text += ", oneof " + field.Oneof
	}
	if field.Message != nil && field.Message.Cut {
		text += " (recursive)"
	}
	return text
}

func (a *App) schemaPane(panel pane, tab *reqTab, scroll *ScrollState) Node {
	if !methodChosen(tab.Req) {
		return a.messagePane(panel, "Pick a method to see its request message", false)
	}
	a.loadMessageSchema(tab)
	state := tab.messageSchema
	rows := []schemaRowView{}
	title := ""
	if state.schema != nil {
		title = state.schema.Name
		for _, row := range schemaRows(*state.schema) {
			rows = append(rows, schemaRowView{Pad: row.Depth * schemaIndent, Text: row.Text})
		}
	}
	return schemaPaneTemplate.MustBuild(a.scope(markup.Scope{
		"scroll":  scroll,
		"fillW":   fillBodyW,
		"reloadW": reloadButtonW,
		"loaded":  state.schema != nil,
		"loading": state.running && state.schema == nil,
		"failed":  state.err != "",
		"error":   state.err,
		"title":   title,
		"rows":    rows,
		"fill":    func() { a.fillBody(tab, state.schema) },
		"reload":  func() { tab.messageSchema = messageSchemaState{} },
	}))
}

// schemaLoading reports whether t shows its Schema pane while the message
// is still being asked for, so a capture waits for it.
func schemaLoading(tab *reqTab) bool {
	panes := requestPanesFor(tab)
	if tab.paneTab < 0 || tab.paneTab >= len(panes) || panes[tab.paneTab].kind != paneSchema || !methodChosen(tab.Req) {
		return false
	}
	return tab.messageSchema.running || !reflect.DeepEqual(tab.messageSchema.asked, schemaKeyOf(tab.Req))
}

// loadMessageSchema asks for the request's message in the background when the
// request changed since the last answer. An answer for an older request
// is dropped by the key comparison and the next build asks again.
func (a *App) loadMessageSchema(tab *reqTab) {
	key := schemaKeyOf(tab.Req)
	state := &tab.messageSchema
	if state.running || (state.asked.Method != "" && reflect.DeepEqual(state.asked, key)) {
		return
	}
	state.running, state.asked = true, key
	req := tab.Req.Expanded(a.varsFor(tab.Req))
	timeout := a.timeoutFor(req) // read here: the goroutine must not touch App fields
	a.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		schema, err := a.describeGRPCInput(ctx, req)
		return func() { // runs on the UI goroutine
			*state = messageSchemaState{asked: key}
			if err != nil {
				state.err = err.Error()
				return
			}
			state.schema = &schema
		}
	})
}

func (a *App) describeGRPCInput(ctx context.Context, req model.Request) (model.MessageSchema, error) {
	if a.grpcSchema == nil {
		return model.MessageSchema{}, errors.New("no gRPC client")
	}
	return a.grpcSchema(ctx, req)
}

// fillBody replaces the body with the skeleton of schema, as typing it
// would (so it can be undone), and shows the Body pane.
func (a *App) fillBody(tab *reqTab, schema *model.MessageSchema) {
	if schema == nil {
		return
	}
	tab.body.SelectAll()
	tab.body.Insert(schema.Skeleton())
	editRequest(tab, editor.BodyTyped{Text: tab.body.Text})
	for i, p := range requestPanesFor(tab) {
		if p.kind == paneBody {
			tab.paneTab = i
		}
	}
}

var schemaPaneTemplate = markup.Must("schemapane.vml", schemapaneSource)

// Templates, parsed once above.
const (
	schemapaneSource = `<scroll state="scroll" pad="{{gutter}}" gap="{{looseGap}}">
  <row h="{{rowH}}" gap="{{tightGap}}" align-items="center">
    <button if="loaded" w="{{fillW}}" on-click="fill">Fill body</button>
    <button w="{{reloadW}}" on-click="reload">Reload</button>
  </row>
  <text if="loading" muted>Loading...</text>
  <text if="failed" small color="error">{{error}}</text>
  <text if="loaded" bold>{{title}}</text>
  <row for="row in rows" pad-l="{{row.Pad}}" h="{{rowH}}" align-items="center">
    <text mono>{{row.Text}}</text>
  </row>
</scroll>
`
)
