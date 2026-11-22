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
		{"enum value not allowed", `{"layout": "diagonal"}`, "layout", "stacked"},
		{"number below range", `{"uiScale": 10}`, "uiScale", "between"},
		{"number above range", `{"uiScale": 900}`, "uiScale", "between"},
		{"shortcuts must be an object of text", `{"shortcuts": {"send": 1}}`, "shortcuts", "text"},
		{"colors must be nested objects", `{"colors": {"dark": "red"}}`, "colors", "object"},
		{"open collections", `{"general": {"collections": [{"folder": "/a", "environment": "Dev"}, {}]}}`, "", ""},
		{"open collections must be a list", `{"general": {"collections": "/a"}}`, "collections", "list"},
		{"open collections have no other keys", `{"general": {"collections": [{"dir": "/a"}]}}`, "collections", "folder and environment"},
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
	good.Layout = "side"
	got, problems := Parse([]byte(`{"layout": "diagonal"`), good)
	if len(problems) == 0 || got.Layout != "side" {
		t.Fatalf("got %+v with problems %+v", got, problems)
	}
	got, problems = Parse([]byte(`{"layout": "stacked", "uiScale": 150}`), good)
	if len(problems) != 0 || got.Layout != "stacked" || got.UIScalePct != 150 {
		t.Fatalf("got %+v with problems %+v", got, problems)
	}
}

func TestChangedAndResetUseTheDefaults(t *testing.T) {
	s := Default()
	if Changed(s, "layout") {
		t.Fatal("a default value is marked changed")
	}
	s.Layout = "side"
	s.Shortcuts = map[string]string{"send": "Ctrl+Enter"}
	if !Changed(s, "layout") || !Changed(s, "shortcuts") {
		t.Fatal("a changed value is not marked")
	}
	for _, key := range []string{"layout", "shortcuts"} {
		if err := Reset(&s, key); err != nil {
			t.Fatal(err)
		}
	}
	if s.Layout != Default().Layout || s.Shortcuts != nil || Changed(s, "layout") || Changed(s, "shortcuts") {
		t.Fatalf("after reset: %+v", s)
	}
}

func TestMarshalLeavesOutWhatIsAtItsDefault(t *testing.T) {
	s := Default()
	s.Layout = "side"
	got := sectioned(t, s)
	if len(got) != 1 || len(got["appearance"]) != 1 || got["appearance"]["layout"] != "side" {
		t.Fatalf("file = %v", got)
	}
}

func TestCompletionReplacesTheTypedPrefixOnly(t *testing.T) {
	text := `{"appearance": {"layout": "st`
	got := Complete(text, len(text), nil)
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
	result := text[:got[0].From] + got[0].Insert + text[len(text):]
	if result != `{"appearance": {"layout": "stacked"` {
		t.Fatalf("applying the completion gives %q", result)
	}
}

func TestValueAndWithReadAndWriteASettingByItsKey(t *testing.T) {
	s := Default()
	tests := []struct {
		key   string
		value Value
		check func(Settings) bool
	}{
		{"layout", Text("side"), func(prefs Settings) bool { return prefs.Layout == "side" }},
		{"uiScale", Number(150), func(prefs Settings) bool { return prefs.UIScalePct == 150 }},
		{"flat", Flag(true), func(prefs Settings) bool { return prefs.Flat }},
		{"proxy", Text("http://localhost:8080"), func(prefs Settings) bool { return prefs.Proxy == "http://localhost:8080" }},
	}
	for _, test := range tests {
		got, err := With(s, test.key, test.value)
		if err != nil || !test.check(got) {
			t.Errorf("With(%s, %v) = %+v, %v", test.key, test.value, got, err)
		}
		if value, _ := Get(got, test.key); value != test.value {
			t.Errorf("Get(%s) = %v, want %v", test.key, value, test.value)
		}
	}
	if _, ok := Get(s, "nope"); ok {
		t.Error("an unknown key has a value")
	}
	if got, _ := With(s, "nope", Number(1)); !reflect.DeepEqual(got, s) {
		t.Error("an unknown key changed the settings")
	}
}

func TestEveryKindSaysWhatItExpects(t *testing.T) {
	tests := []struct{ key, want string }{
		{"layout", "one of stacked, side"},
		{"flat", "true or false"},
		{"uiScale", "a whole number from 50 to 300"},
		{"proxy", "text"},
		{"theme", "dark, light, or the id of a theme file"},
		{"shortcuts", "an object of action id to key text"},
		{"colors", "an object of theme id to token to #rrggbb"},
		{"collections", "a list of objects with folder and environment text"},
	}
	for _, test := range tests {
		field, ok := FieldFor(test.key)
		if !ok {
			t.Fatalf("no field %s", test.key)
		}
		if got := Expectation(field); got != test.want {
			t.Errorf("%s: Expectation = %q, want %q", test.key, got, test.want)
		}
	}
}

func TestCompletionsCarryTheDescriptionAndWhatIsExpected(t *testing.T) {
	text := `{"appearance": { "la`
	got := Complete(text, len(text), nil)
	if len(got) != 1 || got[0].Label != "layout" {
		t.Fatalf("got %+v", got)
	}
	for _, want := range []string{"Place the request and response panes", "one of stacked, side"} {
		if !strings.Contains(got[0].Detail, want) {
			t.Errorf("the detail %q lacks %q", got[0].Detail, want)
		}
	}
	values := Complete(`{"appearance": {"layout": "`, len(`{"appearance": {"layout": "`), nil)
	if len(values) == 0 || !strings.Contains(values[0].Detail, "Request and response") {
		t.Fatalf("a value completion names its setting: %+v", values)
	}
}

func TestTheCertificateSettingsAreFilesAndValidateAsText(t *testing.T) {
	for _, key := range []string{"caCert", "clientCert", "clientKey"} {
		field, _ := FieldFor(key)
		if field.Kind != File {
			t.Errorf("%s is %v, want a file setting", key, field.Kind)
		}
		if Expectation(field) != "the path of a file" {
			t.Errorf("%s expects %q", key, Expectation(field))
		}
	}
	if problems := Validate([]byte(`{"caCert": 3}`)); len(problems) != 1 || !strings.Contains(problems[0].Message, "text") {
		t.Fatalf("problems = %+v", problems)
	}
	if problems := Validate([]byte(`{"caCert": "/etc/ssl/ca.pem"}`)); len(problems) != 0 {
		t.Fatalf("problems = %+v", problems)
	}
}

func TestDefaultLayoutIsStacked(t *testing.T) {
	if got := Default().Layout; got != "stacked" {
		t.Fatalf("default layout = %q", got)
	}
}

// Completing an enum value sorted the schema's own list of allowed values,
// so the settings menu then showed "Side by side" before "Stacked".
func TestCompletingAnEnumValueKeepsTheSchemaOrder(t *testing.T) {
	field, _ := FieldFor("layout")
	before := append([]string(nil), field.Allowed...)
	text := `{"appearance": {"layout": "`
	Complete(text, len(text), nil)
	if after, _ := FieldFor("layout"); !reflect.DeepEqual(after.Allowed, before) {
		t.Fatalf("allowed layouts are %v after a completion, were %v", after.Allowed, before)
	}
}
