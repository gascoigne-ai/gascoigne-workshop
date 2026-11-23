package body

import (
	"reflect"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

const (
	schemaIndent  = 16 // pixels a nested message is moved right
	fillBodyW     = 92
	reloadButtonW = 76
)

var schemaTemplate = markup.Must("schemapane.vml", schemaSource)

// messageSchema is what a tab keeps for its Schema pane between frames:
// the last answer, and what it was asked for. Audit note: running, schema
// and err are a flag bundle over idle, loading, loaded and failed.
type messageSchema struct {
	asked   schemaKey
	running bool
	schema  *request.MessageSchema
	err     string
}

// schemaKey is what a schema depends on; another one asks again.
type schemaKey struct {
	URL, Method string
	Proto       request.ProtoSource
}

func schemaKeyOf(req *request.Request) schemaKey {
	return schemaKey{URL: req.URL, Method: req.Method, Proto: req.Proto}
}

// methodChosen says the request names a gRPC method, not the HTTP one a
// fresh tab still carries.
func methodChosen(req *request.Request) bool { return strings.Contains(req.Method, "/") }

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
func schemaRows(schema request.MessageSchema) []schemaRow {
	var rows []schemaRow
	appendSchemaRows(&rows, schema, 0)
	return rows
}

func appendSchemaRows(rows *[]schemaRow, schema request.MessageSchema, depth int) {
	for _, field := range schema.Fields {
		*rows = append(*rows, schemaRow{Depth: depth, Text: fieldText(field)})
		if field.Message != nil && !field.Message.Cut {
			appendSchemaRows(rows, *field.Message, depth+1)
		}
	}
}

func fieldText(field request.SchemaField) string {
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

// SchemaView is the Schema pane of a gRPC tab: the fields of the request
// message of its method, and a button that fills the Body with a JSON
// skeleton of them. It asks for the message when the request changed.
func (c *Component) SchemaView(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	if !methodChosen(tab.Req) {
		return c.deps.Slots.Message("Pick a method to see its request message")
	}
	c.loadSchema(tab)
	state := c.state(tab).schema
	rows := []schemaRowView{}
	title := ""
	if state.schema != nil {
		title = state.schema.Name
		for _, row := range schemaRows(*state.schema) {
			rows = append(rows, schemaRowView{Pad: row.Depth * schemaIndent, Text: row.Text})
		}
	}
	return schemaTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"scroll":  scroll,
		"fillW":   fillBodyW,
		"reloadW": reloadButtonW,
		"loaded":  state.schema != nil,
		"loading": state.running && state.schema == nil,
		"failed":  state.err != "",
		"error":   state.err,
		"title":   title,
		"rows":    rows,
		"fill":    func() { c.fill(tab, state.schema) },
		"reload":  func() { c.reload(tab) },
	}))
}

// SchemaPending reports whether the message of tab's method is still being
// asked for, or not asked for yet, so a capture waits for it.
func (c *Component) SchemaPending(tab *tabs.Tab) bool {
	if !methodChosen(tab.Req) {
		return false
	}
	state := c.state(tab).schema
	return state.running || !reflect.DeepEqual(state.asked, schemaKeyOf(tab.Req))
}

// loadSchema asks for the request's message in the background when the
// request changed since the last answer. An answer for an older request is
// dropped by the key comparison and the next build asks again.
func (c *Component) loadSchema(tab *tabs.Tab) {
	key := schemaKeyOf(tab.Req)
	state := &c.state(tab).schema
	if state.running || (state.asked.Method != "" && reflect.DeepEqual(state.asked, key)) {
		return
	}
	state.running, state.asked = true, key
	c.deps.Schemas.Describe(tab, func(schema request.MessageSchema, err error) {
		*state = messageSchema{asked: key}
		if err != nil {
			state.err = err.Error()
			return
		}
		state.schema = &schema
	})
}

// reload forgets the answer, so the next frame asks again.
func (c *Component) reload(tab *tabs.Tab) { c.state(tab).schema = messageSchema{} }

// fill replaces the body with the skeleton of schema, as typing it would
// (so it can be undone), and shows the Body pane.
func (c *Component) fill(tab *tabs.Tab, schema *request.MessageSchema) {
	if schema == nil {
		return
	}
	c.Replace(tab, schema.Skeleton())
	c.deps.Window.ShowBody(tab)
}

// Template, parsed once above.
const schemaSource = `<scroll state="scroll" pad="{{gutter}}" gap="{{looseGap}}">
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
