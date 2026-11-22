package send

import "testing"

func FuzzParseSchema(fuzz *testing.F) {
	fuzz.Add([]byte(introspectionAnswer))
	fuzz.Add([]byte(`{"data":{"__schema":{"types":[]}}}`))
	fuzz.Fuzz(func(t *testing.T, body []byte) {
		_, _ = ParseSchema(body)
	})
}
