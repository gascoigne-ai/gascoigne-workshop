package settings

import (
	"os"
	"path/filepath"
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
		{"full file", ptr(`{"theme":"light","textSize":"large"}`), Settings{Theme: "light", TextSize: "large"}, false},
		{"missing field keeps default", ptr(`{"theme":"light"}`), Settings{Theme: "light", TextSize: "medium"}, false},
		{"unknown values pass through", ptr(`{"theme":"neon"}`), Settings{Theme: "neon", TextSize: "medium"}, false},
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
			if (err != nil) != tt.wantErr || got != tt.want {
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
	if err != nil || got != want {
		t.Fatalf("got %+v err %v, want %+v", got, err, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("want only settings.json, got %d entries", len(entries))
	}
}

func ptr(s string) *string { return &s }
