package settingsfile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/userdir"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content *string // nil: no file
		want    settings.Settings
		wantErr bool
	}{
		{"missing file gives defaults", nil, settings.Default(), false},
		{"full file", ptr(`{"theme":"light","textSize":"large"}`), settings.Settings{AutoSave: true, Theme: "light", FontSize: 16, SmallFontSize: 14, CodeFontSize: 15, Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"saved layout", ptr(`{"layout":"stacked"}`), settings.Settings{AutoSave: true, Theme: "dark", FontSize: 14, SmallFontSize: 12, CodeFontSize: 13, Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"saved timeout", ptr(`{"timeoutSec":15}`), settings.Settings{AutoSave: true, Theme: "dark", FontSize: 14, SmallFontSize: 12, CodeFontSize: 13, Layout: "stacked", Contrast: "normal", TimeoutSec: 15, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"missing field keeps default", ptr(`{"theme":"light"}`), settings.Settings{AutoSave: true, Theme: "light", FontSize: 14, SmallFontSize: 12, CodeFontSize: 13, Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
		{"unknown values pass through", ptr(`{"theme":"neon"}`), settings.Settings{AutoSave: true, Theme: "neon", FontSize: 14, SmallFontSize: 12, CodeFontSize: 13, Layout: "stacked", Contrast: "normal", TimeoutSec: 60, UIScalePct: 100, Corners: "soft", ResponseView: "compact"}, false},
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
			got, err := disk.Load(path)
			if (err != nil) != tt.wantErr || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v err %v; want %+v wantErr %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestSaveRoundTripCreatesDirAndLeavesNoTemp(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "cfg")
	path := filepath.Join(dir, "settings.json")
	want := settings.Settings{AutoSave: true, Theme: "light", FontSize: 13, SmallFontSize: 11, CodeFontSize: 12}
	if err := disk.Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := disk.Load(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v err %v, want %+v", got, err, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("want only settings.json, got %d entries", len(entries))
	}
}

func ptr(text string) *string { return &text }

func TestDefaultPathFollowsTheUserConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	config, err := os.UserConfigDir() // XDG_CONFIG_HOME on Linux, under HOME on macOS
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(config, userdir.Name, "settings.json")
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
		if err := disk.Save(filepath.Join(blocker, "settings.json"), settings.Default()); err == nil {
			t.Fatal("want an error when a file sits where the directory must go")
		}
	})
	t.Run("temp file cannot be created", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(dir, 0o700)
		if os.Getuid() == 0 {
			t.Skip("root writes into a read-only directory")
		}
		if err := disk.Save(filepath.Join(dir, "settings.json"), settings.Default()); err == nil {
			t.Fatal("want an error when the directory cannot be written")
		}
	})
	t.Run("target cannot be replaced", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.MkdirAll(filepath.Join(path, "inside"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := disk.Save(path, settings.Default()); err == nil {
			t.Fatal("want an error when the target is a non-empty directory")
		}
	})
}

func TestColorsAndContrastRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	want := settings.Default()
	want.Contrast = "high"
	want.Colors = map[string]map[string]string{"dark": {"selection": "#336699"}, "light": {"accent": "#ffcc00"}}
	if err := disk.Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := disk.Load(path)
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
	s, err := disk.Load(p)
	if err != nil || s.UIScalePct != 100 {
		t.Fatalf("old file: %+v, %v", s, err)
	}
	s.UIScalePct = 125
	if err := disk.Save(p, s); err != nil {
		t.Fatal(err)
	}
	if got, _ := disk.Load(p); got.UIScalePct != 125 {
		t.Fatalf("round trip: %+v", got)
	}
}

func TestCornersAndFlatRoundTripAndOldFilesKeepTheDashboardLook(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := disk.Load(p)
	if err != nil || s.Corners != "soft" || s.Flat {
		t.Fatalf("old file: %+v, %v", s, err)
	}
	s.Corners, s.Flat = "round", true
	if err := disk.Save(p, s); err != nil {
		t.Fatal(err)
	}
	if got, _ := disk.Load(p); got.Corners != "round" || !got.Flat {
		t.Fatalf("round trip: %+v", got)
	}
}

func TestResponseViewDefaultsToCompactAndRoundTrips(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := disk.Load(p)
	if err != nil || s.ResponseView != "compact" {
		t.Fatalf("old file: %+v, %v", s, err)
	}
	s.ResponseView = "raw"
	if err := disk.Save(p, s); err != nil {
		t.Fatal(err)
	}
	if got, _ := disk.Load(p); got.ResponseView != "raw" {
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

// A settings file that did not parse was replaced by defaults on the next
// save, so one wrong key cost the user the whole file. The broken text is
// now kept beside it as settings.json.bak.
func TestSaveKeepsAFileThatDoesNotParseAsBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	broken := `{"theme": "light", "timeoutSec": "soon"}`
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := disk.Save(path, settings.Default()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path + ".bak")
	if err != nil || string(got) != broken {
		t.Fatalf("backup %q, %v; want the broken text", got, err)
	}
}

func TestSaveDoesNotBackUpAFileThatParses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := disk.Save(path, settings.Default()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".bak"); err == nil {
		t.Error("a settings file that parsed must not leave a backup")
	}
}

// A file that parses but holds a value outside the schema's range (a
// timeout of 99999999999 s) was used as written. It is now clamped to
// the nearest allowed value and the load reports which keys it changed.
func TestLoadClampsOutOfRangeValuesAndSaysSo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"timeoutSec": 99999999999, "uiScale": 5, "theme": "light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := disk.Load(path)
	var clamped *settings.ClampedError
	if !errors.As(err, &clamped) {
		t.Fatalf("err = %v, want a ClampedError", err)
	}
	if got.TimeoutSec != 3600 || got.UIScalePct != settings.MinUIScalePct || got.Theme != "light" {
		t.Errorf("settings %+v, want timeout 3600, scale %d, theme kept", got, settings.MinUIScalePct)
	}
	for _, key := range []string{"timeoutSec", "uiScale"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("message %q does not name %s", err, key)
		}
	}
}

// disk is the store under test.
var disk Files
