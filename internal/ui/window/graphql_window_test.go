package window

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The GraphQL body in the test window's Body pane: the caption row with
// the two buttons under the type chips, the query area below it and the
// variables area at the bottom.
const (
	gqlButtonsY    = 244
	gqlLoadSchemaX = 705
	gqlQueryX      = 500
	gqlQueryY      = 400
	gqlVarsY       = 700
)

// openGraphQLBody shows the Body pane of the open request with a GraphQL
// body pointed at url.
func openGraphQLBody(t *testing.T, url string) *Component {
	a := openBodyPane(t)
	t.Cleanup(a.WaitWrites)
	a.cur().Req.URL = url
	a.editor.Body.ChooseType(a.cur().Tab, request.BodyGraphQL)
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

// The variables area is a text field of the window: while it has the
// keyboard, plain-key shortcuts are held back.
func TestFocusInTheGraphQLVariablesCountsAsTyping(t *testing.T) {
	a := openGraphQLBody(t, "http://127.0.0.1:1/graphql")
	if a.typing() {
		t.Fatal("nothing is focused at the start")
	}
	click(a, gqlQueryX, gqlVarsY)
	if !a.typing() {
		t.Fatal("the variables area has the keyboard but textFocused is false")
	}
}

// Load schema sends the introspection query to the tab's URL and the
// answer, which the window hands to the component, makes the Schema
// button show.
func TestTheLoadSchemaButtonLoadsTheSchema(t *testing.T) {
	a := openGraphQLBody(t, introspectionServer(t).URL)
	click(a, gqlLoadSchemaX, gqlButtonsY)
	waitForResponse(t, a, a.cur())
	a.Step(core.Input{})
	if dump := view.Dump(a.Tree()); !strings.Contains(dump, "Schema (1)") {
		t.Fatalf("no schema button for the server's one operation:\n%s", dump)
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

// introspectionServer answers every request with a small schema (shaped
// from the GraphQL specification, not captured from a server).
func introspectionServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte(`{"data":{"__schema":{"queryType":{"name":"Query"},"mutationType":null,"types":[
		 {"kind":"OBJECT","name":"Query","fields":[
		  {"name":"repo","args":[{"name":"name","type":{"kind":"NON_NULL","name":null,"ofType":{"kind":"SCALAR","name":"String","ofType":null}}}],"type":{"kind":"OBJECT","name":"Repo","ofType":null}}]},
		 {"kind":"OBJECT","name":"Repo","fields":[{"name":"name","args":[],"type":{"kind":"SCALAR","name":"String","ofType":null}}]}]}}}`))
	}))
	t.Cleanup(server.Close)
	return server
}
