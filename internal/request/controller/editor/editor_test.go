package editor

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestUpdateURLTypedRebuildsParamsKeepingDescriptionsAndDisabledRows(t *testing.T) {
	req := model.Request{
		URL: "http://x/s?q=old",
		Params: []model.KV{
			{Name: "q", Value: "old", Description: "repo name", Enabled: true},
			{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
		},
	}

	got := Update(req, URLTyped{Text: "http://x/s?q=new&page=2"})

	want := []model.KV{
		{Name: "q", Value: "new", Description: "repo name", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
		{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
	}
	if got.URL != "http://x/s?q=new&page=2" || !reflect.DeepEqual(got.Params, want) {
		t.Fatalf("URL %q params %+v, want the typed URL and %+v", got.URL, got.Params, want)
	}
}

func TestUpdateParamsChangedRewritesTheQueryFromTheEnabledRows(t *testing.T) {
	req := model.Request{
		URL: "http://x/s?q=a",
		Params: []model.KV{
			{Name: "q", Value: "a", Enabled: true},
			{Name: "sort", Value: "stars", Enabled: false},
			{Name: "page", Value: "2", Enabled: true},
		},
	}

	got := Update(req, ParamsChanged{})

	if want := "http://x/s?q=a&page=2"; got.URL != want {
		t.Fatalf("URL %q, want %q", got.URL, want)
	}
}

func TestUpdateProtocolChosenFixesAMethodThatDoesNotFitRest(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		protocol   string
		wantMethod string
	}{
		{"rest with no method gets GET", "", "rest", "GET"},
		{"rest after a gRPC method gets GET", "pkg.Service/Call", "rest", "GET"},
		{"rest keeps its own method", "POST", "rest", "POST"},
		{"another protocol keeps the method", "pkg.Service/Call", "grpc", "pkg.Service/Call"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Update(model.Request{Method: c.method}, ProtocolChosen{Protocol: c.protocol})
			if got.Protocol != c.protocol || got.Method != c.wantMethod {
				t.Fatalf("protocol %q method %q, want %q and %q", got.Protocol, got.Method, c.protocol, c.wantMethod)
			}
		})
	}
}

func TestUpdateMethodChosenAndTimeoutSet(t *testing.T) {
	req := model.Request{Method: "GET", TimeoutSec: 5}

	got := Update(Update(req, MethodChosen{Method: "PUT"}), TimeoutSet{Seconds: 0})

	if got.Method != "PUT" || got.TimeoutSec != 0 {
		t.Fatalf("method %q timeout %d, want PUT and 0", got.Method, got.TimeoutSec)
	}
}

func TestUpdateDoesNotChangeTheRequestItWasGiven(t *testing.T) {
	req := model.Request{URL: "http://x?a=1", Params: []model.KV{{Name: "a", Value: "1", Enabled: true}}}
	before := req.Clone()

	Update(req, URLTyped{Text: "http://x?b=2"})
	Update(req, MethodChosen{Method: "POST"})

	if !reflect.DeepEqual(req, before) {
		t.Fatalf("the input request changed: %+v", req)
	}
}

func TestUpdateBodyAndVariablesTyped(t *testing.T) {
	got := Update(Update(model.Request{}, BodyTyped{Text: `{"a":1}`}), VarsTyped{Text: `{"id":3}`})

	if got.Body != `{"a":1}` || got.GraphQLVars != `{"id":3}` {
		t.Fatalf("body %q variables %q", got.Body, got.GraphQLVars)
	}
}

func TestUpdateBodyTypeChosenKeepsTheRightTextForEachType(t *testing.T) {
	cases := []struct {
		name       string
		req        model.Request
		msg        BodyTypeChosen
		wantType   string
		wantBody   string
		wantMethod string
	}{
		{"to a file body the body is the path",
			model.Request{BodyType: model.BodyJSON, Body: "{}", Method: "POST"},
			BodyTypeChosen{Type: model.BodyFile, Text: "{}", FilePath: "/tmp/a.bin"},
			model.BodyFile, "/tmp/a.bin", "POST"},
		{"from a file body the text comes back",
			model.Request{BodyType: model.BodyFile, Body: "/tmp/a.bin", Method: "POST"},
			BodyTypeChosen{Type: model.BodyText, Text: "hello", FilePath: "/tmp/a.bin"},
			model.BodyText, "hello", "POST"},
		{"between text types the body stays",
			model.Request{BodyType: model.BodyText, Body: "hello", Method: "POST"},
			BodyTypeChosen{Type: model.BodyJSON, Text: "hello"},
			model.BodyJSON, "hello", "POST"},
		{"to GraphQL a GET request becomes a POST",
			model.Request{Protocol: "rest", Method: "GET", BodyType: model.BodyNone},
			BodyTypeChosen{Type: model.BodyGraphQL, Text: "{ me }"},
			model.BodyGraphQL, "{ me }", "POST"},
		{"to GraphQL a request with no method becomes a POST",
			model.Request{Protocol: "rest", Method: "", BodyType: model.BodyNone},
			BodyTypeChosen{Type: model.BodyGraphQL, Text: "{ me }"},
			model.BodyGraphQL, "{ me }", "POST"},
		{"to GraphQL another method is kept",
			model.Request{Protocol: "rest", Method: "PUT", BodyType: model.BodyNone},
			BodyTypeChosen{Type: model.BodyGraphQL, Text: "{ me }"},
			model.BodyGraphQL, "{ me }", "PUT"},
		{"to GraphQL on another protocol the method is kept",
			model.Request{Protocol: "grpc", Method: "GET", BodyType: model.BodyNone},
			BodyTypeChosen{Type: model.BodyGraphQL, Text: "{ me }"},
			model.BodyGraphQL, "{ me }", "GET"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Update(c.req, c.msg)
			if got.BodyType != c.wantType || got.Body != c.wantBody || got.Method != c.wantMethod {
				t.Fatalf("type %q body %q method %q, want %q %q %q", got.BodyType, got.Body, got.Method, c.wantType, c.wantBody, c.wantMethod)
			}
		})
	}
}

