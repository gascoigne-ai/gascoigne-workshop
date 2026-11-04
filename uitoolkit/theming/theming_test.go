package theming

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestThemeFollowsPrefsAndFallsBack(t *testing.T) {
	c := NewCatalog(Builtins())
	light := c.Theme(Prefs{Theme: "light"})
	if light.Color(core.Bg) == c.Theme(Prefs{Theme: "dark"}).Color(core.Bg) {
		t.Fatal("light and dark are the same")
	}
	if c.Theme(Prefs{Theme: "gone"}).Color(core.Bg) != c.Theme(Prefs{Theme: "dark"}).Color(core.Bg) {
		t.Fatal("an unknown theme must fall back to the first built-in")
	}
	if c.ThemeID("gone") != "dark" || c.ThemeID("light") != "light" {
		t.Fatal("ThemeID")
	}
	flat := c.Theme(Prefs{Flat: true, Corners: "round"})
	if !flat.NoGradients || flat.RadiusScale != 1.5 {
		t.Fatalf("flat %v scale %v", flat.NoGradients, flat.RadiusScale)
	}
}

func TestOverridesNeedContrastAndHighContrastRaisesTheBar(t *testing.T) {
	c := NewCatalog(Builtins())
	p := Prefs{Theme: "dark", Colors: map[string]map[string]string{"dark": {
		string(core.Selection): "#ffff00", // readable under the text color? decided by the ratio
		string(core.Accent):    "#nothex",
		"not-a-token":          "#123456",
	}}}
	th := c.Theme(p)
	base := c.Theme(Prefs{Theme: "dark"})
	if th.Color(core.Accent) != base.Color(core.Accent) {
		t.Fatal("a bad hex was applied")
	}
	high := c.Theme(Prefs{Theme: "dark", Contrast: "high"})
	for _, cc := range CustomColors {
		if r := ContrastRatio(high.Color(cc.Token), high.Color(cc.Text)); r < HighRatio-0.01 {
			t.Errorf("%s: %.2f:1 under high contrast", cc.Label, r)
		}
	}
}

func TestUserThemesExtendBuiltinsAndAreListed(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ocean.json"), []byte(`{"extends":"light","colors":{"accent":"#115577"}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"colors":{"bg":"nope"}}`), 0o644)
	c := NewCatalog(Builtins())
	themes, problems := c.LoadUser(dir)
	if len(themes) != 1 || len(problems) != 1 {
		t.Fatalf("themes %v problems %v", themes, problems)
	}
	c.SetUser(themes)
	var ids []string
	for _, ch := range c.All() {
		ids = append(ids, ch.ID)
	}
	if len(ids) != 3 || ids[2] != "ocean" {
		t.Fatalf("%v", ids)
	}
	if c.Theme(Prefs{Theme: "ocean"}).Color(core.Accent) != core.RGB(0x115577) {
		t.Fatal("user theme not applied")
	}
	c.SetUser(nil)
	if len(c.All()) != 2 {
		t.Fatal("user themes not cleared")
	}
}
