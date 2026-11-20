package settings

import (
	"errors"
	"reflect"
	"testing"
)

// sampleFor is a value of the row's kind that differs from the default.
var samples = map[string]Value{
	"theme":        Text("light"),
	"textSize":     Text("large"),
	"uiScale":      Number(150),
	"uiFont":       Text("Inter"),
	"codeFont":     Text("Fira Code"),
	"corners":      Text("round"),
	"flat":         Flag(true),
	"contrast":     Text("high"),
	"layout":       Text("side"),
	"responseView": Text("raw"),
	"autoSave":     Flag(false),
	"colors":       ColorTable{"dark": {"accent": "#00ff00"}},
	"shortcuts":    ShortcutTable{"send": "Ctrl+Enter", "find": ""},
	"timeoutSec":   Number(17),
	"proxy":        Text("http://localhost:8080"),
	"caCert":       Text("/c/ca.pem"),
	"clientCert":   Text("/c/cert.pem"),
	"clientKey":    Text("/c/key.pem"),
	"grpc":         Flag(true),
	"sse":          Flag(true),
	"collections":  Collections{{Folder: "/work/a", Environment: "prod"}, {Environment: "dev"}},
	"workspace":    Text("/old"),
	"environment":  Text("stage"),
}

func TestEveryRowRoundTripsAValue(t *testing.T) {
	for _, field := range Schema {
		t.Run(field.Key, func(t *testing.T) {
			sample, ok := samples[field.Key]
			if !ok {
				t.Fatalf("no sample value for %s", field.Key)
			}
			s := Default()
			if err := Set(&s, field.Key, sample); err != nil {
				t.Fatal(err)
			}
			got, ok := Get(s, field.Key)
			if !ok || !got.Same(sample) {
				t.Fatalf("Get after Set = %v, %v; want %v", got, ok, sample)
			}
			if !Changed(s, field.Key) {
				t.Error("a value that differs from the default is not changed")
			}
			decoded, err := Decode(mustEncode(t, s))
			if err != nil || !reflect.DeepEqual(decoded, s) {
				t.Errorf("Decode(Encode(s)) = %+v, %v; want %+v", decoded, err, s)
			}
			if err := Reset(&s, field.Key); err != nil {
				t.Fatal(err)
			}
			if Changed(s, field.Key) {
				t.Error("still changed after Reset")
			}
			if !reflect.DeepEqual(s, Default()) {
				t.Errorf("after Reset: %+v", s)
			}
		})
	}
}

func mustEncode(t *testing.T, s Settings) []byte {
	t.Helper()
	text, err := Encode(s)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// kindValues is the dynamic type a row of each kind holds.
var kindValues = map[Kind]Value{
	String: Text(""), File: Text(""), Enum: Text(""), Bool: Flag(false), Int: Number(0),
	Shortcuts: ShortcutTable(nil), Colors: ColorTable(nil), CollectionList: Collections(nil),
}

func TestEveryRowHasOneSetterAndItsKindMatchesItsValue(t *testing.T) {
	seen := map[string]bool{}
	for _, field := range Schema {
		if seen[field.Key] {
			t.Errorf("%s has two rows", field.Key)
		}
		seen[field.Key] = true
		value, ok := Get(Default(), field.Key)
		if !ok {
			t.Errorf("%s has no getter", field.Key)
			continue
		}
		if got, want := reflect.TypeOf(value), reflect.TypeOf(kindValues[field.Kind]); got != want {
			t.Errorf("%s is kind %d and holds %v, want %v", field.Key, field.Kind, got, want)
		}
	}
}

func TestSetRefusesAWrongValueTypeAndLeavesSettingsAsTheyWere(t *testing.T) {
	wrong := []Value{Text("x"), Flag(true), Number(1), ShortcutTable{"a": "b"}, ColorTable{"a": {"b": "c"}}, Collections{{Folder: "f"}}}
	for _, field := range Schema {
		for _, value := range wrong {
			if reflect.TypeOf(value) == reflect.TypeOf(kindValues[field.Kind]) {
				continue
			}
			s := Default()
			s.Layout = "side"
			before := s
			if err := Set(&s, field.Key, value); !errors.Is(err, ErrWrongValue) {
				t.Errorf("Set(%s, %T) = %v, want ErrWrongValue", field.Key, value, err)
			}
			if !reflect.DeepEqual(s, before) {
				t.Errorf("Set(%s, %T) changed the settings to %+v", field.Key, value, s)
			}
			got, err := With(before, field.Key, value)
			if !errors.Is(err, ErrWrongValue) || !reflect.DeepEqual(got, before) {
				t.Errorf("With(%s, %T) = %+v, %v", field.Key, value, got, err)
			}
		}
	}
}

func TestSetAndResetRefuseAnUnknownKey(t *testing.T) {
	s := Default()
	before := s
	if err := Set(&s, "nope", Text("x")); !errors.Is(err, ErrUnknownKey) {
		t.Errorf("Set = %v, want ErrUnknownKey", err)
	}
	if err := Reset(&s, "nope"); !errors.Is(err, ErrUnknownKey) {
		t.Errorf("Reset = %v, want ErrUnknownKey", err)
	}
	if got, err := With(s, "nope", Text("x")); !errors.Is(err, ErrUnknownKey) || !reflect.DeepEqual(got, before) {
		t.Errorf("With = %+v, %v", got, err)
	}
	if !reflect.DeepEqual(s, before) {
		t.Errorf("settings changed: %+v", s)
	}
}

func TestGetOfAnUnknownKeyIsFalse(t *testing.T) {
	if value, ok := Get(Default(), "nope"); ok || value != nil {
		t.Fatalf("Get(nope) = %v, %v", value, ok)
	}
}

func TestSetKeepsNoAliasOfTheCallersTable(t *testing.T) {
	s := Default()
	table := ShortcutTable{"send": "Ctrl+Enter"}
	if err := Set(&s, "shortcuts", table); err != nil {
		t.Fatal(err)
	}
	table["send"] = "changed"
	if s.Shortcuts["send"] != "Ctrl+Enter" {
		t.Fatalf("the settings share the caller's map: %v", s.Shortcuts)
	}
}

func BenchmarkChanged(b *testing.B) {
	s := Default()
	for range b.N {
		Changed(s, "layout")
	}
}
