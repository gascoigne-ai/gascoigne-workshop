package settings

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type faceSizes struct{ ui, small, code int }

func sizesOf(prefs Settings) faceSizes {
	return faceSizes{prefs.FontSize, prefs.SmallFontSize, prefs.CodeFontSize}
}

// Version 3 gave the small and code faces their own size. Older files carried
// only fontSize (or the words of textSize), so a reader derives the two
// sizes the old rule gave and nothing changes on screen.
func TestDecodeDerivesFaceSizesOfOlderFiles(t *testing.T) {
	tests := []struct {
		name string
		text string
		want faceSizes
	}{
		{"v0 textSize large", `{"appearance": {"textSize": "large"}}`, faceSizes{16, 14, 15}},
		{"v1 textSize small", `{"version": 1, "appearance": {"textSize": "small"}}`, faceSizes{13, 11, 12}},
		{"v1 flat textSize medium", `{"version": 1, "textSize": "medium"}`, faceSizes{14, 12, 13}},
		{"v0 without a size", `{"appearance": {"theme": "light"}}`, faceSizes{14, 12, 13}},
		{"v2 fontSize 20", `{"version": 2, "appearance": {"fontSize": 20}}`, faceSizes{20, 18, 19}},
		{"v2 lowest keeps small readable", `{"version": 2, "appearance": {"fontSize": 10}}`, faceSizes{10, 9, 9}},
		{"v2 highest", `{"version": 2, "appearance": {"fontSize": 24}}`, faceSizes{24, 22, 23}},
		{"v2 beyond the range derives from the clamped size", `{"version": 2, "appearance": {"fontSize": 99}}`, faceSizes{99, 22, 23}},
		{"v2 no size", `{"version": 2, "appearance": {"theme": "light"}}`, faceSizes{14, 12, 13}},
		{"v2 explicit small stays", `{"version": 2, "appearance": {"fontSize": 20, "smallFontSize": 11}}`, faceSizes{20, 11, 19}},
		{"v2 explicit code stays", `{"version": 2, "appearance": {"fontSize": 20, "codeFontSize": 9}}`, faceSizes{20, 18, 9}},
		{"v3 only fontSize does not derive", `{"version": 3, "appearance": {"fontSize": 20}}`, faceSizes{20, 12, 13}},
		{"v3 all three", `{"version": 3, "appearance": {"fontSize": 20, "smallFontSize": 8, "codeFontSize": 32}}`, faceSizes{20, 8, 32}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Decode([]byte(test.text))
			if err != nil || sizesOf(got) != test.want {
				t.Fatalf("Decode(%s) sizes = %+v, %v; want %+v", test.text, sizesOf(got), err, test.want)
			}
		})
	}
}

func TestDecodeFaceSizeThatIsNotANumberIsAnError(t *testing.T) {
	for _, text := range []string{
		`{"version": 2, "appearance": {"fontSize": "big"}}`,
		`{"version": 3, "appearance": {"smallFontSize": "big"}}`,
	} {
		if _, err := Decode([]byte(text)); err == nil {
			t.Errorf("Decode(%s) succeeded", text)
		}
	}
}

// The fixture is real output of Encode for three non-default sizes.
func TestDecodeReadsFaceSizesV3AndEncodesThemBack(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "settings-v3-fontsizes.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if want := (faceSizes{16, 8, 32}); err != nil || sizesOf(got) != want {
		t.Fatalf("Decode(v3) sizes = %+v, %v; want %+v", sizesOf(got), err, want)
	}
	if encoded, err := Encode(got); err != nil || string(encoded) != string(data) {
		t.Fatalf("Encode = %s, %v\nwant the fixture\n%s", encoded, err, data)
	}
}

func TestClampFaceSizes(t *testing.T) {
	faces := []struct {
		key           string
		min, max, def int
		get           func(Settings) int
		set           func(*Settings, int)
	}{
		{"fontSize", MinFontSize, MaxFontSize, DefaultFontSize, func(s Settings) int { return s.FontSize }, func(s *Settings, v int) { s.FontSize = v }},
		{"smallFontSize", MinSmallFontSize, MaxSmallFontSize, DefaultSmallFontSize, func(s Settings) int { return s.SmallFontSize }, func(s *Settings, v int) { s.SmallFontSize = v }},
		{"codeFontSize", MinCodeFontSize, MaxCodeFontSize, DefaultCodeFontSize, func(s Settings) int { return s.CodeFontSize }, func(s *Settings, v int) { s.CodeFontSize = v }},
	}
	for _, face := range faces {
		cases := []struct {
			name      string
			size      int
			want      int
			wantNotes int
		}{
			{"inside", face.min + 1, face.min + 1, 0},
			{"lowest", face.min, face.min, 0},
			{"highest", face.max, face.max, 0},
			{"zero is the default", 0, face.def, 1},
			{"below", face.min - 1, face.min, 1},
			{"negative", -5, face.min, 1},
			{"above", face.max + 1, face.max, 1},
		}
		for _, test := range cases {
			t.Run(face.key+"/"+test.name, func(t *testing.T) {
				prefs := Default()
				face.set(&prefs, test.size)
				clamped, notes := Clamp(prefs)
				if got := face.get(clamped); got != test.want || len(notes) != test.wantNotes {
					t.Fatalf("Clamp(%d) = %d with notes %q; want %d with %d notes", test.size, got, notes, test.want, test.wantNotes)
				}
				if test.wantNotes > 0 && !strings.Contains(notes[0], face.key+" ") {
					t.Fatalf("note %q does not name %s", notes[0], face.key)
				}
			})
		}
	}
}

