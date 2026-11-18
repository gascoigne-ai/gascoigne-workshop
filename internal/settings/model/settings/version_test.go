package settings

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
)

// The fixtures are settings.json as the program wrote it after the theme
// and the request timeout were changed: v0 before the file carried a
// version, v1 by this build.
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

func TestDecodeRoundTripsV1Fixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "settings-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if encoded, err := Encode(decoded); err != nil || string(encoded) != string(data) {
		t.Fatalf("Encode = %s, %v\nwant the fixture\n%s", encoded, err, data)
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
		{"defaults", Default(), "{\n  \"version\": 1\n}"},
		{"a changed setting", changed, "{\n  \"version\": 1,\n  \"appearance\": {\n    \"theme\": \"light\"\n  }\n}"},
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
		{"current", `{"version": 1}`, ""},
		{"v0", `{"version": 0}`, ""},
		{"newer", `{"version": 2}`, "written by a newer version of the workshop (format 2, this build reads up to 1)"},
		{"negative", `{"version": -1}`, "must be a whole number from 0 to 1"},
		{"fractional", `{"version": 1.5}`, "must be a whole number from 0 to 1"},
		{"text", `{"version": "1"}`, "must be a whole number from 0 to 1"},
		{"inside a section", `{"general": {"version": 1}}`, "belongs at the top level"},
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
			if len(problems) != 1 || problems[0].Key != fileversion.Key || problems[0].Message != test.wantMessage {
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
	got, problems := Parse([]byte(`{"version": 2, "appearance": {"theme": "dark"}}`), last)
	if len(problems) == 0 || !reflect.DeepEqual(got, last) {
		t.Fatalf("Parse = %+v, %+v; want the last settings and a problem", got, problems)
	}
}

func TestDecodeRejectsNewerVersion(t *testing.T) {
	got, err := Decode([]byte(`{"version": 2, "appearance": {"theme": "light"}}`))
	var newer *fileversion.NewerError
	if !errors.As(err, &newer) || !reflect.DeepEqual(got, Default()) {
		t.Fatalf("Decode = %+v, %v; want the defaults and a NewerError", got, err)
	}
}

// The version belongs to the file, not to the preferences: the form, reset
// and completion never see it.
func TestVersionIsNotASetting(t *testing.T) {
	if _, known := FieldFor(fileversion.Key); known {
		t.Fatal("version is in the schema")
	}
	if got := With(Default(), fileversion.Key, 5); !reflect.DeepEqual(got, Default()) {
		t.Fatalf("With(version) changed the settings: %+v", got)
	}
	for _, completion := range Complete(`{"`, 2, nil) {
		if strings.Contains(completion.Label, fileversion.Key) {
			t.Fatalf("completion offers %q", completion.Label)
		}
	}
}
