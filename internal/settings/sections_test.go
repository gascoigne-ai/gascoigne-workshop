package settings

import (
	"encoding/json"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/openset"
	"reflect"
	"strings"
	"testing"
)

func sectioned(t *testing.T, prefs Settings) map[string]map[string]any {
	t.Helper()
	text, err := Minimal(prefs)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(text, &top); err != nil {
		t.Fatalf("not an object: %v\n%s", err, text)
	}
	if version := string(top["version"]); version != "3" { // the file's own key, beside the sections
		t.Fatalf("version %q, want 3:\n%s", version, text)
	}
	delete(top, "version")
	got := map[string]map[string]any{}
	for name, raw := range top {
		var section map[string]any
		if err := json.Unmarshal(raw, &section); err != nil {
			t.Fatalf("section %s is not an object: %v\n%s", name, err, text)
		}
		got[name] = section
	}
	return got
}

func TestTheFileGroupsSettingsBySection(t *testing.T) {
	s := Default()
	s.Layout, s.TimeoutSec, s.AutoSave = "side", 30, false
	s.Shortcuts = map[string]string{"send": "Ctrl+Enter"}
	s.Colors = map[string]map[string]string{"dark": {"accent": "#ff0000"}}
	got := sectioned(t, s)
	want := map[string]map[string]any{
		"appearance": {"layout": "side"},
		"requests":   {"timeoutSec": float64(30)},
		"general":    {"autoSave": false},
		"shortcuts":  {"send": "Ctrl+Enter"},
		"colors":     {"dark": map[string]any{"accent": "#ff0000"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("file = %v\nwant %v", got, want)
	}
}

func TestTheSectionsComeInAStableOrderWithTheirKeysInSchemaOrder(t *testing.T) {
	s := Default()
	s.Corners, s.Theme, s.Proxy, s.TimeoutSec = "round", "light", "http://p", 5
	text, _ := Minimal(s)
	order := func(needles ...string) []int {
		var at []int
		for _, n := range needles {
			at = append(at, strings.Index(string(text), n))
		}
		return at
	}
	positions := order(`"appearance"`, `"requests"`)
	if positions[0] < 0 || positions[0] > positions[1] {
		t.Fatalf("sections out of order:\n%s", text)
	}
	keys := order(`"theme"`, `"corners"`)
	if keys[0] < 0 || keys[0] > keys[1] {
		t.Fatalf("keys not in schema order (theme before corners):\n%s", text)
	}
}

func TestSectionedAndLegacyFlatFilesReadAlike(t *testing.T) {
	nested := `{"appearance": {"layout": "side", "theme": "light"}, "requests": {"timeoutSec": 7}, "general": {"autoSave": false}}`
	flat := `{"layout": "side", "theme": "light", "timeoutSec": 7, "autoSave": false}`
	a, problems := Parse([]byte(nested), Default())
	if len(problems) != 0 {
		t.Fatalf("problems: %+v", problems)
	}
	b, problems := Parse([]byte(flat), Default())
	if len(problems) != 0 {
		t.Fatalf("legacy flat file refused: %+v", problems)
	}
	if !reflect.DeepEqual(a, b) || a.Layout != "side" || a.TimeoutSec != 7 || a.AutoSave {
		t.Fatalf("nested %+v\nflat   %+v", a, b)
	}
}

func TestEverySettingSurvivesAWriteAndARead(t *testing.T) {
	s := Default()
	s.Theme, s.FontSize, s.UIScalePct, s.UIFont, s.CodeFont = "light", 16, 130, "Go", "Go Mono"
	s.Corners, s.Flat, s.Contrast, s.Layout, s.ResponseView = "round", true, "high", "side", "raw"
	s.Proxy, s.CACert, s.ClientCert, s.ClientKey, s.TimeoutSec = "http://p", "/a", "/b", "/c", 9
	s.AutoSave, s.Workspace, s.Environment = false, "/w", "Dev"
	s.Collections = []openset.OpenCollection{{Folder: "/a", Environment: "Prod"}, {Folder: "/b"}}
	s.Shortcuts = map[string]string{"send": "F5"}
	s.Colors = map[string]map[string]string{"dark": {"accent": "#112233"}}
	text, err := Minimal(s)
	if err != nil {
		t.Fatal(err)
	}
	back, problems := Parse(text, Default())
	if len(problems) != 0 || !reflect.DeepEqual(back, s) {
		t.Fatalf("problems %+v\nwrote %s\nread  %+v\nwant  %+v", problems, text, back, s)
	}
}

func TestValidateChecksKeysInsideTheirSection(t *testing.T) {
	tests := []struct {
		name, text, wantKey, wantIn string
	}{
		{"fine", `{"appearance": {"theme": "dark"}}`, "", ""},
		{"unknown key in a section", `{"appearance": {"colour": 1}}`, "colour", "unknown setting"},
		{"a key in the wrong section", `{"appearance": {"proxy": "x"}}`, "proxy", "belongs in requests"},
		{"a bad value inside a section", `{"requests": {"timeoutSec": "ten"}}`, "timeoutSec", "number"},
		{"a section that is not an object", `{"appearance": 3}`, "appearance", "object"},
		{"an unknown section", `{"look": {"theme": "dark"}}`, "look", "unknown setting"},
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
				t.Fatalf("problems = %+v, want %s containing %q", problems, test.wantKey, test.wantIn)
			}
		})
	}
}

