package settings

import "testing"

func FuzzDecode(fuzz *testing.F) {
	encoded, err := Encode(Default())
	if err == nil {
		fuzz.Add(encoded)
	}
	fuzz.Add([]byte(`{"theme":"dark","unknown":1}`))
	for _, seed := range []string{
		`{"textSize":"large"}`,
		`{"version":1,"appearance":{"textSize":7}}`,
		`{"version":1,"appearance":{"textSize":"huge"}}`,
		`{"fontSize":99}`,
		`{"version":2,"appearance":{"fontSize":-5}}`,
	} {
		fuzz.Add([]byte(seed))
	}
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := Decode(data)
		if err != nil {
			return
		}
		if clamped, _ := Clamp(decoded); clamped.FontSize < MinFontSize || clamped.FontSize > MaxFontSize {
			t.Fatalf("Clamp(Decode(%q)).FontSize = %d, outside %d to %d", data, clamped.FontSize, MinFontSize, MaxFontSize)
		}
	})
}
