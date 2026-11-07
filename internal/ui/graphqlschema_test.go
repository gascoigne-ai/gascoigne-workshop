package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// introspectionServer answers every request with a small schema (shaped
// from the GraphQL specification, not captured from a server).
func introspectionServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"data":{"__schema":{"queryType":{"name":"Query"},"mutationType":null,"types":[
		 {"kind":"OBJECT","name":"Query","fields":[
		  {"name":"repo","args":[{"name":"name","type":{"kind":"NON_NULL","name":null,"ofType":{"kind":"SCALAR","name":"String","ofType":null}}}],"type":{"kind":"OBJECT","name":"Repo","ofType":null}}]},
		 {"kind":"OBJECT","name":"Repo","fields":[{"name":"name","args":[],"type":{"kind":"SCALAR","name":"String","ofType":null}}]}]}}}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func graphQLTab(t *testing.T, a *App, url string) *reqTab {
	t.Helper()
	tab := a.cur()
	tab.req.URL, tab.req.Method = url, "POST"
	a.setBodyType(tab, model.BodyGraphQL)
	return tab
}

func TestLoadingTheSchemaOffersItsOperationsInAMenu(t *testing.T) {
	a, _ := newTestApp(t)
	tab := graphQLTab(t, a, introspectionServer(t).URL)
	if strings.Contains(view.Dump(a.graphQLBody(tab)), "Schema (") {
		t.Fatal("a schema button before the schema is loaded")
	}
	a.loadSchema(tab)
	waitForResponse(t, a, tab)

	if !strings.Contains(view.Dump(a.graphQLBody(tab)), "Schema (1)") {
		t.Fatalf("no schema button:\n%s", view.Dump(a.graphQLBody(tab)))
	}
	items := a.schemaMenu(tab)
	if len(items) != 1 || items[0].label != "query repo(name: String!): Repo" {
		t.Fatalf("menu = %+v", items)
	}
}

func TestPickingAnOperationFillsTheQueryAndVariablesOfTheTab(t *testing.T) {
	a, _ := newTestApp(t)
	tab := graphQLTab(t, a, introspectionServer(t).URL)
	a.loadSchema(tab)
	waitForResponse(t, a, tab)

	a.schemaMenu(tab)[0].run()
	wantQuery := "query repo($name: String!) {\n  repo(name: $name) {\n    name\n  }\n}"
	if tab.body.Text != wantQuery || tab.req.Body != wantQuery {
		t.Fatalf("query = %q (request %q), want %q", tab.body.Text, tab.req.Body, wantQuery)
	}
	if want := "{\n  \"name\": \"\"\n}"; tab.gqlVars.Text != want || tab.req.GraphQLVars != want {
		t.Fatalf("variables = %q (request %q), want %q", tab.gqlVars.Text, tab.req.GraphQLVars, want)
	}
}