func TestAProblemInsideASectionPointsAtItsKeyInTheText(t *testing.T) {
	text := "{\n  \"appearance\": {\n    \"theme\": \"dark\",\n    \"colour\": 1\n  }\n}"
	problems := Validate([]byte(text))
	if len(problems) != 1 || text[problems[0].Start:problems[0].End] != `"colour"` {
		t.Fatalf("problems = %+v", problems)
	}
}

func TestCompletionWalksSectionsThenKeysThenValues(t *testing.T) {
	tests := []struct {
		name, text string
		want       []string
		notWant    []string
	}{
		{"top level offers the sections", `{ "`, []string{"appearance", "requests", "general", "colors", "shortcuts"}, []string{"theme", "layout"}},
		{"a section already there is not offered again", `{"appearance": {}, "`, []string{"requests"}, []string{"appearance"}},
		{"inside a section only its keys", `{"appearance": { "`, []string{"theme", "layout", "corners"}, []string{"proxy", "timeoutSec", "autoSave"}},
		{"inside the requests section", `{"requests": { "`, []string{"proxy", "timeoutSec", "caCert"}, []string{"theme"}},
		{"used keys are not offered again", `{"appearance": {"theme": "dark", "`, []string{"layout"}, []string{"theme"}},
		{"values of an enum inside a section", `{"appearance": {"layout": "`, []string{"stacked", "side"}, nil},
		{"booleans inside a section", `{"general": {"autoSave": `, []string{"true", "false"}, nil},
		{"action ids inside shortcuts", `{"shortcuts": { "`, []string{"send"}, []string{"appearance"}},
		{"no values for free text", `{"requests": {"proxy": "`, nil, []string{"side", "true"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Complete(test.text, len(test.text), []string{"send"})
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
					t.Errorf("%q offered but should not be", bad)
				}
			}
		})
	}
}

// A boolean is not text: completing it after the user typed an opening quote
// must replace that quote with the bare word, not leave "false" as a string,
// which the validation then refuses.
func TestCompletingABooleanAfterAQuoteWritesTheBareWord(t *testing.T) {
	tests := []struct{ text, pick, want string }{
		{`{"general": {"autoSave": "`, "false", `{"general": {"autoSave": false`},
		{`{"general": {"autoSave": "f`, "false", `{"general": {"autoSave": false`},
		{`{"general": {"autoSave": `, "true", `{"general": {"autoSave": true`},
		{`{"appearance": {"layout": "st`, "stacked", `{"appearance": {"layout": "stacked"`},
	}
	for _, test := range tests {
		var chosen *Completion
		for _, c := range Complete(test.text, len(test.text), nil) {
			if c.Label == test.pick {
				chosen = &c
			}
		}
		if chosen == nil {
			t.Errorf("%q: %q not offered", test.text, test.pick)
			continue
		}
		if got := test.text[:chosen.From] + chosen.Insert; got != test.want {
			t.Errorf("%q + %s gives %q, want %q", test.text, test.pick, got, test.want)
		}
		if problems := Validate([]byte(test.want + "}}")); len(problems) != 0 {
			t.Errorf("the completed text is refused: %+v", problems)
		}
	}
}

func TestExperimentalProtocolsAreOffByDefaultAndLiveInTheirOwnSection(t *testing.T) {
	if s := Default(); s.GRPC || s.SSE {
		t.Fatalf("defaults %+v, want gRPC and SSE off", s)
	}
	if got := sectioned(t, Default()); len(got["experimental"]) != 0 {
		t.Fatalf("defaults write an experimental section: %v", got)
	}
	s := Default()
	s.GRPC, s.SSE = true, true
	want := map[string]any{"grpc": true, "sse": true}
	if got := sectioned(t, s)["experimental"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("experimental section = %v, want %v", got, want)
	}
	read, problems := Parse([]byte(`{"experimental": {"sse": true}}`), Default())
	if len(problems) != 0 || !read.SSE || read.GRPC {
		t.Fatalf("read %+v, problems %+v: want only SSE on", read, problems)
	}
}
