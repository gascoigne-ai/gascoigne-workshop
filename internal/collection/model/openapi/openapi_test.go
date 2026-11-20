package openapi

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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

func find(folder *model.Folder, name string) *model.Request {
	for _, r := range folder.Requests {
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
		"openapi: 3.0.0":                 "no operations",
		"a: [1, 2\n":                     "not readable as YAML",
		"{broken":                        "not JSON",
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

func TestTheYAMLFormOfTheSameDescriptionGivesTheSameCollection(t *testing.T) {
	fromJSON := petstore(t)
	data, err := os.ReadFile("testdata/petstore.yaml")
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if fromYAML.BaseURL != fromJSON.BaseURL || !reflect.DeepEqual(dump(fromYAML.Folder), dump(fromJSON.Folder)) {
		t.Fatalf("YAML:\n%v\nJSON:\n%v", dump(fromYAML.Folder), dump(fromJSON.Folder))
	}
}

// dump lists a folder as "path: METHOD URL body" lines, to compare two imports.
func dump(folder *model.Folder) []string {
	var out []string
	var walk func(folder *model.Folder, prefix string)
	walk = func(folder *model.Folder, prefix string) {
		for _, r := range folder.Requests {
			out = append(out, prefix+r.Name+": "+r.Method+" "+r.URL+" "+r.Body)
		}
		for _, s := range folder.Folders {
			walk(s, prefix+s.Name+"/")
		}
	}
	walk(folder, "")
	return out
}

// A vendor extension at the paths level ("x-internal": true) failed the
// whole document, and one malformed operation aborted the import, so the
// good operations were lost with it.
func TestImportKeepsGoodOperationsAroundOddOnes(t *testing.T) {
	res, err := Parse([]byte(`{
	  "openapi": "3.0.0", "info": {"title": "Odd"},
	  "paths": {
	    "x-internal": true,
	    "/ok": {"get": {"summary": "Fine"}, "x-note": "ignored"},
	    "/broken": {"post": {"summary": "Bad", "parameters": "nope"}, "get": {"summary": "Also fine"}},
	    "/ref": {"$ref": "#/components/pathItems/Other"}
	  }}`))
	if err != nil {
		t.Fatal(err)
	}
	if find(res.Folder, "Fine") == nil || find(res.Folder, "Also fine") == nil {
		t.Errorf("good operations missing: %+v", res.Folder.Requests)
	}
	if find(res.Folder, "Bad") != nil {
		t.Error("the malformed operation was imported")
	}
	if len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0], "POST /broken") {
		t.Errorf("Skipped = %q, want the broken operation named", res.Skipped)
	}
}

