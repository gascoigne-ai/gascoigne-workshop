package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	request "github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// pinnedSchema is a loaded schema with two operations, so the Schema
// button shows its count.
var pinnedSchema = &send.Schema{Operations: []send.Operation{
	{Label: "query repo(name: String!): Repo", Query: "query($name: String!) { repo(name: $name) { name } }", Variables: `{"name": ""}`},
	{Label: "query viewer: User", Query: "{ viewer { login } }"},
}}

// graphQLBodyStates are the looks of the GraphQL body editor: no schema
// and no variables, a loaded schema, and valid or invalid variables.
var graphQLBodyStates = []struct {
	name  string
	setUp func(tab *reqTab)
}{
	{"empty", func(tab *reqTab) {}},
	{"schema", func(tab *reqTab) { tab.schema = pinnedSchema }},
	{"valid-vars", func(tab *reqTab) { tab.gqlVars = widget.NewTextArea(`{"name": "gio"}`) }},
	{"invalid-vars", func(tab *reqTab) { tab.gqlVars = widget.NewTextArea(`{"name": `) }},
}

// The GraphQL body editor is pinned as text (element kinds, options,
// rects) for each state at the request pane's width and at the narrowest
// main area, so moving it to a template cannot change it unseen.
func TestTheGraphQLBodyLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", settingsWideW},
		{"narrow", settingsNarrowW},
	}
	for _, state := range graphQLBodyStates {
		for _, size := range sizes {
			name := "graphql-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a := testApp(t)
				t.Cleanup(a.WaitWrites)
				tab := a.cur()
				a.setBodyType(tab, request.BodyGraphQL)
				state.setUp(tab)
				root := a.graphQLBody(tab)
				Layout(a.ctx, root, size.width, settingsPaneH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// What Dump does not show of the two areas: the query edits the tab's
// body and the variables its own area. The query is drawn without the
// JSON styler a JSON body leaves on the shared body area.
func TestTheGraphQLAreasEditTheTabsTexts(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	tab.body.Styler = jsonStyler
	areas := behaviorStates(a.graphQLBody(tab), "*widget.areaB")
	if len(areas) != 2 {
		t.Fatalf("%d areas, want 2: %q", len(areas), areas)
	}
	for i, state := range []*widget.TextArea{&tab.body, &tab.gqlVars} {
		if want := fmt.Sprintf("%p", state); !strings.Contains(areas[i], want) {
			t.Errorf("area %d lacks %s: %s", i, want, areas[i])
		}
	}
	if tab.body.Styler != nil {
		t.Error("the query area keeps a styler")
	}
}

// The GraphQL body in the test window's Body pane: the caption row with
// the two buttons under the type chips, the query area below it and the
// variables area at the bottom.
const (
	gqlButtonsY    = 244
	gqlSchemaX     = 595
	gqlLoadSchemaX = 705
	gqlQueryX      = 500
	gqlQueryY      = 400
	gqlVarsY       = 700
)

// openGraphQLBody shows the Body pane of the open request with a GraphQL
// body pointed at url.
func openGraphQLBody(t *testing.T, url string) *App {
	a := openBodyPane(t)
	t.Cleanup(a.WaitWrites)
	a.cur().Req.URL = url
	a.setBodyType(a.cur(), request.BodyGraphQL)
	a.Step(core.Input{})
	return a
}

// Typing into the query area makes it the request's body.
func TestTypingAGraphQLQuerySetsTheBody(t *testing.T) {
	a := openGraphQLBody(t, "http://127.0.0.1:1/graphql")
	click(a, gqlQueryX, gqlQueryY)
	a.typeText("{ viewer }")
	if got := a.cur().Req.Body; !strings.Contains(got, "{ viewer }") {
		t.Fatalf("body %q, want the typed query in it", got)
	}
}

// Typing into the variables area makes them the request's variables.
func TestTypingGraphQLVariablesSetsThem(t *testing.T) {
	a := openGraphQLBody(t, "http://127.0.0.1:1/graphql")
	click(a, gqlQueryX, gqlVarsY)
	a.typeText(`{"a": 1}`)
	if got := a.cur().Req.GraphQLVars; got != `{"a": 1}` {
		t.Fatalf("variables %q, want the typed text", got)
	}
}

// The Schema button opens a menu of the loaded operations, and picking
// one puts its starter into the query and variables.
func TestTheSchemaButtonOffersTheOperations(t *testing.T) {
	a := openGraphQLBody(t, "http://127.0.0.1:1/graphql")
	a.cur().schema = pinnedSchema
	a.Step(core.Input{})
	click(a, gqlSchemaX, gqlButtonsY)
	want := []string{pinnedSchema.Operations[0].Label, pinnedSchema.Operations[1].Label}
	if widget.PopupMenu(a.ctx) == nil || !slices.Equal(menuLabels(a), want) {
		t.Fatalf("menu = %v, want %q", widget.PopupMenu(a.ctx), want)
	}
	widget.PopupMenu(a.ctx).Items[0].Run()
	tab := a.cur()
	if tab.Req.Body != pinnedSchema.Operations[0].Query || tab.Req.GraphQLVars != pinnedSchema.Operations[0].Variables {
		t.Fatalf("body %q variables %q, want the first starter", tab.Req.Body, tab.Req.GraphQLVars)
	}
}

// Load schema sends the introspection query to the tab's URL and keeps
// the schema of the answer.
func TestTheLoadSchemaButtonLoadsTheSchema(t *testing.T) {
	a := openGraphQLBody(t, introspectionServer(t).URL)
	click(a, gqlLoadSchemaX, gqlButtonsY)
	tab := a.cur()
	waitForResponse(t, a, tab)
	if tab.schema == nil || len(tab.schema.Operations) != 1 {
		t.Fatalf("schema = %+v, want the server's one operation", tab.schema)
	}
}

// Right-clicking either area opens the text menu.
func TestTheGraphQLAreasHaveTheTextMenu(t *testing.T) {
	for _, y := range []int{gqlQueryY, gqlVarsY} {
		a := openGraphQLBody(t, "http://127.0.0.1:1/graphql")
		rightClick(a, gqlQueryX, y)
		if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
			t.Fatalf("at y=%d menu = %v", y, widget.PopupMenu(a.ctx))
		}
	}
}
