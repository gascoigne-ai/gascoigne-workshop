package settings

import "testing"

func FuzzDecode(fuzz *testing.F) {
	encoded, err := Encode(Default())
	if err == nil {
		fuzz.Add(encoded)
	}
	fuzz.Add([]byte(`{"theme":"dark","unknown":1}`))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Decode(data)
	})
}
