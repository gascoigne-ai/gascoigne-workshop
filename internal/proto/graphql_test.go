package proto

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestGraphQLBodyCarriesQueryAndVariablesAsJSON(t *testing.T) {
	var got map[string]any
	var ct string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ct = r.Header.Get("Content-Type")
		json.Unmarshal(b, &got)
		w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()
	req := model.Request{Method: "POST", URL: srv.URL, BodyType: model.BodyGraphQL,
		Body: "query($n: Int!) { repos(first: $n) { name } } # <tag>", GraphQLVars: `{"n": 3}`}
	if resp, err := (REST{}).Send(context.Background(), req); err != nil || resp.Err != "" {
		t.Fatalf("%+v %v", resp, err)
	}
	if ct != "application/json" || got["query"] != req.Body {
		t.Fatalf("type %q payload %v", ct, got)
	}
	vars, _ := got["variables"].(map[string]any)
	if vars["n"] != float64(3) {
		t.Fatalf("variables %v", got["variables"])
	}
	req.GraphQLVars = "  "
	body, _, _ := encodeBody(req)
	if strings.Contains(string(body), "variables") || !strings.Contains(string(body), "<tag>") {
		t.Fatalf("no variables, and HTML characters stay as typed: %s", body)
	}
}

func TestGraphQLBodyErrorsAreExplained(t *testing.T) {
	if _, _, err := encodeBody(model.Request{BodyType: model.BodyGraphQL}); err == nil || !strings.Contains(err.Error(), "write a GraphQL query") {
		t.Fatalf("err = %v", err)
	}
	_, _, err := encodeBody(model.Request{BodyType: model.BodyGraphQL, Body: "{ a }", GraphQLVars: "[1]"})
	if err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Fatalf("err = %v", err)
	}
}