// Pins that a schema whose many properties all refer back to it stays
// cheap: the depth limit bounds the expansion (width cubed, about 0.2 s
// for 150 properties), so it needs no cycle detection.
func TestASelfReferencingSchemaStaysCheapToExpand(t *testing.T) {
	var props []string
	for i := range 150 {
		props = append(props, fmt.Sprintf(`"p%d": {"$ref": "#/components/schemas/Node"}`, i))
	}
	src := `{"openapi": "3.0.0", "info": {"title": "T"},
	  "paths": {"/n": {"post": {"requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Node"}}}}}}},
	  "components": {"schemas": {"Node": {"type": "object", "properties": {` + strings.Join(props, ",") + `}}}}}`
	done := make(chan Result, 1)
	go func() {
		res, err := Parse([]byte(src))
		if err != nil {
			t.Error(err)
		}
		done <- res
	}()
	select {
	case res := <-done:
		if r := find(res.Folder, "POST /n"); r == nil || r.BodyType != model.BodyJSON {
			t.Errorf("request %+v, want a JSON body", r)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Parse is still expanding the schema after 3s")
	}
}

// A requestBody written as {"$ref": "#/components/requestBodies/..."},
// which specs use for bodies shared by several operations, gave the
// request no body at all, silently.
func TestARequestBodyRefIsResolved(t *testing.T) {
	res, err := Parse([]byte(`{
	  "openapi": "3.0.0", "info": {"title": "T"},
	  "paths": {"/pets": {"post": {"summary": "Add", "requestBody": {"$ref": "#/components/requestBodies/PetBody"}}}},
	  "components": {
	    "requestBodies": {"PetBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Pet"}}}}},
	    "schemas": {"Pet": {"type": "object", "required": ["name"], "properties": {"name": {"type": "string"}}}}
	  }}`))
	if err != nil {
		t.Fatal(err)
	}
	req := find(res.Folder, "Add")
	if req == nil || req.BodyType != model.BodyJSON || !strings.Contains(req.Body, `"name"`) {
		t.Fatalf("request %+v, want a JSON body with the Pet skeleton", req)
	}
}

// FuzzParse: any document text, JSON or YAML.
func FuzzParse(fuzz *testing.F) {
	if data, err := os.ReadFile("testdata/petstore.json"); err == nil {
		fuzz.Add(data)
	}
	fuzz.Add([]byte("openapi: 3.0.0\ninfo:\n  title: T\npaths:\n  /a:\n    get:\n      summary: A\n"))
	fuzz.Add([]byte(`{"openapi":"3.0.0","paths":{"/a":{"get":[]}}}`))
	fuzz.Add([]byte(`{"openapi":"3.0.0","paths":{"/a":{"post":{"requestBody":{"content":{"application/json":{"schema":` +
		strings.Repeat(`{"type":"object","required":["a"],"properties":{"a":`, 60) + `{"$ref":"#/components/schemas/S"}` + strings.Repeat(`}}`, 60) +
		`}}}}}},"components":{"schemas":{"S":{"$ref":"#/components/schemas/S"}}}}`))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}

// Shared parameters of a path that failed to decode were dropped without
// a word, so its requests came in without them and nobody knew why.
func TestUnreadableSharedParametersAreReportedAsSkipped(t *testing.T) {
	res, err := Parse([]byte(`{
	  "openapi": "3.0.0", "info": {"title": "Odd"},
	  "paths": {"/pets": {"parameters": "nope", "get": {"summary": "List"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if find(res.Folder, "List") == nil {
		t.Error("the operation was lost with its shared parameters")
	}
	if len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0], "/pets") || !strings.Contains(res.Skipped[0], "parameters") {
		t.Errorf("Skipped = %q, want the unreadable parameters of /pets named", res.Skipped)
	}
}

// What a parameter and a body get as sample values: pinned before the
// schemas became typed.
func TestSampleValuesComeFromExamplesDefaultsEnumsAndSchemas(t *testing.T) {
	res, err := Parse([]byte(`{
	  "openapi": "3.0.0", "info": {"title": "Samples"},
	  "paths": {
	    "/a": {"get": {"summary": "A", "parameters": [
	      {"name": "ex", "in": "query", "example": "e1", "schema": {"type": "string", "default": "d"}},
	      {"name": "num", "in": "query", "example": 5},
	      {"name": "def", "in": "query", "schema": {"type": "integer", "default": 10}},
	      {"name": "must", "in": "query", "required": true, "schema": {"enum": ["x", "y"]}},
	      {"name": "may", "in": "query", "schema": {"enum": ["x", "y"]}}
	    ]}},
	    "/b": {"post": {"summary": "B", "requestBody": {"content": {"application/json": {"example": {"z": 1, "a": "x"}}}}}},
	    "/c": {"post": {"summary": "C", "requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Outer"}}}}}},
	    "/d": {"post": {"summary": "D", "requestBody": {"content": {"application/json": {"schema": {"type": "object", "required": ["far"], "properties": {"far": {"$ref": "http://example.com/s.json"}}}}}}}}
	  },
	  "components": {"schemas": {
	    "Outer": {"type": "object", "required": ["inner", "name"], "properties": {"inner": {"$ref": "#/components/schemas/Inner"}, "name": {"type": "string"}}},
	    "Inner": {"type": "object", "required": ["id"], "properties": {"id": {"type": "integer"}}}
	  }}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	a := find(res.Folder, "A")
	values := map[string]string{}
	for _, query := range a.Params {
		values[query.Name] = query.Value
	}
	want := map[string]string{"ex": "e1", "num": "5", "def": "10", "must": "x", "may": ""}
	if !reflect.DeepEqual(values, want) {
		t.Errorf("parameter samples %v, want %v", values, want)
	}
	bodies := map[string]string{
		"B": "{\n  \"a\": \"x\",\n  \"z\": 1\n}",
		"C": "{\n  \"inner\": {\n    \"id\": 0\n  },\n  \"name\": \"\"\n}",
		"D": "{\n  \"far\": null\n}",
	}
	for name, want := range bodies {
		if got := find(res.Folder, name).Body; got != want {
			t.Errorf("body of %s = %q, want %q", name, got, want)
		}
	}
}

// A path whose value is not an object was dropped without a word. Keys
// that start with "x-" are extensions, not paths, and stay quiet.
func TestDroppedPathItemsAreWarnedAbout(t *testing.T) {
	res, err := Parse([]byte(`{
	  "openapi": "3.0.0", "info": {"title": "Odd"},
	  "paths": {"/ok": {"get": {"summary": "List"}}, "/bad": "nope", "x-internal": "note"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "/bad") {
		t.Errorf("Warnings = %q, want only /bad named", res.Warnings)
	}
	if len(res.Skipped) != 0 {
		t.Errorf("Skipped = %q, want none: a dropped path is a warning", res.Skipped)
	}
}
