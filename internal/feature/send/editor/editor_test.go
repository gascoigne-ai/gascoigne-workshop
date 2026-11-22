package editor

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestUpdateURLTypedRebuildsParamsKeepingDescriptionsAndDisabledRows(t *testing.T) {
	req := request.Request{
		URL: "http://x/s?q=old",
		Params: []request.KV{
			{Name: "q", Value: "old", Description: "repo name", Enabled: true},
			{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
		},
	}

	got := Update(req, URLTyped{Text: "http://x/s?q=new&page=2"})

	want := []request.KV{
		{Name: "q", Value: "new", Description: "repo name", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
		{Name: "sort", Value: "stars", Description: "metric", Enabled: false},
	}
	if got.URL != "http://x/s?q=new&page=2" || !reflect.DeepEqual(got.Params, want) {
		t.Fatalf("URL %q params %+v, want the typed URL and %+v", got.URL, got.Params, want)
	}
}

func TestUpdateParamsChangedRewritesTheQueryFromTheEnabledRows(t *testing.T) {
	req := request.Request{
		URL: "http://x/s?q=a",
		Params: []request.KV{
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
		protocol   request.Protocol
		wantMethod string
	}{
		{"rest with no method gets GET", "", "rest", "GET"},
		{"rest after a gRPC method gets GET", "pkg.Service/Call", "rest", "GET"},
		{"rest keeps its own method", "POST", "rest", "POST"},
		{"another protocol keeps the method", "pkg.Service/Call", "grpc", "pkg.Service/Call"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Update(request.Request{Method: c.method}, ProtocolChosen{Protocol: c.protocol})
			if got.Protocol != c.protocol || got.Method != c.wantMethod {
				t.Fatalf("protocol %q method %q, want %q and %q", got.Protocol, got.Method, c.protocol, c.wantMethod)
			}
		})
	}
}

func TestUpdateMethodChosenAndTimeoutSet(t *testing.T) {
	req := request.Request{Method: "GET", TimeoutSec: 5}

	got := Update(Update(req, MethodChosen{Method: "PUT"}), TimeoutSet{Seconds: 0})

	if got.Method != "PUT" || got.TimeoutSec != 0 {
		t.Fatalf("method %q timeout %d, want PUT and 0", got.Method, got.TimeoutSec)
	}
}

func TestUpdateDoesNotChangeTheRequestItWasGiven(t *testing.T) {
	req := request.Request{URL: "http://x?a=1", Params: []request.KV{{Name: "a", Value: "1", Enabled: true}}}
	before := req.Clone()

	Update(req, URLTyped{Text: "http://x?b=2"})
	Update(req, MethodChosen{Method: "POST"})

	if !reflect.DeepEqual(req, before) {
		t.Fatalf("the input request changed: %+v", req)
	}
}

func TestUpdateBodyAndVariablesTyped(t *testing.T) {
	got := Update(Update(request.Request{}, BodyTyped{Text: `{"a":1}`}), VarsTyped{Text: `{"id":3}`})

	if got.Body != `{"a":1}` || got.GraphQLVars != `{"id":3}` {
		t.Fatalf("body %q variables %q", got.Body, got.GraphQLVars)
	}
}

func TestUpdateBodyTypeChosenKeepsTheRightTextForEachType(t *testing.T) {
	cases := []struct {
		name       string
		req        request.Request
		msg        BodyTypeChosen
		wantType   request.BodyType
		wantBody   string
		wantMethod string
	}{
		{"to a file body the body is the path",
			request.Request{BodyType: request.BodyJSON, Body: "{}", Method: "POST"},
			BodyTypeChosen{Type: request.BodyFile, Text: "{}", FilePath: "/tmp/a.bin"},
			request.BodyFile, "/tmp/a.bin", "POST"},
		{"from a file body the text comes back",
			request.Request{BodyType: request.BodyFile, Body: "/tmp/a.bin", Method: "POST"},
			BodyTypeChosen{Type: request.BodyText, Text: "hello", FilePath: "/tmp/a.bin"},
			request.BodyText, "hello", "POST"},
		{"between text types the body stays",
			request.Request{BodyType: request.BodyText, Body: "hello", Method: "POST"},
			BodyTypeChosen{Type: request.BodyJSON, Text: "hello"},
			request.BodyJSON, "hello", "POST"},
		{"to GraphQL a GET request becomes a POST",
			request.Request{Protocol: "rest", Method: "GET", BodyType: request.BodyNone},
			BodyTypeChosen{Type: request.BodyGraphQL, Text: "{ me }"},
			request.BodyGraphQL, "{ me }", "POST"},
		{"to GraphQL a request with no method becomes a POST",
			request.Request{Protocol: "rest", Method: "", BodyType: request.BodyNone},
			BodyTypeChosen{Type: request.BodyGraphQL, Text: "{ me }"},
			request.BodyGraphQL, "{ me }", "POST"},
		{"to GraphQL another method is kept",
			request.Request{Protocol: "rest", Method: "PUT", BodyType: request.BodyNone},
			BodyTypeChosen{Type: request.BodyGraphQL, Text: "{ me }"},
			request.BodyGraphQL, "{ me }", "PUT"},
		{"to GraphQL on another protocol the method is kept",
			request.Request{Protocol: "grpc", Method: "GET", BodyType: request.BodyNone},
			BodyTypeChosen{Type: request.BodyGraphQL, Text: "{ me }"},
			request.BodyGraphQL, "{ me }", "GET"},
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
	got := Update(request.Request{}, OperationChosen{Query: "query { me }", Variables: `{"a":1}`})

	if got.Body != "query { me }" || got.GraphQLVars != `{"a":1}` {
		t.Fatalf("body %q variables %q", got.Body, got.GraphQLVars)
	}
}

func TestUpdateAuthTypeAndFieldsAreSetOneAtATime(t *testing.T) {
	cases := []struct {
		field AuthField
		read  func(request.Auth) string
	}{
		{"token", func(auth request.Auth) string { return auth.Token }},
		{"user", func(auth request.Auth) string { return auth.User }},
		{"password", func(auth request.Auth) string { return auth.Password }},
		{"keyname", func(auth request.Auth) string { return auth.KeyName }},
		{"keyvalue", func(auth request.Auth) string { return auth.KeyValue }},
	}
	for _, c := range cases {
		t.Run(string(c.field), func(t *testing.T) {
			got := Update(request.Request{}, AuthFieldSet{Field: c.field, Value: "v"})

			if c.read(got.Auth) != "v" || AuthValue(got.Auth, c.field) != "v" {
				t.Fatalf("%s not set: %+v", c.field, got.Auth)
			}
			other := request.Auth{Token: "t", User: "u", Password: "p", KeyName: "n", KeyValue: "kv", KeyIn: request.KeyInQuery}
			after := Update(request.Request{Auth: other}, AuthFieldSet{Field: c.field, Value: "v"}).Auth
			if before := AuthValue(other, c.field); before == "v" || after == other {
				t.Fatalf("setting %s changed nothing, or changed the others: %+v", c.field, after)
			}
		})
	}

	got := Update(request.Request{}, AuthTypeChosen{Type: request.AuthBearer})
	if got.Auth.Type != request.AuthBearer {
		t.Fatalf("auth type %q", got.Auth.Type)
	}
}

func TestUpdateAuthKeyInChosenSetsWhereTheKeyGoes(t *testing.T) {
	before := request.Request{Auth: request.Auth{KeyName: "k", KeyIn: request.KeyInHeader}}

	got := Update(before, AuthKeyInChosen{In: request.KeyInQuery})

	if got.Auth.KeyIn != request.KeyInQuery || got.Auth.KeyName != "k" {
		t.Fatalf("auth = %+v", got.Auth)
	}
	if before.Auth.KeyIn != request.KeyInHeader {
		t.Fatalf("Update changed its input: %+v", before.Auth)
	}
}

func TestUpdateAnUnknownAuthFieldIsABug(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an unknown auth field must panic, not be dropped")
		}
	}()

	Update(request.Request{}, AuthFieldSet{Field: "pasword", Value: "x"})
}

func TestUpdateTestsTyped(t *testing.T) {
	if got := Update(request.Request{}, TestsTyped{Text: "status == 200"}); got.Tests != "status == 200" {
		t.Fatalf("tests %q", got.Tests)
	}
}

func TestUpdateTogglesFlipTheirOwnFlagOnly(t *testing.T) {
	flipped := Update(request.Request{}, NoRedirectsToggled{})
	if !flipped.NoRedirects || flipped.InsecureTLS {
		t.Fatalf("NoRedirectsToggled: %+v", flipped)
	}
	if again := Update(flipped, NoRedirectsToggled{}); again.NoRedirects {
		t.Fatal("a second NoRedirectsToggled must turn it off")
	}

	flipped = Update(request.Request{}, InsecureTLSToggled{})
	if !flipped.InsecureTLS || flipped.NoRedirects {
		t.Fatalf("InsecureTLSToggled: %+v", flipped)
	}
	if again := Update(flipped, InsecureTLSToggled{}); again.InsecureTLS {
		t.Fatal("a second InsecureTLSToggled must turn it off")
	}
}

func TestUpdateProtoPathsAreAddedSetAndRemovedPerList(t *testing.T) {
	cases := []struct {
		list ProtoList
		read func(request.ProtoSource) []string
	}{
		{ProtoFiles, func(source request.ProtoSource) []string { return source.Files }},
		{ImportPaths, func(source request.ProtoSource) []string { return source.ImportPaths }},
	}
	for _, c := range cases {
		t.Run(string(c.list), func(t *testing.T) {
			original := request.Request{Protocol: "grpc"}
			req := Update(original, ProtoPathAdded{List: c.list, Path: "/protos/a.proto"})
			req = Update(req, ProtoPathAdded{List: c.list, Path: ""})
			req = Update(req, ProtoPathSet{List: c.list, Index: 1, Path: "/protos/b.proto"})
			if got := c.read(req.Proto); !reflect.DeepEqual(got, []string{"/protos/a.proto", "/protos/b.proto"}) {
				t.Fatalf("after adding and setting: %v", got)
			}
			before := req
			req = Update(req, ProtoPathRemoved{List: c.list, Index: 0})
			if got := c.read(req.Proto); !reflect.DeepEqual(got, []string{"/protos/b.proto"}) {
				t.Fatalf("after removing the first: %v", got)
			}
			if got := c.read(before.Proto); !reflect.DeepEqual(got, []string{"/protos/a.proto", "/protos/b.proto"}) {
				t.Fatalf("removing changed the request it was given: %v", got)
			}
			if c.read(original.Proto) != nil {
				t.Fatalf("adding changed the request it was given: %+v", original.Proto)
			}
			if got := Update(req, ProtoPathRemoved{List: c.list, Index: 0}); c.read(got.Proto) != nil {
				t.Fatalf("an emptied list is %v, want none, so it is not saved", c.read(got.Proto))
			}
		})
	}
}

func TestUpdateProtoPathOutOfRangeChangesNothing(t *testing.T) {
	req := request.Request{Proto: request.ProtoSource{Files: []string{"/a.proto"}}}
	for _, msg := range []Msg{ProtoPathSet{List: ProtoFiles, Index: 3, Path: "x"}, ProtoPathRemoved{List: ProtoFiles, Index: -1}} {
		if got := Update(req, msg); !reflect.DeepEqual(got.Proto, req.Proto) {
			t.Errorf("%T: %+v", msg, got.Proto)
		}
	}
}

func TestUpdateReflectionChosenDropsTheFilesAndImportPaths(t *testing.T) {
	req := request.Request{Proto: request.ProtoSource{Files: []string{"/a.proto"}, ImportPaths: []string{"/protos"}}}
	got := Update(req, ReflectionChosen{})
	if !reflect.DeepEqual(got.Proto, request.ProtoSource{}) || got.Proto.Kind() != request.SourceReflection {
		t.Fatalf("proto source %+v", got.Proto)
	}
}
