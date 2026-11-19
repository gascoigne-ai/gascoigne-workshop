package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func values(texts ...string) []JSONValue {
	out := make([]JSONValue, len(texts))
	for i, text := range texts {
		out[i] = JSONValue(text)
	}
	return out
}

func nested(depth int) string {
	return strings.Repeat(`{"type":"object","properties":{"a":`, depth) + `{"type":"string"}` + strings.Repeat(`}}`, depth)
}

func schemaDepth(schema JSONSchema) int {
	deepest := 0
	for _, property := range schema.Properties {
		deepest = max(deepest, schemaDepth(property.Schema))
	}
	if schema.Items != nil {
		deepest = max(deepest, schemaDepth(*schema.Items))
	}
	return deepest + 1
}

func TestDecodeJSONSchema(t *testing.T) {
	tests := []struct {
		name string
		json string
		want JSONSchema
	}{
		{"every keyword", `{"type":"object","description":"d","properties":{"a":{"type":"string"}},"required":["a"],"items":{"type":"number"},"enum":["x",1],"default":"x","const":2,"$ref":"#/c/s"}`, JSONSchema{Type: SchemaObject, Description: "d", Properties: []SchemaProperty{{Name: "a", Required: true, Schema: JSONSchema{Type: SchemaString}}},
			Items: &JSONSchema{Type: SchemaNumber}, Enum: values(`"x"`, `1`), Default: JSONValue(`"x"`), Const: JSONValue(`2`), Ref: "#/c/s"}},
		{"type list skips null", `{"type":["null","integer"]}`, JSONSchema{Type: SchemaInteger}},
		{"type list of only null", `{"type":["null"]}`, JSONSchema{Type: SchemaNull}},
		{"object inferred from properties", `{"properties":{"a":{}}}`, JSONSchema{Type: SchemaObject, Properties: []SchemaProperty{{Name: "a"}}}},
		{"unknown type is none", `{"type":"banana"}`, JSONSchema{}},
		{"non-string type is none", `{"type":5}`, JSONSchema{}},
		{"required without property is dropped", `{"type":"object","properties":{"a":{}},"required":["a","ghost"]}`,
			JSONSchema{Type: SchemaObject, Properties: []SchemaProperty{{Name: "a", Required: true}}}},
		{"true is the zero schema", `true`, JSONSchema{}},
		{"false is the zero schema", `false`, JSONSchema{}},
		{"string is the zero schema", `"x"`, JSONSchema{}},
		{"null is the zero schema", `null`, JSONSchema{}},
		{"array is the zero schema", `[{"type":"string"}]`, JSONSchema{}},
		{"property that is true is the zero schema", `{"properties":{"a":true}}`, JSONSchema{Type: SchemaObject, Properties: []SchemaProperty{{Name: "a"}}}},
		{"document order is kept", `{"properties":{"z":{},"a":{},"m":{}}}`, JSONSchema{Type: SchemaObject, Properties: []SchemaProperty{{Name: "z"}, {Name: "a"}, {Name: "m"}}}},
		{"later duplicate wins", `{"properties":{"a":{"type":"string"},"b":{},"a":{"type":"number"}}}`,
			JSONSchema{Type: SchemaObject, Properties: []SchemaProperty{{Name: "a", Schema: JSONSchema{Type: SchemaNumber}}, {Name: "b"}}}},
		{"ref is kept", `{"$ref":"#/components/schemas/Pet"}`, JSONSchema{Ref: "#/components/schemas/Pet"}},
		{"unknown keywords are ignored", `{"type":"string","pattern":"x","anyOf":[{}],"$defs":{}}`, JSONSchema{Type: SchemaString}},
		{"enum entries keep their bytes", `{"enum":[null,{"a":1}]}`, JSONSchema{Enum: values(`null`, `{"a":1}`)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DecodeJSONSchema([]byte(test.json))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got  %#v\nwant %#v", got, test.want)
			}
		})
	}
}

func TestDecodeJSONSchemaRejectsInvalidJSON(t *testing.T) {
	for _, text := range []string{``, `{`, `{"type":}`, `{"properties":{"a":}}`, `{} x`} {
		if _, err := DecodeJSONSchema([]byte(text)); err == nil {
			t.Errorf("%q: no error", text)
		}
	}
}

