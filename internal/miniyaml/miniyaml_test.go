package miniyaml

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func parse(t *testing.T, src string) any {
	t.Helper()
	v, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("%v\n%s", err, src)
	}
	return v
}

func TestAnAPIDescriptionStyleDocument(t *testing.T) {
	src := `---
openapi: 3.0.2   # the version
info:
  title: "Swagger: Petstore"
  version: '1.0.19'
servers:
  - url: /api/v3
paths:
  /pet/{petId}:
    get:
      tags: [pet, store]
      parameters:
      - name: petId
        in: path
        required: true
        schema: {type: integer, format: int64}
      - $ref: '#/components/parameters/trace'
      description: |
        Returns a single pet.
        Second line.
      summary: >-
        Find pet
        by ID
      deprecated: false
      example: ~
  /empty: {}
`
	got := parse(t, src)
	var want any
	json.Unmarshal([]byte(`{
	  "openapi": "3.0.2",
	  "info": {"title": "Swagger: Petstore", "version": "1.0.19"},
	  "servers": [{"url": "/api/v3"}],
	  "paths": {
	    "/pet/{petId}": {"get": {
	      "tags": ["pet", "store"],
	      "parameters": [
	        {"name": "petId", "in": "path", "required": true, "schema": {"type": "integer", "format": "int64"}},
	        {"$ref": "#/components/parameters/trace"}
	      ],
	      "description": "Returns a single pet.\nSecond line.\n",
	      "summary": "Find pet by ID",
	      "deprecated": false,
	      "example": null
	    }},
	    "/empty": {}
	  }
	}`), &want)
	// 3.0.2 is not a number: it stays text, like the JSON above
	if !reflect.DeepEqual(got, want) {
		g, _ := json.MarshalIndent(got, "", " ")
		w, _ := json.MarshalIndent(want, "", " ")
		t.Fatalf("got\n%s\nwant\n%s", g, w)
	}
}

func TestScalarsFollowTheCoreSchema(t *testing.T) {
	got := parse(t, "a: 1\nb: 1.5\nc: true\nd: null\ne: ~\nf: 007\ng: 1e3\nh: \"1\"\ni: yes\nj: 3.0.2\nk: -4\nl: http://x.test/a:b\nm: 'it''s'\nn: \"tab\\there\"\no:\n").(map[string]any)
	want := map[string]any{"a": 1.0, "b": 1.5, "c": true, "d": nil, "e": nil, "f": "007", "g": 1000.0, "h": "1", "i": "yes", "j": "3.0.2", "k": -4.0, "l": "http://x.test/a:b", "m": "it's", "n": "tab\there", "o": nil}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestSequencesAtTheKeysIndentAndNested(t *testing.T) {
	got := parse(t, "list:\n- a\n- b\nnested:\n  - - 1\n    - 2\n  - [3, 4]\nmaps:\n  - x: 1\n    y: 2\n  - z: 3\nempty:\n  -\n  - k: v\n")
	want := map[string]any{
		"list":   []any{"a", "b"},
		"nested": []any{[]any{1.0, 2.0}, []any{3.0, 4.0}},
		"maps":   []any{map[string]any{"x": 1.0, "y": 2.0}, map[string]any{"z": 3.0}},
		"empty":  []any{nil, map[string]any{"k": "v"}},
	}
	if !reflect.DeepEqual(got, want) {
		g, _ := json.Marshal(got)
		w, _ := json.Marshal(want)
		t.Fatalf("got %s\nwant %s", g, w)
	}
}

func TestCommentsQuotesAndFlowCollections(t *testing.T) {
	got := parse(t, "a: \"x # not a comment\" # a comment\nb: [ 'p, q', \"r\", s ]\nc: {k: v, 'quoted key': 2, list: [1, 2]}\n# whole line\nd: e#f\n").(map[string]any)
	if got["a"] != "x # not a comment" || got["d"] != "e#f" {
		t.Fatalf("%v", got)
	}
	if !reflect.DeepEqual(got["b"], []any{"p, q", "r", "s"}) {
		t.Fatalf("b %v", got["b"])
	}
	if !reflect.DeepEqual(got["c"], map[string]any{"k": "v", "quoted key": 2.0, "list": []any{1.0, 2.0}}) {
		t.Fatalf("c %v", got["c"])
	}
}

func TestBlockScalarsKeepTheirLinesOrFold(t *testing.T) {
	got := parse(t, "lit: |\n  a\n   b\n\n  c\nfold: >\n  one\n  two\n\n  three\nstrip: |-\n  x\nnext: 1\n").(map[string]any)
	if got["lit"] != "a\n b\n\nc\n" || got["fold"] != "one two\nthree\n" || got["strip"] != "x" || got["next"] != 1.0 {
		t.Fatalf("%q", got)
	}
}

func TestUnsupportedAndBrokenInputIsReportedWithALine(t *testing.T) {
	for src, want := range map[string]string{
		"":                     "empty",
		"a: &x 1\n":            "line 1: anchors",
		"a: *x\n":              "line 1: anchors",
		"a: !!str 1\n":         "line 1: anchors",
		"a: 1\na: 2\n":         "line 2: the key \"a\" appears twice",
		"a:\n  b: 1\n c: 2\n":  "line 3",
		"\ta: 1\n":             "tabs",
		"a: [1, 2\n":           "not closed",
		"a: \"open\n":          "unterminated",
		"just text\nmore: 1\n": "line 2",
		"a: 1\n  b: 2\n":       "line 2",
	} {
		_, err := Parse([]byte(src))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) = %v, want %q", src, err, want)
		}
	}
}
