package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func writeTheme(t *testing.T, dir, file, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUserThemeFilesBuildOnABuiltInThemeAndAreListed(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "ocean.json", `{"name": "Ocean", "extends": "light", "colors": {"accent": "#115577"}, "controls": {"button": {"radius": 12}}}`)
	writeTheme(t, dir, "plain.json", `{"colors": {"bg": "#101010"}}`)
	writeTheme(t, dir, "notes.txt", `not a theme`)
	themes, problems := loadUserThemes(dir)
	if len(problems) != 0 || len(themes) != 2 {
		t.Fatalf("themes %v problems %v", themes, problems)
	}
	ocean := themes["ocean"]
	if ocean.label != "Ocean" || ocean.build().Color(widget.Accent) != widget.RGB(0x115577) {
		t.Fatalf("ocean: %+v", ocean)
	}
	if ocean.build().Color(widget.Bg) != lightTheme().Color(widget.Bg) || ocean.build().Style("button").Radius != 12 {
		t.Fatal("it builds on the light theme and overrides its button")
	}
	if themes["plain"].build().Color(widget.Text) != darkTheme().Color(widget.Text) || themes["plain"].label != "plain" {
		t.Fatal("no extends means dark, and the label falls back to the file name")
	}
}

func TestBrokenUserThemeIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "bad.json", `{"colors": {"bg": "nope"}}`)
	writeTheme(t, dir, "orphan.json", `{"extends": "nonexistent"}`)
	writeTheme(t, dir, "good.json", `{"colors": {"accent": "#223344"}}`)
	themes, problems := loadUserThemes(dir)
	if len(themes) != 1 || themes["good"].label != "good" || len(problems) != 2 {
		t.Fatalf("themes %v problems %v", themes, problems)
	}
	if themes, problems := loadUserThemes(filepath.Join(dir, "missing")); len(themes) != 0 || len(problems) != 0 {
		t.Fatalf("a missing folder is no problem: %v %v", themes, problems)
	}
}

func TestSettingsCanSelectAUserThemeAndFallBackWhenItVanishes(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "ocean.json", `{"extends": "light", "colors": {"accent": "#115577"}}`)
	a, err := NewApp(Options{ThemesDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, c := range allThemeChoices() {
		ids = append(ids, c.id)
	}
	if len(ids) != 3 || ids[0] != "dark" || ids[1] != "light" || ids[2] != "ocean" {
		t.Fatalf("choices %v", ids)
	}
	s := settings.Default()
	s.Theme = "ocean"
	if ThemeWith(s).Color(widget.Accent) != widget.RGB(0x115577) {
		t.Fatal("the user theme is selected")
	}
	s.Theme = "gone"
	if ThemeWith(s).Color(widget.Bg) != darkTheme().Color(widget.Bg) {
		t.Fatal("a missing theme falls back to dark")
	}
	_ = a
	if _, err := NewApp(Options{}); err != nil { // no folder: only the built-ins
		t.Fatal(err)
	}
	if len(allThemeChoices()) != 2 {
		t.Fatalf("choices after a start without a themes folder: %d", len(allThemeChoices()))
	}
}
