package request

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzParseMCPLists(fuzz *testing.F) {
	fuzz.Add(`{"tools":[{"name":"a","inputSchema":{"type":"object","properties":{"x":{"type":"string"}}}}]}`)
	fuzz.Add(`{"resources":[{"uri":"file:///a","name":"a"}]}`)
	fuzz.Add(`{"prompts":[{"name":"p","arguments":[{"name":"x","required":true}]}]}`)
	fuzz.Fuzz(func(t *testing.T, result string) {
		tools, _ := DecodeTools([]byte(result))
		for _, tool := range tools {
			_ = Skeleton(tool.Schema)
			_ = ToolFields(tool.Schema)
		}
		_, _ = DecodeResources([]byte(result))
		prompts, _ := DecodePrompts([]byte(result))
		for _, prompt := range prompts {
			_ = prompt.ArgsJSON()
		}
	})
}

// A schema decodes within the bounds of the decoder, or to an error, and
// never panics.
func FuzzDecodeJSONSchema(fuzz *testing.F) {
	fuzz.Add(`{"type":"object","description":"d","properties":{"a":{"type":["null","string"]}},"required":["a","b"],"items":{"type":"number"},"enum":["x",1],"default":"x","const":2,"$ref":"#/c/s"}`)
	fuzz.Add(`true`)
	fuzz.Add(`{"properties":{"a":true,"a":{"properties":{"b":{}}}}}`)
	fuzz.Add(`{"enum":[null,{"a":[1,2]},"s"]}`)
	fuzz.Add(strings.Repeat(`{"properties":{"a":`, 40) + `{}` + strings.Repeat(`}}`, 40))
	fuzz.Add(strings.Repeat(`{"items":`, 40) + `{}` + strings.Repeat(`}`, 40))
	if fixture, err := os.ReadFile(filepath.Join("testdata", "mcp", "tools-list-result.json")); err == nil {
		fuzz.Add(string(fixture))
	}
	fuzz.Fuzz(func(t *testing.T, text string) {
		schema, err := DecodeJSONSchema([]byte(text))
		if err != nil {
			return
		}
		if depth := schemaDepth(schema); depth > maxSchemaDepth {
			t.Fatalf("depth %d passed the bound", depth)
		}
		checkSchemaCounts(t, schema)
		_ = Skeleton(schema)
		_ = ToolFields(schema)
	})
}

func checkSchemaCounts(t *testing.T, schema JSONSchema) {
	t.Helper()
	if len(schema.Properties) > maxSchemaProperties || len(schema.Enum) > maxSchemaEnum {
		t.Fatalf("%d properties and %d enum entries passed the bound", len(schema.Properties), len(schema.Enum))
	}
	seen := map[string]bool{}
	for _, property := range schema.Properties {
		if seen[property.Name] {
			t.Fatalf("property %q twice", property.Name)
		}
		seen[property.Name] = true
		checkSchemaCounts(t, property.Schema)
	}
	if schema.Items != nil {
		checkSchemaCounts(t, *schema.Items)
	}
}

// A protocol, body type or auth type parsed from any text is one of the
// known values and parses again to itself.
func FuzzParseProtocol(fuzz *testing.F) {
	fuzz.Add("rest")
	fuzz.Add("ws ")
	fuzz.Add("")
	fuzz.Fuzz(func(t *testing.T, text string) {
		protocol, err := ParseProtocol(text)
		if err != nil {
			return
		}
		if again, err := ParseProtocol(string(protocol)); err != nil || again != protocol {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, protocol, again, err)
		}
	})
}

func FuzzParseBodyType(fuzz *testing.F) {
	fuzz.Add("json")
	fuzz.Add("")
	fuzz.Add("xml")
	fuzz.Fuzz(func(t *testing.T, text string) {
		bodyType, err := ParseBodyType(text)
		if err != nil {
			return
		}
		if again, err := ParseBodyType(string(bodyType)); err != nil || again != bodyType {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, bodyType, again, err)
		}
	})
}

func FuzzParseAuthType(fuzz *testing.F) {
	fuzz.Add("bearer")
	fuzz.Add("none")
	fuzz.Add("")
	fuzz.Fuzz(func(t *testing.T, text string) {
		authType, err := ParseAuthType(text)
		if err != nil {
			return
		}
		if again, err := ParseAuthType(string(authType)); err != nil || again != authType {
			t.Fatalf("%q parsed to %q, which parses to %q, %v", text, authType, again, err)
		}
	})
}
