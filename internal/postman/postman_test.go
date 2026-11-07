package postman

import (
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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

func find(f *model.Folder, name string) *model.Request {
	for _, r := range f.Requests {
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
	if len(res.Vars) != 1 || res.Vars[0].Name != "baseUrl" || !res.Vars[0].Enabled {
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
	if get.Auth.Type != model.AuthBearer || get.Auth.Token != "{{token}}" {
		t.Fatalf("auth %+v", get.Auth)
	}
	post := find(load(t).Folder.Folders[0], "POST json")
	if post.BodyType != model.BodyJSON || !strings.Contains(post.Body, `"a": 1`) || post.URL != "{{baseUrl}}/post" {
		t.Fatalf("%+v", post)
	}
}

func TestBodyModesAndOtherAuthTypes(t *testing.T) {
	root := load(t).Folder
	form := find(root, "Form")
	if form.BodyType != model.BodyForm || len(form.Form) != 2 || form.Form[0].Value != "ann" || form.Form[1].Enabled {
		t.Fatalf("%+v", form)
	}
	up := find(root, "Upload")
	if up.BodyType != model.BodyMultipart || up.Form[1].Value != "@/tmp/a.txt" || up.Auth.Type != model.AuthBasic || up.Auth.User != "u" || up.Auth.Password != "p" {
		t.Fatalf("%+v", up)
	}
	key := find(root, "Keyed")
	if key.Auth.Type != model.AuthAPIKey || key.Auth.KeyName != "X-Api-Key" || key.Auth.KeyValue != "secret" || key.Auth.KeyIn != model.KeyInHeader {
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

func TestExportThenParseKeepsWhatPostmanCanHold(t *testing.T) {
	first := load(t)
	root := first.Folder
	root.AddRequest(model.Request{Name: "Socket", Protocol: "websocket", URL: "wss://x"})
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
		if b == nil || a.Method != b.Method || a.URL != b.URL || a.BodyType != b.BodyType || a.Auth != b.Auth || len(a.Form) != len(b.Form) {
			t.Errorf("%s changed:\n before %+v\n after  %+v", name, a, b)
		}
	}
	get := find(again.Folder.Folders[0], "GET with query")
	if get.URL != "{{baseUrl}}/get?foo=bar&baz=1" || len(get.Params) != 3 || get.Auth.Token != "{{token}}" || len(get.Headers) != 2 || get.Headers[1].Enabled {
		t.Fatalf("%+v", get)
	}
	if len(again.Vars) != 1 || again.Vars[0].Name != "baseUrl" {
		t.Fatalf("vars %+v", again.Vars)
	}
	if find(again.Folder, "Socket") != nil {
		t.Fatal("a WebSocket request has no Postman form")
	}
}
