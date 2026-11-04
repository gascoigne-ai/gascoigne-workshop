package openapi

import (
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func petstore(t *testing.T) Result {
	t.Helper()
	data, err := os.ReadFile("testdata/petstore.json")
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

func TestPetstoreBecomesFoldersPerTagAndRequestsPerOperation(t *testing.T) {
	res := petstore(t)
	if res.Folder.Name != "Swagger Petstore - OpenAPI 3.0" || res.BaseURL != "/api/v3" {
		t.Fatalf("%q %q", res.Folder.Name, res.BaseURL)
	}
	var tags []string
	for _, f := range res.Folder.Folders {
		tags = append(tags, f.Name)
	}
	if strings.Join(tags, ",") != "pet,user" {
		t.Fatalf("folders %v", tags)
	}
	if len(res.Folder.Requests) != 1 || res.Folder.Requests[0].Name != "Returns pet inventories by status" {
		t.Fatalf("an untagged operation goes in the root: %+v", res.Folder.Requests)
	}
	pet := res.Folder.Folders[0]
	if len(pet.Requests) != 4 {
		t.Fatalf("%d pet requests", len(pet.Requests))
	}
}

func TestOperationsKeepMethodURLParametersAndBody(t *testing.T) {
	pet := petstore(t).Folder.Folders[0]

	add := find(pet, "Add a new pet to the store")
	if add == nil || add.Method != "POST" || add.URL != "{{baseUrl}}/pet" || add.BodyType != model.BodyJSON {
		t.Fatalf("%+v", add)
	}
	if want := "{\n  \"name\": \"\",\n  \"photoUrls\": []\n}"; add.Body != want {
		t.Fatalf("body %q, want %q", add.Body, want)
	}

	status := find(pet, "Finds Pets by status")
	if status.URL != "{{baseUrl}}/pet/findByStatus?status=available" || len(status.Params) != 1 || !status.Params[0].Enabled || status.Params[0].Value != "available" {
		t.Fatalf("%+v", status)
	}

	byID := find(pet, "Find pet by ID")
	if byID.URL != "{{baseUrl}}/pet/{{petId}}" {
		t.Fatalf("path parameters become variables: %q", byID.URL)
	}
	if len(byID.Headers) != 1 || byID.Headers[0].Name != "X-Trace" || !byID.Headers[0].Enabled {
		t.Fatalf("a $ref parameter is followed: %+v", byID.Headers)
	}

	del := find(pet, "deletePet")
	if del == nil || del.Method != "DELETE" || len(del.Headers) != 1 || del.Headers[0].Enabled {
		t.Fatalf("an optional header is listed but off: %+v", del)
	}
}

func TestParseRejectsWhatIsNotOpenAPI(t *testing.T) {
	for in, want := range map[string]string{
		"":                               "not JSON",
		"openapi: 3.0.0":                 "not JSON",
		`{"swagger":"2.0","paths":{}}`:   "not an OpenAPI 3",
		`{"openapi":"3.0.0","paths":{}}`: "no operations",
		`{"openapi":"3.0.0","paths":{"/a":{"get":[]}}}`: "GET /a",
	} {
		if _, err := Parse([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) = %v, want %q", in, err, want)
		}
	}
}

func TestEqualNamesInOneFolderStayDistinct(t *testing.T) {
	res, err := Parse([]byte(`{"openapi":"3.0.0","paths":{"/a":{"get":{"summary":"Same"},"post":{"summary":"Same"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Folder.Requests) != 2 || res.Folder.Requests[0].Name == res.Folder.Requests[1].Name {
		t.Fatalf("%+v", res.Folder.Requests)
	}
}
