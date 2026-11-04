package appearance

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
	themes, problems := LoadUserThemes(dir)
	if len(problems) != 0 || len(themes) != 2 {
		t.Fatalf("themes %v problems %v", themes, problems)
	}
	ocean := themes["ocean"]
	if ocean.Label != "Ocean" || ocean.Build().Color(widget.Accent) != widget.RGB(0x115577) {
		t.Fatalf("ocean: %+v", ocean)
	}
	if ocean.Build().Color(widget.Bg) != LightTheme().Color(widget.Bg) || ocean.Build().Style("button").Radius != 12 {
		t.Fatal("it builds on the light theme and overrides its button")
	}
	if themes["plain"].Build().Color(widget.Text) != DarkTheme().Color(widget.Text) || themes["plain"].Label != "plain" {
		t.Fatal("no extends means dark, and the label falls back to the file name")
	}
}

func TestBrokenUserThemeIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "bad.json", `{"colors": {"bg": "nope"}}`)
	writeTheme(t, dir, "orphan.json", `{"extends": "nonexistent"}`)
	writeTheme(t, dir, "good.json", `{"colors": {"accent": "#223344"}}`)
	themes, problems := LoadUserThemes(dir)
	if len(themes) != 1 || themes["good"].Label != "good" || len(problems) != 2 {
		t.Fatalf("themes %v problems %v", themes, problems)
	}
	if themes, problems := LoadUserThemes(filepath.Join(dir, "missing")); len(themes) != 0 || len(problems) != 0 {
		t.Fatalf("a missing folder is no problem: %v %v", themes, problems)
	}
}

func TestSettingsCanSelectAUserThemeAndFallBackWhenItVanishes(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "ocean.json", `{"extends": "light", "colors": {"accent": "#115577"}}`)
	themes, _ := LoadUserThemes(dir)
	SetUserThemes(themes)
	t.Cleanup(func() { SetUserThemes(nil) })
	var ids []string
	for _, c := range AllThemeChoices() {
		ids = append(ids, c.ID)
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
	if ThemeWith(s).Color(widget.Bg) != DarkTheme().Color(widget.Bg) {
		t.Fatal("a missing theme falls back to dark")
	}
	none, _ := LoadUserThemes("") // no folder: only the built-ins
	SetUserThemes(none)
	if len(AllThemeChoices()) != 2 {
		t.Fatalf("choices after a start without a themes folder: %d", len(AllThemeChoices()))
	}
}
