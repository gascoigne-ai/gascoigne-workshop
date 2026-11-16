package model

import "testing"

func FuzzParseSchema(f *testing.F) {
	f.Add([]byte(introspectionAnswer))
	f.Add([]byte(`{"data":{"__schema":{"types":[]}}}`))
	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = ParseSchema(body)
	})
}

func FuzzParseMCPLists(f *testing.F) {
	f.Add(`{"result":{"tools":[{"name":"a","inputSchema":{"type":"object","properties":{"x":{"type":"string"}}}}]}}`)
	f.Add(`{"result":{"resources":[{"uri":"file:///a","name":"a"}]}}`)
	f.Add(`{"result":{"prompts":[{"name":"p","arguments":[{"name":"x","required":true}]}]}}`)
	f.Fuzz(func(t *testing.T, message string) {
		for _, tool := range ParseTools(message) {
			_ = Skeleton(tool.Schema)
		}
		_ = ParseResources(message)
		for _, prompt := range ParsePrompts(message) {
			_ = prompt.ArgsJSON()
		}
	})
}

func FuzzParseServerInfo(f *testing.F) {
	f.Add(`{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"s","version":"1"}}`)
	f.Fuzz(func(t *testing.T, result string) {
		_, _ = ParseServerInfo(result)
	})
}
