package model

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func schemaOf(t *testing.T, text string) map[string]any {
	t.Helper()
	var schema map[string]any
	if err := json.Unmarshal([]byte(text), &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func TestToolFieldsListOnePerPropertyRequiredFirst(t *testing.T) {
	schema := schemaOf(t, `{"type":"object","required":["query"],"properties":{
		"limit":{"type":"integer","description":"How many","default":10},
		"query":{"type":"string","description":"What to find"},
		"safe":{"type":"boolean"},
		"sort":{"type":"string","enum":["asc","desc"]},
		"range":{"type":"number"},
		"filters":{"type":"object"},
		"tags":{"type":"array","items":{"type":"string"}},
		"free":{}}}`)
	got := ToolFields(schema)
	type brief struct {
		Name     string
		Kind     FieldKind
		Required bool
	}
	var names []brief
	for _, field := range got {
		names = append(names, brief{field.Name, field.Kind, field.Required})
	}
	want := []brief{
		{"query", FieldString, true},
		{"filters", FieldJSON, false}, {"free", FieldJSON, false},
		{"limit", FieldInteger, false}, {"range", FieldNumber, false},
		{"safe", FieldBoolean, false}, {"sort", FieldEnum, false}, {"tags", FieldJSON, false},
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("got %+v\nwant %+v", names, want)
	}
	if got[0].Description != "What to find" {
		t.Errorf("description = %q", got[0].Description)
	}
	if limit := got[3]; limit.Initial() != "10" {
		t.Errorf("the default is the initial text, got %q", limit.Initial())
	}
}

func TestToolFieldsOfASchemaWithoutPropertiesIsEmpty(t *testing.T) {
	for _, text := range []string{`{}`, `{"type":"object"}`, `{"type":"string"}`} {
		if got := ToolFields(schemaOf(t, text)); len(got) != 0 {
			t.Errorf("%s: %+v", text, got)
		}
	}
	if ToolFields(nil) != nil {
		t.Error("no schema, no fields")
	}
}

func TestArgumentsBuildsTypedJSONFromTheFormValues(t *testing.T) {
	fields := ToolFields(schemaOf(t, `{"type":"object","required":["query"],"properties":{
		"query":{"type":"string"},
		"limit":{"type":"integer"},
		"ratio":{"type":"number"},
		"safe":{"type":"boolean"},
		"sort":{"type":"string","enum":["asc","desc"]},
		"rank":{"type":"integer","enum":[1,2,3]},
		"filters":{"type":"object"}}}`))
	tests := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"only the required one", map[string]string{"query": "go"}, `{"query":"go"}`},
		{"every type", map[string]string{
			"query": "a \"b\"", "limit": "5", "ratio": "0.5", "safe": "true", "sort": "desc", "rank": "2",
			"filters": "{ \"a\": [1, 2] }",
		}, `{"query":"a \"b\"","filters":{"a":[1,2]},"limit":5,"rank":2,"ratio":0.5,"safe":true,"sort":"desc"}`},
		{"empty optional values are left out", map[string]string{"query": "x", "limit": "", "sort": "", "filters": "  "}, `{"query":"x"}`},
		{"an unset boolean is left out, a set false is sent", map[string]string{"query": "x", "safe": "false"}, `{"query":"x","safe":false}`},
		{"a required empty string is still sent", map[string]string{"query": ""}, `{"query":""}`},
		{"a string keeps its spaces", map[string]string{"query": "  a b "}, `{"query":"  a b "}`},
	}
	for _, tt := range tests {
		got, err := Arguments(fields, tt.values)
		if err != nil || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.name, got, err, tt.want)
		}
	}
}

func TestArgumentsRefusesWhatDoesNotFitAndNamesTheField(t *testing.T) {
	fields := ToolFields(schemaOf(t, `{"type":"object","required":["id"],"properties":{
		"id":{"type":"integer"},
		"ratio":{"type":"number"},
		"sort":{"type":"string","enum":["asc","desc"]},
		"filters":{"type":"object"}}}`))
	tests := []struct {
		name    string
		values  map[string]string
		wantErr string
	}{
		{"a required number is missing", map[string]string{}, "id is required"},
		{"an integer with a fraction", map[string]string{"id": "1.5"}, "id must be a whole number"},
		{"text for a number", map[string]string{"id": "1", "ratio": "lots"}, "ratio must be a number"},
		{"a value outside the enum", map[string]string{"id": "1", "sort": "up"}, "sort must be one of asc, desc"},
		{"broken JSON", map[string]string{"id": "1", "filters": `{"a":`}, "filters is not valid JSON"},
	}
	for _, tt := range tests {
		if got, err := Arguments(fields, tt.values); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%s: got %q, %v; want an error mentioning %q", tt.name, got, err, tt.wantErr)
		}
	}
}
