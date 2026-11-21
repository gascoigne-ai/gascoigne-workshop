package theming

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	for _, cc := range CustomColors() {
		if r := core.ContrastRatio(high.Color(cc.Token), high.Color(cc.Text)); r < HighRatio-0.01 {
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

// A folder name with a glob metacharacter made Glob fail and every user
// theme vanish without a trace; the reason is logged at debug now.
func TestLoadUserLogsAnUnreadableFolderPattern(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(previous)

	themes, problems := NewCatalog(Builtins()).LoadUser(filepath.Join(t.TempDir(), "themes[")) // an unclosed [ is ErrBadPattern
	if len(themes) != 0 || len(problems) != 0 {
		t.Errorf("themes %v problems %v, want none", themes, problems)
	}
	if !strings.Contains(logs.String(), filepath.ErrBadPattern.Error()) {
		t.Errorf("log %q does not give the reason %q", logs.String(), filepath.ErrBadPattern)
	}
}

func TestForFallsBackToMediumForAnUnknownTextSize(t *testing.T) {
	c := NewCatalog(Builtins())
	medium := c.For("dark", "medium")
	if got := c.For("dark", "gone"); got.FontSize != medium.FontSize {
		t.Fatalf("unknown size gave %v, medium is %v", got.FontSize, medium.FontSize)
	}
	if c.For("dark", "large").FontSize == medium.FontSize {
		t.Fatal("large is not medium")
	}
}

func TestCornerScaleOfEachIDAndOfAnUnknownOne(t *testing.T) {
	tests := []struct {
		id   string
		want float64
	}{{"square", 0}, {"soft", 1}, {"round", 1.5}, {"gone", 1}, {"", 1}}
	for _, test := range tests {
		if got := CornerScale(test.id); got != test.want {
			t.Errorf("CornerScale(%q) = %v, want %v", test.id, got, test.want)
		}
	}
}

func TestAnOverrideBelowTheRequiredContrastIsIgnored(t *testing.T) {
	c := NewCatalog(Builtins())
	base := c.Theme(Prefs{Theme: "dark"})
	sameAsText := core.FormatHex(base.Color(core.Text))
	got := c.Theme(Prefs{Theme: "dark", Colors: map[string]map[string]string{"dark": {string(core.Selection): sameAsText}}})
	if got.Color(core.Selection) != base.Color(core.Selection) {
		t.Fatal("an override as dark as its text was applied")
	}
}

// The choice tables are shared by every caller: a caller that changes the
// slice it got must not change what the next one sees.
func TestChoiceTablesAreCopies(t *testing.T) {
	SizeChoices()[0].ID = "mutated"
	CornerChoices()[0].ID = "mutated"
	ContrastChoices()[0].ID = "mutated"
	CustomColors()[0].Label = "mutated"
	if SizeChoices()[0].ID == "mutated" || CornerChoices()[0].ID == "mutated" || ContrastChoices()[0].ID == "mutated" || CustomColors()[0].Label == "mutated" {
		t.Fatal("a returned table is the shared one")
	}
}

// The size bases are the line heights of the default faces: the medium
// text size in the default fonts. If the default size or font changes,
// the bases (and every height built on them) must move with it.
func TestBaseLineIsTheDefaultFaceLineHeight(t *testing.T) {
	th := NewCatalog(Builtins()).Theme(Prefs{})
	ctx, err := core.NewCtx(th)
	if err != nil {
		t.Fatal(err)
	}
	if got := ctx.LineH(ctx.F.UI); got != core.BaseLine {
		t.Errorf("UI face line height = %d, BaseLine = %d", got, core.BaseLine)
	}
	if got := ctx.LineH(ctx.F.Small); got != core.BaseSmallLine {
		t.Errorf("small face line height = %d, BaseSmallLine = %d", got, core.BaseSmallLine)
	}
}

// Every text size choice gives rows that hold its own text.
func TestSizeChoicesGiveSaneSizes(t *testing.T) {
	for _, choice := range SizeChoices() {
		t.Run(choice.ID, func(t *testing.T) {
			th := NewCatalog(Builtins()).For("dark", choice.ID)
			ctx, err := core.NewCtx(th)
			if err != nil {
				t.Fatal(err)
			}
			sizes := ctx.Sizes
			if sizes.Line < core.MinLine || sizes.Line > core.MaxLine {
				t.Errorf("Line %d outside %d..%d", sizes.Line, core.MinLine, core.MaxLine)
			}
			if sizes.Row < sizes.Line || sizes.Compact < sizes.Line || sizes.Slim < sizes.Line {
				t.Errorf("a height is below the line: %+v", sizes)
			}
			if !(sizes.Row > sizes.Compact && sizes.Compact > sizes.Slim) {
				t.Errorf("want Row > Compact > Slim: %+v", sizes)
			}
			if sizes.Note < sizes.SmallLine {
				t.Errorf("Note %d below small line %d", sizes.Note, sizes.SmallLine)
			}
		})
	}
}
