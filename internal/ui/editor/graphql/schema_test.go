package graphql

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// answer is the response of a server to the introspection query.
func answer(body string) *request.Response { return &request.Response{Status: 200, Body: []byte(body)} }

// dumped is the editor of the open tab as text.
func (world *world) dumped() string { return Dump(world.view()) }

func TestLoadingTheSchemaOffersItsOperationsInAMenu(t *testing.T) {
	world := newWorld(t)
	if strings.Contains(world.dumped(), "Schema (") {
		t.Fatal("a schema button before the schema is loaded")
	}
	world.click("Load schema")

	if len(world.fake.sends) != 1 {
		t.Fatalf("%d sends, want the introspection query", len(world.fake.sends))
	}
	sent := world.fake.sends[0]
	if sent.tab != world.tab || sent.req.Method != "POST" || sent.req.BodyType != request.BodyGraphQL ||
		sent.req.Body != send.IntrospectionQuery || sent.req.GraphQLVars != "" || sent.req.URL != world.tab.Req.URL {
		t.Fatalf("sent %+v for tab %p", sent.req, sent.tab)
	}
	if world.tab.Req.Body == send.IntrospectionQuery {
		t.Fatal("loading the schema must not touch the tab's own query")
	}

	world.pane.Remember(world.tab, answer(introspection))
	if !strings.Contains(world.dumped(), "Schema (1)") {
		t.Fatalf("no schema button:\n%s", world.dumped())
	}
	world.click("Schema (1)")
	if len(world.fake.menus) != 1 || len(world.fake.menus[0].items) != 1 ||
		world.fake.menus[0].items[0].Label != "query repo(name: String!): Repo" {
		t.Fatalf("menus = %+v", world.fake.menus)
	}
}

func TestPickingAnOperationFillsTheQueryAndVariablesOfTheTab(t *testing.T) {
	world := newWorld(t)
	world.pane.Remember(world.tab, answer(introspection))
	world.click("Schema (1)")

	world.fake.menus[0].items[0].Run()
	wantQuery := "query repo($name: String!) {\n  repo(name: $name) {\n    name\n  }\n}"
	if query := world.fake.Area(world.tab).Text; query != wantQuery || world.tab.Req.Body != wantQuery {
		t.Fatalf("query = %q (request %q), want %q", query, world.tab.Req.Body, wantQuery)
	}
	vars := world.pane.state(world.tab).vars.Text
	if want := "{\n  \"name\": \"\"\n}"; vars != want || world.tab.Req.GraphQLVars != want {
		t.Fatalf("variables = %q (request %q), want %q", vars, world.tab.Req.GraphQLVars, want)
	}
}

// The Schema button opens a menu of the loaded operations, and picking
// one puts its starter into the query and variables.
func TestTheSchemaButtonOffersTheOperations(t *testing.T) {
	world := newWorld(t)
	world.pane.state(world.tab).schema = pinnedSchema
	world.click("Schema (2)")
	if len(world.fake.menus) != 1 {
		t.Fatalf("%d menus opened, want 1", len(world.fake.menus))
	}
	items := world.fake.menus[0].items
	if len(items) != 2 || items[0].Label != pinnedSchema.Operations[0].Label || items[1].Label != pinnedSchema.Operations[1].Label {
		t.Fatalf("menu = %+v, want the two operations", items)
	}
	items[0].Run()
	if world.tab.Req.Body != pinnedSchema.Operations[0].Query || world.tab.Req.GraphQLVars != pinnedSchema.Operations[0].Variables {
		t.Fatalf("body %q variables %q, want the first starter", world.tab.Req.Body, world.tab.Req.GraphQLVars)
	}
}

// An answer is a schema only for a tab whose body is GraphQL, and only when
// it parses; any other answer leaves what the tab knows.
func TestRememberKeepsTheSchemaOfIntrospectionAnswersOnly(t *testing.T) {
	t.Run("a tab whose body is not GraphQL ignores it", func(t *testing.T) {
		world := newWorld(t)
		world.tab.Req.BodyType = request.BodyJSON
		world.pane.Remember(world.tab, answer(introspection))
		if _, made := world.pane.states[world.tab]; made {
			t.Fatal("a state was made for an answer that was ignored")
		}
	})
	t.Run("an answer that is no schema keeps the old one", func(t *testing.T) {
		world := newWorld(t)
		world.pane.state(world.tab).schema = pinnedSchema
		world.pane.Remember(world.tab, answer(`{"data":{"viewer":{"login":"gio"}}}`))
		if world.pane.state(world.tab).schema != pinnedSchema {
			t.Fatal("the old schema was replaced")
		}
	})
	t.Run("an answer for a tab with no schema yet makes no state", func(t *testing.T) {
		world := newWorld(t)
		world.pane.Remember(world.tab, answer(`not json`))
		if _, made := world.pane.states[world.tab]; made {
			t.Fatal("a state was made for an answer that is no schema")
		}
	})
}
