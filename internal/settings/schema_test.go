package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// jsonKeys are the keys a Settings value can be written with.
func jsonKeys(t *testing.T) []string {
	t.Helper()
	typ := reflect.TypeOf(Settings{})
	var keys []string
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		keys = append(keys, name)
	}
	return keys
}

func TestTheSchemaListsEveryFieldOfSettingsExactlyOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, field := range Schema {
		if seen[field.Key] {
			t.Errorf("%s is listed twice", field.Key)
		}
		seen[field.Key] = true
		if field.Title == "" || field.Description == "" || field.Category == "" {
			t.Errorf("%s lacks a title, description or category: %+v", field.Key, field)
		}
	}
	for _, key := range jsonKeys(t) {
		if !seen[key] {
			t.Errorf("settings key %s is missing from the schema", key)
		}
	}
	if len(seen) != len(jsonKeys(t)) {
		t.Errorf("schema has %d keys, Settings has %d", len(seen), len(jsonKeys(t)))
	}
}

func TestEnumDefaultsAreAllowedValues(t *testing.T) {
	defaults := defaultsMap(t)
	for _, field := range Schema {
		if field.Kind != Enum {
			continue
		}
		value, _ := defaults[field.Key].(string)
		if !contains(field.Allowed, value) {
			t.Errorf("%s: default %q is not among %v", field.Key, value, field.Allowed)
		}
	}
}

func defaultsMap(t *testing.T) map[string]any {
	t.Helper()
	data, _ := json.Marshal(Default())
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestValidateReportsUnknownKeysWrongTypesAndBadValues(t *testing.T) {
	tests := []struct {
		name, text string
		wantKey    string // the key of the one expected problem; "" for none
		wantIn     string // text its message contains
	}{
		{"valid", `{"theme": "dark", "uiScale": 120}`, "", ""},
		{"empty object", `{}`, "", ""},
		{"unknown key", `{"colour": "red"}`, "colour", "unknown setting"},
		{"wrong type string", `{"theme": 3}`, "theme", "text"},
		{"wrong type number", `{"timeoutSec": "ten"}`, "timeoutSec", "number"},
		{"wrong type bool", `{"flat": "yes"}`, "flat", "true or false"},
		{"enum value not allowed", `{"layout": "diagonal"}`, "layout", "side"},
		{"number below range", `{"uiScale": 10}`, "uiScale", "between"},
		{"number above range", `{"uiScale": 900}`, "uiScale", "between"},
		{"shortcuts must be an object of text", `{"shortcuts": {"send": 1}}`, "shortcuts", "text"},
		{"colors must be nested objects", `{"colors": {"dark": "red"}}`, "colors", "object"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			problems := Validate([]byte(test.text))
			if test.wantKey == "" {
				if len(problems) != 0 {
					t.Fatalf("problems = %+v", problems)
				}
				return
			}
			if len(problems) != 1 || problems[0].Key != test.wantKey || !strings.Contains(problems[0].Message, test.wantIn) {
				t.Fatalf("problems = %+v, want one for %s containing %q", problems, test.wantKey, test.wantIn)
			}
		})
	}
}

func TestValidateGivesTheTextRangeOfTheKey(t *testing.T) {
	text := "{\n  \"theme\": \"dark\",\n  \"colour\": 1\n}"
	problems := Validate([]byte(text))
	if len(problems) != 1 {
		t.Fatalf("problems = %+v", problems)
	}
	got := text[problems[0].Start:problems[0].End]
	if got != `"colour"` {
		t.Fatalf("range covers %q, want the key", got)
	}
}

func TestValidateReportsASyntaxErrorWithItsPosition(t *testing.T) {
	problems := Validate([]byte("{\"theme\": \"dark\",\n \"layout\": }"))
	if len(problems) != 1 || problems[0].Key != "" || problems[0].Message == "" {
		t.Fatalf("problems = %+v", problems)
	}
}

func TestParseKeepsTheLastGoodSettingsWhenTheTextIsBad(t *testing.T) {
	good := Default()
	good.Layout = "stacked"
	got, problems := Parse([]byte(`{"layout": "diagonal"`), good)
	if len(problems) == 0 || got.Layout != "stacked" {
		t.Fatalf("got %+v with problems %+v", got, problems)
	}
	got, problems = Parse([]byte(`{"layout": "side", "uiScale": 150}`), good)
	if len(problems) != 0 || got.Layout != "side" || got.UIScalePct != 150 {
		t.Fatalf("got %+v with problems %+v", got, problems)
	}
}

func TestChangedAndResetUseTheDefaults(t *testing.T) {
	s := Default()
	if Changed(s, "layout") {
		t.Fatal("a default value is marked changed")
	}
	s.Layout = "stacked"
	s.Shortcuts = map[string]string{"send": "Ctrl+Enter"}
	if !Changed(s, "layout") || !Changed(s, "shortcuts") {
		t.Fatal("a changed value is not marked")
	}
	Reset(&s, "layout")
	Reset(&s, "shortcuts")
	if s.Layout != Default().Layout || s.Shortcuts != nil || Changed(s, "layout") || Changed(s, "shortcuts") {
		t.Fatalf("after reset: %+v", s)
	}
}

func TestMarshalLeavesOutWhatIsAtItsDefault(t *testing.T) {
	s := Default()
	s.Layout = "stacked"
	text, err := Minimal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(text, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["layout"] != "stacked" {
		t.Fatalf("file = %s", text)
	}
}

func TestCompleteOffersMissingKeysAndEnumValues(t *testing.T) {
	tests := []struct {
		name, text string
		want       []string
		notWant    []string
	}{
		{"keys in an empty object", `{ "`, []string{"theme", "layout"}, nil},
		{"key prefix", `{ "la`, []string{"layout"}, []string{"theme"}},
		{"present keys are not offered again", `{"layout": "side", "`, []string{"theme"}, []string{"layout"}},
		{"enum values after the colon", `{"layout": "`, []string{"side", "stacked"}, []string{"dark"}},
		{"enum value prefix", `{"layout": "st`, []string{"stacked"}, []string{"side"}},
		{"enum value before a quote", `{"layout": `, []string{"side", "stacked"}, nil},
		{"booleans", `{"flat": `, []string{"true", "false"}, nil},
		{"nothing for free text", `{"proxy": "`, nil, []string{"side", "theme"}},
		{"shortcut action ids inside shortcuts", `{"shortcuts": { "`, []string{"send"}, []string{"theme"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Complete(test.text, len(test.text), []string{"send", "save"})
			labels := map[string]bool{}
			for _, c := range got {
				labels[c.Label] = true
			}
			for _, want := range test.want {
				if !labels[want] {
					t.Errorf("%q not offered; got %v", want, got)
				}
			}
			for _, bad := range test.notWant {
				if labels[bad] {
					t.Errorf("%q offered but should not be; got %v", bad, got)
				}
			}
		})
	}
}

func TestCompletionReplacesTheTypedPrefixOnly(t *testing.T) {
	text := `{"layout": "st`
	got := Complete(text, len(text), nil)
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
	result := text[:got[0].From] + got[0].Insert + text[len(text):]
	if result != `{"layout": "stacked"` {
		t.Fatalf("applying the completion gives %q", result)
	}
}
