package settings

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// update rewrites the golden files from the current build; run it only to
// capture the file format on purpose.
var update = flag.Bool("update", false, "rewrite the testdata/encode-*.golden files")

// goldenCases are settings whose file text must not change when the schema
// code does: the bytes are what users already have on disk.
func goldenCases() []struct {
	name  string
	prefs Settings
} {
	alone := Default()
	alone.Proxy = "http://localhost:8080"

	changed := Default()
	changed.Theme = "light"
	changed.Layout = "side"
	changed.Flat = true
	changed.UIScalePct = 150
	changed.TimeoutSec = 17

	tables := Default()
	tables.Colors = map[string]map[string]string{"light": {"accent": "#ff0000", "bg": "#ffffff"}, "dark": {"accent": "#00ff00"}}
	tables.Shortcuts = map[string]string{"send": "Ctrl+Enter", "find": ""}
	tables.Collections = []OpenCollection{{Folder: "/work/a", Environment: "prod"}, {Environment: "dev"}}

	everything := Default()
	everything.Theme, everything.TextSize, everything.Corners = "mine", "large", "round"
	everything.UIFont, everything.CodeFont = "Inter", "Fira Code"
	everything.Contrast, everything.ResponseView, everything.AutoSave = "high", "raw", false
	everything.Proxy, everything.CACert = "http://p:1", "/c/ca.pem"
	everything.ClientCert, everything.ClientKey = "/c/cert.pem", "/c/key.pem"
	everything.GRPC, everything.SSE = true, true
	everything.Workspace, everything.Environment = "/old", "stage"
	everything.Colors = tables.Colors
	everything.Shortcuts = tables.Shortcuts
	everything.Collections = tables.Collections

	return []struct {
		name  string
		prefs Settings
	}{
		{"defaults", Default()},
		{"first-only", alone},
		{"changed", changed},
		{"tables-and-collections", tables},
		{"sectioned-everything", everything},
	}
}

func TestEncodeIsByteIdentical(t *testing.T) {
	for _, test := range goldenCases() {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join("testdata", "encode-"+test.name+".golden")
			got, err := Encode(test.prefs)
			if err != nil {
				t.Fatal(err)
			}
			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Fatalf("Encode =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

func TestChangedTreatsEmptyAndNilAlike(t *testing.T) {
	s := Default()
	s.Shortcuts = map[string]string{}
	s.Colors = map[string]map[string]string{}
	s.Collections = []OpenCollection{}
	for _, key := range []string{"shortcuts", "colors", "collections"} {
		if Changed(s, key) {
			t.Errorf("an empty %s counts as changed", key)
		}
	}
}

func TestUIScaleZeroIsChanged(t *testing.T) {
	s := Default()
	s.UIScalePct = 0
	if !Changed(s, "uiScale") {
		t.Fatal("0 is not the default 100, so it is a change")
	}
}

func TestResetOfAnUnknownKeyChangesNothing(t *testing.T) {
	s := Default()
	s.Layout = "side"
	want := s
	Reset(&s, "nope")
	if !reflect.DeepEqual(s, want) {
		t.Fatalf("Reset(nope) changed %+v to %+v", want, s)
	}
	if Changed(s, "nope") {
		t.Fatal("an unknown key is changed")
	}
}
