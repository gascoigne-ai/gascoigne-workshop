package settings

import "testing"

func FuzzDecode(f *testing.F) {
	encoded, err := Encode(Default())
	if err == nil {
		f.Add(encoded)
	}
	f.Add([]byte(`{"theme":"dark","unknown":1}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Decode(data)
	})
}
