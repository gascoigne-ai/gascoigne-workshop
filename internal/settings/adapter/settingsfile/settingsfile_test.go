package settingsfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content *string // nil: no file
		want    settings.Settings
		wantErr bool
	}{
		{"missing file gives defaults", nil, settings.Default(), false},
		{"full file", ptr(`{"theme":"light","textSize":"large"}`), settings.Settings{AutoSave: true, Theme: "light", TextSize: "large", Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"saved layout", ptr(`{"layout":"stacked"}`), settings.Settings{AutoSave: true, Theme: "dark", TextSize: "medium", Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"saved timeout", ptr(`{"timeoutSec":15}`), settings.Settings{AutoSave: true, Theme: "dark", TextSize: "medium", Layout: "stacked", Contrast: "normal", TimeoutSec: 15, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"missing field keeps default", ptr(`{"theme":"light"}`), settings.Settings{AutoSave: true, Theme: "light", TextSize: "medium", Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"unknown values pass through", ptr(`{"theme":"neon"}`), settings.Settings{AutoSave: true, Theme: "neon", TextSize: "medium", Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"broken json gives defaults and an error", ptr(`{"theme":`), settings.Default(), true},
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
	want := settings.Settings{AutoSave: true, Theme: "light", TextSize: "small"}
	if err := Save(path, want); err != nil {
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
		if err := Save(filepath.Join(blocker, "settings.json"), settings.Default()); err == nil {
			t.Fatal("want an error when a file sits where the directory must go")
		}
	})
	t.Run("temp file cannot be written", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.Mkdir(path+".tmp", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Save(path, settings.Default()); err == nil {
			t.Fatal("want an error when the temp path is a directory")
		}
	})
	t.Run("target cannot be replaced", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.MkdirAll(filepath.Join(path, "inside"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := Save(path, settings.Default()); err == nil {
			t.Fatal("want an error when the target is a non-empty directory")
		}
	})
}

func TestColorsAndContrastRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := settings.Default()
	want.Contrast = "high"
	want.Colors = map[string]map[string]string{"dark": {"selection": "#336699"}, "light": {"accent": "#ffcc00"}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v err %v", got, err)
	}
	if def := settings.Default(); def.Contrast != "normal" || def.Colors != nil {
		t.Fatalf("defaults = %+v", def)
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
	if err := Save(p, s); err != nil {
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
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	if got, _ := Load(p); got.Corners != "round" || !got.Flat {
		t.Fatalf("round trip: %+v", got)
	}
}

func TestResponseViewDefaultsToCompactAndRoundTrips(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(p)
	if err != nil || s.ResponseView != "compact" {
		t.Fatalf("old file: %+v, %v", s, err)
	}
	s.ResponseView = "raw"
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	if got, _ := Load(p); got.ResponseView != "raw" {
		t.Fatalf("round trip: %+v", got)
	}
}

func TestStatChangesWhenTheFileIsWrittenAndIsZeroWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := Files{}
	if got := store.Stat(path); got != (settings.FileStamp{}) {
		t.Fatalf("missing file stamp = %+v, want zero", got)
	}
	if err := store.WriteText(path, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	first := store.Stat(path)
	if !first.Exists || first.Size != 2 {
		t.Fatalf("stamp = %+v", first)
	}
	if err := store.WriteText(path, []byte(`{"theme":"light"}`)); err != nil {
		t.Fatal(err)
	}
	if store.Stat(path) == first {
		t.Fatal("stamp must change when the file changes")
	}
	if text, err := store.ReadText(path); err != nil || string(text) != `{"theme":"light"}` {
		t.Fatalf("ReadText = %q, %v", text, err)
	}
}