func TestClampNoteStatesTheRange(t *testing.T) {
	prefs := Default()
	prefs.SmallFontSize = 7
	_, notes := Clamp(prefs)
	if want := "smallFontSize 7 is outside 8 to 24, using 8"; len(notes) != 1 || notes[0] != want {
		t.Fatalf("notes = %q; want %q", notes, want)
	}
}

func TestFaceSizeRows(t *testing.T) {
	rows := []struct {
		key, title, description string
		min, max                int
		expectation             string
	}{
		{"fontSize", "Interface font size", "The size of the interface text in pixels, 10 to 24.", 10, 24, "a whole number from 10 to 24"},
		{"smallFontSize", "Small text size", "The size of captions, hints and table headers in pixels, 8 to 24.", 8, 24, "a whole number from 8 to 24"},
		{"codeFontSize", "Code font size", "The size of bodies and code views in pixels, 8 to 32.", 8, 32, "a whole number from 8 to 32"},
	}
	for _, want := range rows {
		t.Run(want.key, func(t *testing.T) {
			field, ok := FieldFor(want.key)
			if !ok || field.Kind != Int || field.Category != "Appearance" || field.Min != want.min || field.Max != want.max || field.Title != want.title || field.Description != want.description {
				t.Fatalf("FieldFor(%s) = %+v, %v", want.key, field, ok)
			}
			if got := Expectation(field); got != want.expectation {
				t.Fatalf("Expectation = %q, want %q", got, want.expectation)
			}
		})
	}
}

// The font block sits together: each size right after the font it sizes.
func TestAppearanceRowOrderPutsEachSizeNextToItsFont(t *testing.T) {
	var keys []string
	for _, field := range Schema {
		if field.Category == "Appearance" {
			keys = append(keys, field.Key)
		}
	}
	want := []string{"theme", "uiFont", "fontSize", "smallFontSize", "codeFont", "codeFontSize", "uiScale", "corners", "flat", "contrast", "layout", "responseView"}
	if !slices.Equal(keys, want) {
		t.Fatalf("Appearance keys = %v; want %v", keys, want)
	}
}

func TestValidateSmallAndCodeFontSize(t *testing.T) {
	tests := []struct {
		name, key, text, wantMessage string
	}{
		{"small inside", "smallFontSize", `{"version": 3, "appearance": {"smallFontSize": 12}}`, ""},
		{"small lowest", "smallFontSize", `{"version": 3, "appearance": {"smallFontSize": 8}}`, ""},
		{"small below", "smallFontSize", `{"version": 3, "appearance": {"smallFontSize": 7}}`, "must be a whole number between 8 and 24"},
		{"small above", "smallFontSize", `{"version": 3, "appearance": {"smallFontSize": 25}}`, "must be a whole number between 8 and 24"},
		{"small text", "smallFontSize", `{"version": 3, "appearance": {"smallFontSize": "big"}}`, "must be a number"},
		{"code highest", "codeFontSize", `{"version": 3, "appearance": {"codeFontSize": 32}}`, ""},
		{"code above", "codeFontSize", `{"version": 3, "appearance": {"codeFontSize": 33}}`, "must be a whole number between 8 and 32"},
		{"code fractional", "codeFontSize", `{"version": 3, "appearance": {"codeFontSize": 13.5}}`, "must be a whole number between 8 and 32"},
		{"small in a v2 file", "smallFontSize", `{"version": 2, "appearance": {"smallFontSize": 12}}`, ""},
		{"code flat in a v0 file", "codeFontSize", `{"codeFontSize": 12}`, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			problems := Validate([]byte(test.text))
			if test.wantMessage == "" {
				if len(problems) != 0 {
					t.Fatalf("Validate(%s) = %+v, want none", test.text, problems)
				}
				return
			}
			if len(problems) != 1 || problems[0].Key != test.key || problems[0].Message != test.wantMessage {
				t.Fatalf("Validate(%s) = %+v, want %q on %s", test.text, problems, test.wantMessage, test.key)
			}
		})
	}
}

func TestMinimalWritesFaceSizesOnlyWhenChanged(t *testing.T) {
	text, err := Minimal(Default())
	if err != nil || strings.Contains(string(text), "FontSize") {
		t.Fatalf("Minimal(default) = %s, %v; want no sizes", text, err)
	}
	prefs := Default()
	prefs.SmallFontSize, prefs.CodeFontSize = 8, 32
	text, err = Minimal(prefs)
	if err != nil || !strings.Contains(string(text), `"smallFontSize": 8`) || !strings.Contains(string(text), `"codeFontSize": 32`) || strings.Contains(string(text), `"fontSize"`) {
		t.Fatalf("Minimal = %s, %v; want the two changed sizes only", text, err)
	}
}

func TestSetAndResetFaceSizes(t *testing.T) {
	for _, key := range []string{"smallFontSize", "codeFontSize"} {
		prefs := Default()
		if err := Set(&prefs, key, Number(9)); err != nil {
			t.Fatal(err)
		}
		if !Changed(prefs, key) {
			t.Errorf("%s 9 is not changed", key)
		}
		if err := Set(&prefs, key, Text("big")); !errors.Is(err, ErrWrongValue) {
			t.Errorf("Set(%s, Text) = %v, want ErrWrongValue", key, err)
		}
		if err := Reset(&prefs, key); err != nil || Changed(prefs, key) {
			t.Errorf("Reset(%s) = %v, changed %v", key, err, Changed(prefs, key))
		}
	}
}

func TestLegacyTextSizeKeyIsNotInTheSchema(t *testing.T) {
	if _, known := FieldFor("textSize"); known {
		t.Fatal("the legacy key is in the schema")
	}
}
