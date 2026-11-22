package request

import (
	"testing"
)

func TestDecodeToolsReadsAToolsListResult(t *testing.T) {
	msg := `{"tools":[{"name":"echo","description":"Echo it","inputSchema":{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}},{"name":"ping"}]}`
	tools, _ := DecodeTools([]byte(msg))
	if len(tools) != 2 || tools[0].Name != "echo" || tools[0].Description != "Echo it" || tools[1].Name != "ping" {
		t.Fatalf("%+v", tools)
	}
	for _, bad := range []string{"", "not json", `{}`, `{"tools":[{"description":"no name"}]}`} {
		if got, _ := DecodeTools([]byte(bad)); got != nil {
			t.Errorf("DecodeTools(%q) = %+v", bad, got)
		}
	}
}

func TestSkeletonFitsTheSchema(t *testing.T) {
	tests := []struct {
		name, schema, want string
	}{
		{"required scalars only", `{"type":"object","properties":{"b":{"type":"boolean"},"a":{"type":"string"},"n":{"type":"integer"},"opt":{"type":"string"}},"required":["a","n","b"]}`, `{"a":"","b":false,"n":0}`},
		{"enum takes the first value", `{"type":"object","properties":{"mode":{"type":"string","enum":["fast","slow"]}},"required":["mode"]}`, `{"mode":"fast"}`},
		{"a default is used and kept even when optional", `{"type":"object","properties":{"limit":{"type":"integer","default":10}}}`, `{"limit":10}`},
		{"nested objects", `{"type":"object","properties":{"opts":{"type":"object","properties":{"deep":{"type":"number"}},"required":["deep"]}},"required":["opts"]}`, `{"opts":{"deep":0}}`},
		{"arrays", `{"type":"object","properties":{"tags":{"type":"array"}},"required":["tags"]}`, `{"tags":[]}`},
		{"type lists skip null", `{"type":"object","properties":{"x":{"type":["null","string"]}},"required":["x"]}`, `{"x":""}`},
		{"no properties", `{"type":"object"}`, `{}`},
		{"no schema at all", `{}`, `null`},
	}
	for _, tt := range tests {
		tools, _ := DecodeTools([]byte(`{"tools":[{"name":"t","inputSchema":` + tt.schema + `}]}`))
		if len(tools) != 1 {
			t.Fatalf("%s: %v", tt.name, tools)
		}
		if got := Skeleton(tools[0].Schema); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.name, got, tt.want)
		}
	}
	if got := Skeleton(JSONSchema{Type: SchemaString, Default: JSONValue(`"<b>"`)}); got != `"<b>"` {
		t.Errorf("HTML characters must not be escaped in a skeleton: %s", got)
	}
}

func TestDecodeResourcesAndPrompts(t *testing.T) {
	res, _ := DecodeResources([]byte(`{"resources":[{"uri":"file:///a.txt","name":"A"},{"name":"no uri"}]}`))
	if len(res) != 1 || res[0].URI != "file:///a.txt" || res[0].Name != "A" {
		t.Fatalf("%+v", res)
	}
	prompts, _ := DecodePrompts([]byte(`{"prompts":[{"name":"review","arguments":[{"name":"code","required":true},{"name":"style"}]},{"name":"hello"}]}`))
	if len(prompts) != 2 || prompts[0].Name != "review" || len(prompts[0].Args) != 1 || prompts[0].ArgsJSON() != `{"code":""}` || prompts[1].ArgsJSON() != `{}` {
		t.Fatalf("%+v", prompts)
	}
	badResources, _ := DecodeResources([]byte("x"))
	badPrompts, _ := DecodePrompts([]byte(`{}`))
	if badResources != nil || badPrompts != nil {
		t.Fatal("garbage must give nothing")
	}
}

// One tool with an inputSchema that is no object (true, a string) keeps its
// place in the list with no arguments, and its form is empty.
func TestAMalformedInputSchemaLeavesAnEmptyFormAndTheOtherToolsAlone(t *testing.T) {
	for _, bad := range []string{`true`, `"text"`, `[1]`} {
		result := `{"tools":[{"name":"a","inputSchema":` + bad + `},{"name":"b","inputSchema":{"type":"object","properties":{"q":{"type":"string"}},"required":["q"]}}]}`
		tools, err := DecodeTools([]byte(result))
		if err != nil || len(tools) != 2 {
			t.Fatalf("%s: %+v, %v", bad, tools, err)
		}
		if fields := ToolFields(tools[0].Schema); len(fields) != 0 {
			t.Errorf("%s: a tool with a malformed schema has fields %+v", bad, fields)
		}
		if fields := ToolFields(tools[1].Schema); len(fields) != 1 || fields[0].Name != "q" || !fields[0].Required {
			t.Errorf("%s: the other tool lost its form: %+v", bad, fields)
		}
	}
}
