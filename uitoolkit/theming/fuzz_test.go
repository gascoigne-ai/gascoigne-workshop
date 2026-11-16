package theming

import "testing"

func FuzzParseTheme(f *testing.F) {
	f.Add([]byte(`{"name":"x","extends":"dark","colors":{"bg":"#101010"}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		base := Dark()
		_, _ = ParseTheme(data, &base)
	})
}
