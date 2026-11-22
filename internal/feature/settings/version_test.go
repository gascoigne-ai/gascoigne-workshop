package settings

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
)

// The fixtures are settings.json as the program wrote it after the theme
// and the request timeout were changed: v0 before the file carried a
// version, v1 by the build before the numeric text size.
func TestDecodeReadsSettingsFixtures(t *testing.T) {
	want := Default()
	want.Theme, want.TimeoutSec = "light", 17
	for _, file := range []string{"settings-v0.json", "settings-v1.json"} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			got, err := Decode(data)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("Decode = %+v, %v\nwant %+v", got, err, want)
			}
		})
	}
}

func TestMinimalAlwaysWritesVersionFirst(t *testing.T) {
	changed := Default()
	changed.Theme = "light"
	tests := []struct {
		name  string
		prefs Settings
		want  string
	}{
		{"defaults", Default(), "{\n  \"version\": 3\n}"},
		{"a changed setting", changed, "{\n  \"version\": 3,\n  \"appearance\": {\n    \"theme\": \"light\"\n  }\n}"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Minimal(test.prefs)
			if err != nil || string(got) != test.want {
				t.Fatalf("Minimal = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantMessage string // "" for no problem
	}{
		{"current", `{"version": 3}`, ""},
		{"v2", `{"version": 2}`, ""},
		{"v1", `{"version": 1}`, ""},
		{"v0", `{"version": 0}`, ""},
		{"newer", `{"version": 4}`, "written by a newer version of the workshop (format 4, this build reads up to 3)"},
		{"negative", `{"version": -1}`, "must be a whole number from 0 to 3"},
		{"fractional", `{"version": 1.5}`, "must be a whole number from 0 to 3"},
		{"text", `{"version": "2"}`, "must be a whole number from 0 to 3"},
		{"inside a section", `{"general": {"version": 3}}`, "belongs at the top level"},
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
			if len(problems) != 1 || problems[0].Key != storage.Key || problems[0].Message != test.wantMessage {
				t.Fatalf("Validate(%s) = %+v, want one problem %q on version", test.text, problems, test.wantMessage)
			}
			if at := test.text[problems[0].Start:problems[0].End]; at != `"version"` {
				t.Fatalf("problem points at %q, want the key", at)
			}
		})
	}
}

func TestParseKeepsLastOnNewerVersion(t *testing.T) {
	last := Default()
	last.Theme = "light"
	got, problems := Parse([]byte(`{"version": 4, "appearance": {"theme": "dark"}}`), last)
	if len(problems) == 0 || !reflect.DeepEqual(got, last) {
		t.Fatalf("Parse = %+v, %+v; want the last settings and a problem", got, problems)
	}
}

func TestDecodeRejectsNewerVersion(t *testing.T) {
	got, err := Decode([]byte(`{"version": 4, "appearance": {"theme": "light"}}`))
	var newer *storage.NewerError
	if !errors.As(err, &newer) || !reflect.DeepEqual(got, Default()) {
		t.Fatalf("Decode = %+v, %v; want the defaults and a NewerError", got, err)
	}
}

// The version belongs to the file, not to the preferences: the form, reset
// and completion never see it.
func TestVersionIsNotASetting(t *testing.T) {
	if _, known := FieldFor(storage.Key); known {
		t.Fatal("version is in the schema")
	}
	if got, _ := With(Default(), storage.Key, Number(5)); !reflect.DeepEqual(got, Default()) {
		t.Fatalf("With(version) changed the settings: %+v", got)
	}
	for _, completion := range Complete(`{"`, 2, nil) {
		if strings.Contains(completion.Label, storage.Key) {
			t.Fatalf("completion offers %q", completion.Label)
		}
	}
}

// Versions 0 and 1 wrote the text size as small, medium or large; version 2
// writes the pixel size. Anything the old table does not know reads as the
// default.
func TestDecodeMapsLegacyTextSize(t *testing.T) {
	tests := []struct {
		name  string
		inner string // the appearance section
		want  int
	}{
		{"small", `"textSize": "small"`, 13},
		{"medium", `"textSize": "medium"`, 14},
		{"large", `"textSize": "large"`, 16},
		{"unknown text", `"textSize": "huge"`, 14},
		{"a number", `"textSize": 7`, 14},
		{"absent", `"theme": "light"`, 14},
		{"both keys: fontSize wins", `"textSize": "large", "fontSize": 20`, 20},
	}
	for _, version := range []string{"", `"version": 1, `} {
		for _, test := range tests {
			text := "{" + version + `"appearance": {` + test.inner + "}}"
			t.Run(test.name+"/"+text, func(t *testing.T) {
				got, err := Decode([]byte(text))
				if err != nil || got.FontSize != test.want {
					t.Fatalf("Decode(%s).FontSize = %d, %v; want %d", text, got.FontSize, err, test.want)
				}
			})
		}
	}
}

func TestDecodeMapsLegacyTextSizeFixtures(t *testing.T) {
	for _, file := range []string{"settings-v0-textsize.json", "settings-v1-textsize.json"} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			got, err := Decode(data)
			if err != nil || got.FontSize != 16 {
				t.Fatalf("Decode(%s).FontSize = %d, %v; want 16", file, got.FontSize, err)
			}
		})
	}
}

func TestDecodeReadsFontSizeV2(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "settings-v2-fontsize.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil || got.FontSize != 16 {
		t.Fatalf("Decode(v2).FontSize = %d, %v; want 16", got.FontSize, err)
	}
}

// A file of an older version re-encodes as the newest version.
func TestDecodeRoundTripsOlderFixturesAsV3(t *testing.T) {
	for _, file := range []string{"settings-v0.json", "settings-v1.json", "settings-v0-textsize.json", "settings-v1-textsize.json"} {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := Decode(data)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := Encode(decoded)
			if err != nil || !strings.HasPrefix(string(encoded), "{\n  \"version\": 3") {
				t.Fatalf("Encode = %s, %v; want version 3 first", encoded, err)
			}
			again, err := Decode(encoded)
			if err != nil || !reflect.DeepEqual(again, decoded) {
				t.Fatalf("re-decoded %+v, %v; want %+v", again, err, decoded)
			}
		})
	}
}

func TestValidateLegacyTextSize(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantMessage string // "" for no problem
	}{
		{"v0 small", `{"appearance": {"textSize": "small"}}`, ""},
		{"v1 large", `{"version": 1, "appearance": {"textSize": "large"}}`, ""},
		{"v1 flat", `{"version": 1, "textSize": "medium"}`, ""},
		{"v1 unknown", `{"version": 1, "appearance": {"textSize": "huge"}}`, "must be one of small, medium, large"},
		{"v1 number", `{"version": 1, "appearance": {"textSize": 7}}`, "must be text, one of small, medium, large"},
		{"v2", `{"version": 2, "appearance": {"textSize": "large"}}`, "replaced by fontSize"},
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
			if len(problems) != 1 || problems[0].Key != "textSize" || problems[0].Message != test.wantMessage {
				t.Fatalf("Validate(%s) = %+v, want one problem %q on textSize", test.text, problems, test.wantMessage)
			}
		})
	}
}
