package logic

import (
	"os"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

const (
	petstorePath = "../../../feature/collections/importers/openapi/testdata/petstore.json"
	postmanPath  = "../../../feature/collections/importers/postman/testdata/collection.json"
	curlLine     = `curl -X POST 'https://api.test/v1/users?x=1' -H 'Content-Type: application/json' -d '{"a":1}'`
)

func TestImportersRefuseTextOfAnotherKindWithTheirOwnWording(t *testing.T) {
	tests := []struct {
		kind        ImportKind
		text        string
		refusal     string
		skippedWhat string
	}{
		{ImportCurl, "hello", "Not a cURL command: ", ""},
		{ImportOpenAPI, "{}", "Not an OpenAPI document: ", "parts could not be read"},
		{ImportPostman, "[]", "Not a Postman collection: ", "things could not be imported"},
	}
	for _, test := range tests {
		importer := Importers[test.kind]
		if importer.Refusal != test.refusal || importer.SkippedWhat != test.skippedWhat {
			t.Errorf("kind %d: refusal %q, skipped %q; want %q, %q", test.kind, importer.Refusal, importer.SkippedWhat, test.refusal, test.skippedWhat)
		}
		if importer.Parse == nil {
			t.Fatalf("kind %d has no parser", test.kind)
		}
		if _, err := importer.Parse(test.text); err == nil {
			t.Errorf("kind %d read %q without an error", test.kind, test.text)
		}
	}
}

func TestImportersParseARealInputIntoTheRightVariant(t *testing.T) {
	t.Run("cURL is one request named after its call", func(t *testing.T) {
		got, err := Importers[ImportCurl].Parse(curlLine)
		imported, ok := got.(ImportedRequest)
		if err != nil || !ok || imported.Request.Name != "POST /v1/users" || imported.Request.Body != `{"a":1}` || len(got.Skipped()) != 0 {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
	t.Run("OpenAPI is a folder with the base URL and what was left out", func(t *testing.T) {
		got, err := Importers[ImportOpenAPI].Parse(readFixture(t, petstorePath))
		imported, ok := got.(ImportedFolder)
		if err != nil || !ok || imported.Folder == nil || len(imported.Vars) != 1 || imported.Vars[0].Name != "baseUrl" {
			t.Fatalf("got %+v, %v", got, err)
		}
		if !reflect.DeepEqual(got.Skipped(), imported.Problems) {
			t.Errorf("Skipped() = %q, want the problems %q", got.Skipped(), imported.Problems)
		}
	})
	t.Run("Postman is a folder with its variables", func(t *testing.T) {
		got, err := Importers[ImportPostman].Parse(readFixture(t, postmanPath))
		imported, ok := got.(ImportedFolder)
		if err != nil || !ok || imported.Folder == nil || imported.Folder.Name != "Echo API" || len(imported.Vars) != 3 {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
}

func TestAttachingARequestAddsItUnderAFreeNameAndShowsIt(t *testing.T) {
	tree := New()
	dir := &request.Folder{Name: "root", Requests: []*request.Request{{Name: "GET /users"}}}
	name, show := ImportedRequest{Request: request.Request{Name: "GET /users", Method: "GET"}}.Attach(tree, dir, &request.Collection{})
	if name != "GET /users 2" || show == nil || show.Name != name || show.Method != "GET" {
		t.Fatalf("name %q, show %+v", name, show)
	}
	if len(dir.Requests) != 2 || dir.Requests[1] != show {
		t.Error("the request is not at the end of the folder")
	}
	if !tree.Expanded()[dir] {
		t.Error("the folder it went into is closed")
	}
}

func TestAttachingAFolderMergesVariablesAndShowsItsFirstRequest(t *testing.T) {
	first := &request.Request{Name: "first"}
	build := func() *request.Folder {
		return &request.Folder{Name: "API", Requests: []*request.Request{first}, Folders: []*request.Folder{{Name: "sub", Requests: []*request.Request{{Name: "deep"}}}}}
	}
	vars := []request.KV{{Name: "baseUrl", Value: "theirs", Enabled: true}, {Name: "token", Value: "t", Enabled: true}}
	t.Run("a free name, new variables only, the folder opened", func(t *testing.T) {
		tree := New()
		dir := &request.Folder{Name: "root", Folders: []*request.Folder{{Name: "API"}}}
		owner := &request.Collection{Vars: []request.KV{{Name: "baseUrl", Value: "mine", Enabled: true}}}
		imported := build()
		name, show := ImportedFolder{Folder: imported, Vars: vars}.Attach(tree, dir, owner)
		if name != "API 2" || len(dir.Folders) != 2 || dir.Folders[1] != imported {
			t.Fatalf("name %q, folders %d", name, len(dir.Folders))
		}
		want := []request.KV{{Name: "baseUrl", Value: "mine", Enabled: true}, {Name: "token", Value: "t", Enabled: true}}
		if !reflect.DeepEqual(owner.Vars, want) {
			t.Errorf("variables %+v, want %+v", owner.Vars, want)
		}
		if show != first || !tree.Expanded()[dir] || !tree.Expanded()[imported] || !tree.Expanded()[imported.Folders[0]] {
			t.Errorf("show %v, expanded %v", show, tree.Expanded())
		}
	})
	t.Run("no owner adds no variables", func(t *testing.T) {
		dir := &request.Folder{Name: "root"}
		name, _ := ImportedFolder{Folder: build(), Vars: vars}.Attach(New(), dir, nil)
		if name != "API" || len(dir.Folders) != 1 {
			t.Errorf("name %q, folders %d", name, len(dir.Folders))
		}
	})
	t.Run("an empty folder shows nothing", func(t *testing.T) {
		dir := &request.Folder{Name: "root"}
		_, show := ImportedFolder{Folder: &request.Folder{Name: "empty"}}.Attach(New(), dir, &request.Collection{})
		if show != nil {
			t.Errorf("show %v, want nil", show)
		}
	})
}