func TestDecodeJSONSchemaCutsDeepSchemas(t *testing.T) {
	tests := []struct {
		depth     int
		wantDepth int
	}{
		{depth: 10, wantDepth: 11},
		{depth: maxSchemaDepth - 1, wantDepth: maxSchemaDepth},
		{depth: maxSchemaDepth, wantDepth: maxSchemaDepth},
		{depth: maxSchemaDepth + 1, wantDepth: maxSchemaDepth},
		{depth: 3000, wantDepth: maxSchemaDepth},
	}
	for _, test := range tests {
		t.Run(fmt.Sprint(test.depth), func(t *testing.T) {
			got, err := DecodeJSONSchema([]byte(nested(test.depth)))
			if err != nil {
				t.Fatal(err)
			}
			if depth := schemaDepth(got); depth != test.wantDepth {
				t.Errorf("depth %d, want %d", depth, test.wantDepth)
			}
		})
	}
}

func TestDecodeJSONSchemaCutsItemsNesting(t *testing.T) {
	text := strings.Repeat(`{"type":"array","items":`, 100) + `{"type":"string"}` + strings.Repeat(`}`, 100)
	got, err := DecodeJSONSchema([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if depth := schemaDepth(got); depth > maxSchemaDepth {
		t.Errorf("depth %d", depth)
	}
}

func TestDecodeJSONSchemaCutsManyProperties(t *testing.T) {
	var text strings.Builder
	text.WriteString(`{"properties":{`)
	for i := range maxSchemaProperties + 1 {
		if i > 0 {
			text.WriteString(",")
		}
		fmt.Fprintf(&text, `"p%d":{}`, i)
	}
	text.WriteString(`}}`)
	got, err := DecodeJSONSchema([]byte(text.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Properties) != maxSchemaProperties {
		t.Fatalf("%d properties, want %d", len(got.Properties), maxSchemaProperties)
	}
	if first, last := got.Properties[0].Name, got.Properties[maxSchemaProperties-1].Name; first != "p0" || last != fmt.Sprintf("p%d", maxSchemaProperties-1) {
		t.Errorf("kept %q to %q", first, last)
	}
}

func TestDecodeJSONSchemaCutsHugeEnums(t *testing.T) {
	text := `{"enum":[` + strings.Repeat(`1,`, maxSchemaEnum+500) + `1]}`
	got, err := DecodeJSONSchema([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Enum) != maxSchemaEnum {
		t.Errorf("%d enum entries, want %d", len(got.Enum), maxSchemaEnum)
	}
}

func TestJSONValueText(t *testing.T) {
	tests := []struct {
		name  string
		value JSONValue
		want  string
	}{
		{"absent", nil, ""},
		{"string is unquoted", JSONValue(`"a b"`), "a b"},
		{"escaped string", JSONValue(`"a\"b"`), `a"b`},
		{"number", JSONValue(`1.5`), "1.5"},
		{"object is compact", JSONValue(`{ "a" : 1 }`), `{"a":1}`},
		{"null", JSONValue(`null`), "null"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.value.Text(); got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

// The tools/list answer of the example server (testdata/mcp/README.md)
// decodes with the list decoders and with the schema decoder.
func TestExampleServerFixturesDecode(t *testing.T) {
	result, err := os.ReadFile(filepath.Join("testdata", "mcp", "tools-list-result.json"))
	if err != nil {
		t.Fatal(err)
	}
	tools, err := DecodeTools(result)
	if err != nil || len(tools) != 2 {
		t.Fatalf("tools %v, %v", tools, err)
	}
	var raw struct {
		Tools []struct {
			InputSchema json.RawMessage `json:"inputSchema"`
		}
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		t.Fatal(err)
	}
	want := []JSONSchema{
		{Type: SchemaObject, Properties: []SchemaProperty{{Name: "text", Required: true, Schema: JSONSchema{Type: SchemaString}}}},
		{Type: SchemaObject, Properties: []SchemaProperty{{Name: "a", Required: true, Schema: JSONSchema{Type: SchemaNumber}}, {Name: "b", Required: true, Schema: JSONSchema{Type: SchemaNumber}}}},
	}
	if len(raw.Tools) != len(want) {
		t.Fatalf("%d raw tools", len(raw.Tools))
	}
	for i, tool := range raw.Tools {
		got, err := DecodeJSONSchema(tool.InputSchema)
		if err != nil || !reflect.DeepEqual(got, want[i]) {
			t.Errorf("tool %d: %#v, %v", i, got, err)
		}
	}
}
