package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content *string // nil: no file
		want    Settings
		wantErr bool
	}{
		{"missing file gives defaults", nil, Default(), false},
		{"full file", ptr(`{"theme":"light","textSize":"large"}`), Settings{Theme: "light", TextSize: "large", Layout: "side", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft"}, false},
		{"saved layout", ptr(`{"layout":"stacked"}`), Settings{Theme: "dark", TextSize: "medium", Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft"}, false},
		{"saved timeout", ptr(`{"timeoutSec":15}`), Settings{Theme: "dark", TextSize: "medium", Layout: "side", Contrast: "normal", TimeoutSec: 15, UIScalePct: 100, Corners: "soft"}, false},
		{"missing field keeps default", ptr(`{"theme":"light"}`), Settings{Theme: "light", TextSize: "medium", Layout: "side", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft"}, false},
		{"unknown values pass through", ptr(`{"theme":"neon"}`), Settings{Theme: "neon", TextSize: "medium", Layout: "side", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft"}, false},
		{"broken json gives defaults and an error", ptr(`{"theme":`), Default(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if tt.content != nil {
				if err := os.WriteFile(path, []byte(*tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := Load(path)
			if (err != nil) != tt.wantErr || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v err %v; want %+v wantErr %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestSaveRoundTripCreatesDirAndLeavesNoTemp(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "cfg")
	path := filepath.Join(dir, "settings.json")
	want := Settings{Theme: "light", TextSize: "small"}
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v err %v, want %+v", got, err, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("want only settings.json, got %d entries", len(entries))
	}
}

func ptr(s string) *string { return &s }

func TestDefaultPathFollowsTheUserConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	want := filepath.Join(dir, "gascoigne-workshop", "settings.json")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath = %q, want %q", got, want)
	}
}

func TestSaveFailuresAreReported(t *testing.T) {
	t.Run("directory cannot be created", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(blocker, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Default().Save(filepath.Join(blocker, "settings.json")); err == nil {
			t.Fatal("want an error when a file sits where the directory must go")
		}
	})
	t.Run("temp file cannot be written", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.Mkdir(path+".tmp", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Default().Save(path); err == nil {
			t.Fatal("want an error when the temp path is a directory")
		}
	})
	t.Run("target cannot be replaced", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.MkdirAll(filepath.Join(path, "inside"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Default().Save(path); err == nil {
			t.Fatal("want an error when the target is a non-empty directory")
		}
	})
}

func TestColorsAndContrastRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := Default()
	want.Contrast = "high"
	want.Colors = map[string]map[string]string{"dark": {"selection": "#336699"}, "light": {"accent": "#ffcc00"}}
	if err := want.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v err %v", got, err)
	}
	if def := Default(); def.Contrast != "normal" || def.Colors != nil {
		t.Fatalf("defaults = %+v", def)
	}
}

func TestUIScaleDefaultsToTheDisplayScaleAndIsBounded(t *testing.T) {
	if got := Default().UIScale(); got != 1 {
		t.Fatalf("default %v", got)
	}
	for _, tt := range []struct {
		pct  int
		want float64
	}{{0, 1}, {100, 1}, {150, 1.5}, {5, 0.5}, {9000, 3}} {
		if got := (Settings{UIScalePct: tt.pct}).UIScale(); got != tt.want {
			t.Errorf("%d%% -> %v, want %v", tt.pct, got, tt.want)
		}
	}
}

func TestUIScaleRoundTripsAndOldFilesKeepTheDefault(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(p)
	if err != nil || s.UIScalePct != 100 {
		t.Fatalf("old file: %+v, %v", s, err)
	}
	s.UIScalePct = 125
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	if got, _ := Load(p); got.UIScalePct != 125 {
		t.Fatalf("round trip: %+v", got)
	}
}

func TestCornersAndFlatRoundTripAndOldFilesKeepTheDashboardLook(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(p)
	if err != nil || s.Corners != "soft" || s.Flat {
		t.Fatalf("old file: %+v, %v", s, err)
	}
	s.Corners, s.Flat = "round", true
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	if got, _ := Load(p); got.Corners != "round" || !got.Flat {
		t.Fatalf("round trip: %+v", got)
	}
}
