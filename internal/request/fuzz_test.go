package request

import "testing"

func FuzzParseSchema(fuzz *testing.F) {
	fuzz.Add([]byte(introspectionAnswer))
	fuzz.Add([]byte(`{"data":{"__schema":{"types":[]}}}`))
	fuzz.Fuzz(func(t *testing.T, body []byte) {
		_, _ = ParseSchema(body)
	})
}

func FuzzParseServerInfo(fuzz *testing.F) {
	fuzz.Add(`{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"s","version":"1"}}`)
	fuzz.Fuzz(func(t *testing.T, result string) {
		_, _ = ParseServerInfo(result)
	})
}
