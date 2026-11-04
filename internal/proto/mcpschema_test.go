package proto

import (
	"testing"
)

func TestParseToolsReadsAToolsListResponse(t *testing.T) {
	msg := `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","description":"Echo it","inputSchema":{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}},{"name":"ping"}]}}`
	tools := ParseTools(msg)
	if len(tools) != 2 || tools[0].Name != "echo" || tools[0].Description != "Echo it" || tools[1].Name != "ping" {
		t.Fatalf("%+v", tools)
	}
	for _, bad := range []string{"", "not json", `{"result":{}}`, `{"result":{"tools":[{"description":"no name"}]}}`} {
		if got := ParseTools(bad); got != nil {
			t.Errorf("ParseTools(%q) = %+v", bad, got)
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
		tools := ParseTools(`{"result":{"tools":[{"name":"t","inputSchema":` + tt.schema + `}]}}`)
		if len(tools) != 1 {
			t.Fatalf("%s: %v", tt.name, tools)
		}
		if got := Skeleton(tools[0].Schema); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.name, got, tt.want)
		}
	}
	if got := Skeleton(map[string]any{"type": "string", "default": "<b>"}); got != `"<b>"` {
		t.Errorf("HTML characters must not be escaped in a skeleton: %s", got)
	}
}

func TestParseResourcesAndPrompts(t *testing.T) {
	res := ParseResources(`{"result":{"resources":[{"uri":"file:///a.txt","name":"A"},{"name":"no uri"}]}}`)
	if len(res) != 1 || res[0].URI != "file:///a.txt" || res[0].Name != "A" {
		t.Fatalf("%+v", res)
	}
	prompts := ParsePrompts(`{"result":{"prompts":[{"name":"review","arguments":[{"name":"code","required":true},{"name":"style"}]},{"name":"hello"}]}}`)
	if len(prompts) != 2 || prompts[0].Name != "review" || len(prompts[0].Args) != 1 || prompts[0].ArgsJSON() != `{"code":""}` || prompts[1].ArgsJSON() != `{}` {
		t.Fatalf("%+v", prompts)
	}
	if ParseResources("x") != nil || ParsePrompts(`{"result":{}}`) != nil {
		t.Fatal("garbage must give nothing")
	}
}
