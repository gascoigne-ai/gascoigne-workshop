package postman

import (
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

func load(t *testing.T) Result {
	t.Helper()
	data, err := os.ReadFile("testdata/collection.json")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func find(folder *request.Folder, name string) *request.Request {
	for _, r := range folder.Requests {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func TestCollectionKeepsFoldersRequestsAndVariables(t *testing.T) {
	res := load(t)
	if res.Folder.Name != "Echo API" || len(res.Folder.Folders) != 1 || res.Folder.Folders[0].Name != "Requests" || len(res.Folder.Requests) != 3 {
		t.Fatalf("%q folders %d requests %d", res.Folder.Name, len(res.Folder.Folders), len(res.Folder.Requests))
	}
	if len(res.Vars) != 3 || res.Vars[0].Name != "baseUrl" || !res.Vars[0].Enabled || res.Vars[1].Name != "token" || res.Vars[2].Name != "apiKey" {
		t.Fatalf("vars %+v", res.Vars)
	}
}

func TestRequestsCarryMethodURLHeadersQueryAndAuth(t *testing.T) {
	get := find(load(t).Folder.Folders[0], "GET with query")
	if get == nil || get.Method != "GET" || get.URL != "{{baseUrl}}/get?foo=bar&baz=1" {
		t.Fatalf("%+v", get)
	}
	if len(get.Params) != 3 || get.Params[2].Name != "skip" || get.Params[2].Enabled {
		t.Fatalf("a disabled query row stays, off: %+v", get.Params)
	}
	if len(get.Headers) != 2 || get.Headers[1].Enabled {
		t.Fatalf("headers %+v", get.Headers)
	}
	if get.Auth.Type != request.AuthBearer || get.Auth.Token != "{{token}}" {
		t.Fatalf("auth %+v", get.Auth)
	}
	post := find(load(t).Folder.Folders[0], "POST json")
	if post.BodyType != request.BodyJSON || !strings.Contains(post.Body, `"a": 1`) || post.URL != "{{baseUrl}}/post" {
		t.Fatalf("%+v", post)
	}
}

func TestBodyModesAndOtherAuthTypes(t *testing.T) {
	root := load(t).Folder
	form := find(root, "Form")
	if form.BodyType != request.BodyForm || len(form.Form) != 2 || form.Form[0].Value != "ann" || form.Form[1].Enabled {
		t.Fatalf("%+v", form)
	}
	up := find(root, "Upload")
	if up.BodyType != request.BodyMultipart || up.Form[1].Value != "@a.txt" || up.Auth.Type != request.AuthBasic || up.Auth.User != "u" || up.Auth.Password != "p" {
		t.Fatalf("%+v", up)
	}
	key := find(root, "Keyed")
	if key.Auth.Type != request.AuthAPIKey || key.Auth.KeyName != "X-Api-Key" || key.Auth.KeyValue != "{{apiKey}}" || key.Auth.KeyIn != request.KeyInHeader {
		t.Fatalf("%+v", key.Auth)
	}
}

func TestParseRejectsWhatIsNotAPostmanCollection(t *testing.T) {
	for in, want := range map[string]string{
		"":   "not JSON",
		"{}": "not a Postman collection",
		`{"info":{"name":"x","schema":"https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},"item":[]}`: "no requests",
	} {
		if _, err := Parse([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) = %v, want %q", in, err, want)
		}
	}
}

// withoutLiteralCredentials is auth as an export leaves it: a literal
// token, password or key value is not written; a {{variable}} is.
func withoutLiteralCredentials(auth request.Auth) request.Auth {
	auth.Token, auth.Password, auth.KeyValue = shareable(auth.Token), shareable(auth.Password), shareable(auth.KeyValue)
	return auth
}

func TestExportThenParseKeepsWhatPostmanCanHold(t *testing.T) {
	first := load(t)
	root := first.Folder
	root.AddRequest(request.Request{Name: "Socket", Protocol: "websocket", URL: "wss://x"})
	data, err := Export(root, first.Vars)
	if err != nil {
		t.Fatal(err)
	}
	if Skipped(root) != 1 {
		t.Fatalf("skipped %d", Skipped(root))
	}
	again, err := Parse(data)
	if err != nil {
		t.Fatalf("%v\n%s", err, data)
	}
	if again.Folder.Name != root.Name || len(again.Folder.Folders) != 1 || len(again.Folder.Requests) != 3 {
		t.Fatalf("structure: %d folders, %d requests", len(again.Folder.Folders), len(again.Folder.Requests))
	}
	for _, name := range []string{"Form", "Upload", "Keyed"} {
		a, b := find(root, name), find(again.Folder, name)
		if b == nil || a.Method != b.Method || a.URL != b.URL || a.BodyType != b.BodyType || withoutLiteralCredentials(a.Auth) != b.Auth || len(a.Form) != len(b.Form) {
			t.Errorf("%s changed:\n before %+v\n after  %+v", name, a, b)
		}
	}
	get := find(again.Folder.Folders[0], "GET with query")
	if get.URL != "{{baseUrl}}/get?foo=bar&baz=1" || len(get.Params) != 3 || get.Auth.Token != "{{token}}" || len(get.Headers) != 2 || get.Headers[1].Enabled {
		t.Fatalf("%+v", get)
	}
	if len(again.Vars) != 3 || again.Vars[0].Name != "baseUrl" {
		t.Fatalf("vars %+v", again.Vars)
	}
	if find(again.Folder, "Socket") != nil {
		t.Fatal("a WebSocket request has no Postman form")
	}
}

// A numeric or boolean variable value ("value": 8080) and a v2.0 request
// written as a bare URL string failed the whole import with a JSON type
// error, so one odd item cost the user the collection.
func TestImportSurvivesNumericValuesAndAStringRequest(t *testing.T) {
	res, err := Parse([]byte(`{
	  "info": {"name": "Odd", "schema": "https://schema.getpostman.com/json/collection/v2.0.0/collection.json"},
	  "variable": [{"key": "port", "value": 8080}, {"key": "debug", "value": true}, {"key": "none", "value": null}],
	  "item": [
	    {"name": "Short", "request": "https://api.test/short"},
	    {"name": "Long", "request": {"method": "GET", "url": "https://api.test/long", "header": [{"key": "X-N", "value": 5}]}}
	  ]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"port": "8080", "debug": "true", "none": ""}
	for _, v := range res.Vars {
		if v.Value != want[v.Name] {
			t.Errorf("var %s = %q, want %q", v.Name, v.Value, want[v.Name])
		}
	}
	short, long := find(res.Folder, "Short"), find(res.Folder, "Long")
	if short == nil || short.URL != "https://api.test/short" || short.Method != "GET" {
		t.Errorf("short request %+v", short)
	}
	if long == nil || len(long.Headers) != 1 || long.Headers[0].Value != "5" {
		t.Errorf("long request %+v", long)
	}
}

// Auth types and body modes the app has no equivalent for were dropped
// without a word, so the user sent unauthenticated requests wondering why.
func TestImportListsWhatItCannotCarryOver(t *testing.T) {
	res, err := Parse([]byte(`{
	  "info": {"name": "Skips", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
	  "item": [
	    {"name": "OAuth call", "request": {"method": "GET", "url": "https://api.test/a", "auth": {"type": "oauth2"}}},
	    {"name": "Upload", "request": {"method": "POST", "url": "https://api.test/b", "body": {"mode": "file"}}},
	    {"name": "Plain", "request": {"method": "GET", "url": "https://api.test/c", "auth": {"type": "noauth"}}}
	  ]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 2 || !strings.Contains(res.Skipped[0], "OAuth call") || !strings.Contains(res.Skipped[0], "oauth2") ||
		!strings.Contains(res.Skipped[1], "Upload") || !strings.Contains(res.Skipped[1], "file") {
		t.Fatalf("Skipped = %q, want the oauth2 auth and the file body, not noauth", res.Skipped)
	}
}

// FuzzParse: any collection text.
func FuzzParse(fuzz *testing.F) {
	if data, err := os.ReadFile("testdata/collection.json"); err == nil {
		fuzz.Add(data)
	}
	fuzz.Add([]byte(`{"info":{"name":"x","schema":"https://schema.getpostman.com/json/collection/v2.0.0/collection.json"},"item":[{"name":"a","request":"https://x.test"}]}`))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}

// Export wrote the value of Secret variables and every request's token,
// password and API key into a file made to be shared. They are now left
// empty; the structure (names, user names, key names) stays.
func TestExportLeavesSecretsOut(t *testing.T) {
	root := &request.Folder{Name: "API", Requests: []*request.Request{
		{Name: "Bearer", Protocol: "rest", Method: "GET", URL: "u", Auth: request.Auth{Type: request.AuthBearer, Token: "tok-secret"}},
		{Name: "Basic", Protocol: "rest", Method: "GET", URL: "u", Auth: request.Auth{Type: request.AuthBasic, User: "alice", Password: "pw-secret"}},
		{Name: "Key", Protocol: "rest", Method: "GET", URL: "u", Auth: request.Auth{Type: request.AuthAPIKey, KeyName: "X-Key", KeyValue: "key-secret", KeyIn: request.KeyInHeader}},
	}}
	vars := []request.KV{{Name: "token", Value: "var-secret", Enabled: true, Secret: true}, {Name: "host", Value: "visible.test", Enabled: true}}
	data, err := Export(root, vars)
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"tok-secret", "pw-secret", "key-secret", "var-secret"} {
		if strings.Contains(string(data), leaked) {
			t.Errorf("the export holds %q", leaked)
		}
	}
	for _, kept := range []string{"alice", "X-Key", "visible.test", `"token"`} {
		if !strings.Contains(string(data), kept) {
			t.Errorf("the export lost %q", kept)
		}
	}
}

// Postman holds HTTP requests only: every other protocol is left out of
// the export and counted as skipped.
func TestPostmanExportSkipsEveryProtocolButREST(t *testing.T) {
	root := &request.Folder{Name: "c"}
	for _, protocol := range []request.Protocol{"rest", "websocket", "mcp", "sse", "grpc"} {
		root.AddRequest(request.Request{Name: string(protocol), Protocol: protocol, Method: "GET", URL: "u"})
	}
	if got := Skipped(root); got != 4 {
		t.Errorf("Skipped = %d, want 4", got)
	}
	data, err := Export(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Folder.Requests) != 1 || again.Folder.Requests[0].Name != "rest" {
		t.Fatalf("exported %d requests, want only rest", len(again.Folder.Requests))
	}
}
