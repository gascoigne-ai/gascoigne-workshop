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
		`{"version":2,"appearance":{"fontSize":20}}`,
		`{"version":3,"appearance":{"smallFontSize":8,"codeFontSize":32}}`,
		`{"version":3,"appearance":{"smallFontSize":7,"codeFontSize":33}}`,
		`{"version":3,"appearance":{"smallFontSize":"big"}}`,
		`{"textSize":"large"}`,
		`{"version":4,"appearance":{"fontSize":20}}`,
	} {
		fuzz.Add([]byte(seed))
	}
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		decoded, err := Decode(data)
		if err != nil {
			return
		}
		clamped, _ := Clamp(decoded)
		for _, face := range []struct {
			key            string
			size, min, max int
		}{
			{"fontSize", clamped.FontSize, MinFontSize, MaxFontSize},
			{"smallFontSize", clamped.SmallFontSize, MinSmallFontSize, MaxSmallFontSize},
			{"codeFontSize", clamped.CodeFontSize, MinCodeFontSize, MaxCodeFontSize},
		} {
			if face.size < face.min || face.size > face.max {
				t.Fatalf("Clamp(Decode(%q)).%s = %d, outside %d to %d", data, face.key, face.size, face.min, face.max)
			}
		}
	})
}
