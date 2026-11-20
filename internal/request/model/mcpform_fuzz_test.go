package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// formFromTools yields the form fields of every tool a tools/list result
// holds.
func formFromTools(result string) [][]ToolField {
	tools, err := DecodeTools([]byte(result))
	if err != nil {
		return nil
	}
	var forms [][]ToolField
	for _, tool := range tools {
		forms = append(forms, ToolFields(tool.Schema))
	}
	return forms
}

// The first seed is the captured tools/list result in testdata/mcp; the
// others are synthetic.
func addToolSeeds(fuzz *testing.F) {
	if fixture, err := os.ReadFile(filepath.Join("testdata", "mcp", "tools-list-result.json")); err == nil {
		fuzz.Add(string(fixture))
	}
	fuzz.Add(`{"tools":[{"name":"t","inputSchema":{"type":"object","required":["a","b"],"properties":{"a":{"type":"integer"},"b":{"enum":["x",1]},"c":{"type":"boolean","default":null}}}}]}`)
	fuzz.Add(`{"tools":[{"name":"t","inputSchema":{"properties":{"a":true}}}]}`)
}

// Fields are listed once per property, required ones first, and never
// panic.
func FuzzToolFields(fuzz *testing.F) {
	addToolSeeds(fuzz)
	fuzz.Fuzz(func(t *testing.T, result string) {
		for _, fields := range formFromTools(result) {
			seenOptional := false
			for _, field := range fields {
				if field.Required && seenOptional {
					t.Fatalf("required field %q after an optional one", field.Name)
				}
				seenOptional = seenOptional || !field.Required
			}
		}
	})
}

// Typed form values give valid JSON or an error, never a panic.
func FuzzArguments(fuzz *testing.F) {
	fuzz.Add(`{"tools":[{"name":"t","inputSchema":{"type":"object","required":["a"],"properties":{"a":{"type":"integer"},"b":{"type":"string"},"c":{"type":"object"},"d":{"type":"boolean"},"e":{"enum":["x",1]}}}}]}`, "1.5e400")
	fuzz.Add(`{"tools":[{"name":"t","inputSchema":{"properties":{"a":{"type":"number"},"c":{"type":"object"}}}}]}`, `{"x":[`)
	fuzz.Add(`{"tools":[{"name":"t","inputSchema":{"properties":{"a":{"type":"array"},"e":{"enum":[1,"x"]}}}}]}`, "x")
	if fixture, err := os.ReadFile(filepath.Join("testdata", "mcp", "tools-list-result.json")); err == nil {
		fuzz.Add(string(fixture), "7")
	}
	fuzz.Fuzz(func(t *testing.T, result, value string) {
		for _, fields := range formFromTools(result) {
			values := map[string]string{}
			for _, field := range fields {
				values[field.Name] = value
			}
			out, err := Arguments(fields, values)
			if err == nil && !json.Valid([]byte(out)) {
				t.Fatalf("arguments are not JSON: %q", out)
			}
		}
	})
}
