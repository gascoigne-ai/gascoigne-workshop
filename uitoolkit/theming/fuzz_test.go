package theming

import "testing"

func FuzzParseTheme(fuzz *testing.F) {
	fuzz.Add([]byte(`{"name":"x","extends":"dark","colors":{"bg":"#101010"}}`))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		base := Dark()
		_, _ = ParseTheme(data, &base)
	})
}