func TestUpdateOperationChosenFillsQueryAndVariables(t *testing.T) {
	got := Update(model.Request{}, OperationChosen{Query: "query { me }", Variables: `{"a":1}`})

	if got.Body != "query { me }" || got.GraphQLVars != `{"a":1}` {
		t.Fatalf("body %q variables %q", got.Body, got.GraphQLVars)
	}
}

func TestUpdateAuthTypeAndFieldsAreSetOneAtATime(t *testing.T) {
	cases := []struct {
		field string
		read  func(model.Auth) string
	}{
		{"token", func(auth model.Auth) string { return auth.Token }},
		{"user", func(auth model.Auth) string { return auth.User }},
		{"password", func(auth model.Auth) string { return auth.Password }},
		{"keyname", func(auth model.Auth) string { return auth.KeyName }},
		{"keyvalue", func(auth model.Auth) string { return auth.KeyValue }},
		{"keyin", func(auth model.Auth) string { return auth.KeyIn }},
	}
	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			got := Update(model.Request{}, AuthFieldSet{Field: c.field, Value: "v"})

			if c.read(got.Auth) != "v" || AuthValue(got.Auth, c.field) != "v" {
				t.Fatalf("%s not set: %+v", c.field, got.Auth)
			}
			other := model.Auth{Token: "t", User: "u", Password: "p", KeyName: "n", KeyValue: "kv", KeyIn: "i"}
			after := Update(model.Request{Auth: other}, AuthFieldSet{Field: c.field, Value: "v"}).Auth
			if before := AuthValue(other, c.field); before == "v" || after == other {
				t.Fatalf("setting %s changed nothing, or changed the others: %+v", c.field, after)
			}
		})
	}

	got := Update(model.Request{}, AuthTypeChosen{Type: model.AuthBearer})
	if got.Auth.Type != model.AuthBearer {
		t.Fatalf("auth type %q", got.Auth.Type)
	}
}

func TestUpdateAnUnknownAuthFieldIsABug(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an unknown auth field must panic, not be dropped")
		}
	}()

	Update(model.Request{}, AuthFieldSet{Field: "pasword", Value: "x"})
}

func TestUpdateTestsTyped(t *testing.T) {
	if got := Update(model.Request{}, TestsTyped{Text: "status == 200"}); got.Tests != "status == 200" {
		t.Fatalf("tests %q", got.Tests)
	}
}
