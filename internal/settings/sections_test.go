package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func sectioned(t *testing.T, s Settings) map[string]map[string]any {
	t.Helper()
	text, err := Minimal(s)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]map[string]any
	if err := json.Unmarshal(text, &got); err != nil {
		t.Fatalf("not an object of objects: %v\n%s", err, text)
	}
	return got
}

func TestTheFileGroupsSettingsBySection(t *testing.T) {
	s := Default()
	s.Layout, s.TimeoutSec, s.AutoSave = "stacked", 30, false
	s.Shortcuts = map[string]string{"send": "Ctrl+Enter"}
	s.Colors = map[string]map[string]string{"dark": {"accent": "#ff0000"}}
	got := sectioned(t, s)
	want := map[string]map[string]any{
		"appearance": {"layout": "stacked"},
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
	nested := `{"appearance": {"layout": "stacked", "theme": "light"}, "requests": {"timeoutSec": 7}, "general": {"autoSave": false}}`
	flat := `{"layout": "stacked", "theme": "light", "timeoutSec": 7, "autoSave": false}`
	a, problems := Parse([]byte(nested), Default())
	if len(problems) != 0 {
		t.Fatalf("problems: %+v", problems)
	}
	b, problems := Parse([]byte(flat), Default())
	if len(problems) != 0 {
		t.Fatalf("legacy flat file refused: %+v", problems)
	}
	if !reflect.DeepEqual(a, b) || a.Layout != "stacked" || a.TimeoutSec != 7 || a.AutoSave {
		t.Fatalf("nested %+v\nflat   %+v", a, b)
	}
}

func TestEverySettingSurvivesAWriteAndARead(t *testing.T) {
	s := Default()
	s.Theme, s.TextSize, s.UIScalePct, s.UIFont, s.CodeFont = "light", "large", 130, "Go", "Go Mono"
	s.Corners, s.Flat, s.Contrast, s.Layout, s.ResponseView = "round", true, "high", "stacked", "raw"
	s.Proxy, s.CACert, s.ClientCert, s.ClientKey, s.TimeoutSec = "http://p", "/a", "/b", "/c", 9
	s.AutoSave, s.Workspace, s.Environment = false, "/w", "Dev"
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
		{"values of an enum inside a section", `{"appearance": {"layout": "`, []string{"side", "stacked"}, nil},
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
