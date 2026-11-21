package theming

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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

// Each face is brought into its own range; 0 is the face's default.
func TestFontSizesResolved(t *testing.T) {
	tests := []struct {
		name                        string
		sizes                       FontSizes
		wantUI, wantSmall, wantMono float64
	}{
		{"zero is every default", FontSizes{}, 14, 12, 13},
		{"inside", FontSizes{UI: 16, Small: 14, Code: 15}, 16, 14, 15},
		{"independent", FontSizes{UI: 10, Small: 24, Code: 8}, 10, 24, 8},
		{"lowest", FontSizes{UI: MinFontSize, Small: MinSmallSize, Code: MinCodeSize}, 10, 8, 8},
		{"highest", FontSizes{UI: MaxFontSize, Small: MaxSmallSize, Code: MaxCodeSize}, 24, 24, 32},
		{"below", FontSizes{UI: 9, Small: 7, Code: 7}, 10, 8, 8},
		{"above", FontSizes{UI: 99, Small: 99, Code: 99}, 24, 24, 32},
		{"negative", FontSizes{UI: -1, Small: -1, Code: -1}, 10, 8, 8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ui, small, mono := test.sizes.Resolved()
			if ui != test.wantUI || small != test.wantSmall || mono != test.wantMono {
				t.Errorf("%+v.Resolved() = %v, %v, %v; want %v, %v, %v", test.sizes, ui, small, mono, test.wantUI, test.wantSmall, test.wantMono)
			}
		})
	}
}

func TestCatalogThemeUsesTheThreeSizes(t *testing.T) {
	catalog := NewCatalog(Builtins())
	for _, test := range []struct {
		prefs                       Prefs
		wantUI, wantSmall, wantMono float64
	}{
		{Prefs{Theme: "dark", Fonts: FontSizes{UI: 18, Small: 11, Code: 20}}, 18, 11, 20},
		{Prefs{Theme: "dark"}, 14, 12, 13},
		{Prefs{Theme: "light", Fonts: FontSizes{UI: 10}}, 10, 12, 13},
	} {
		th := catalog.Theme(test.prefs)
		if th.FontSize != test.wantUI || th.SmallSize != test.wantSmall || th.MonoSize != test.wantMono {
			t.Errorf("Theme(%+v) sizes = %v, %v, %v; want %v, %v, %v", test.prefs, th.FontSize, th.SmallSize, th.MonoSize, test.wantUI, test.wantSmall, test.wantMono)
		}
	}
	if got := catalog.For("dark", FontSizes{UI: 20, Small: 9, Code: 30}); got.FontSize != 20 || got.SmallSize != 9 || got.MonoSize != 30 {
		t.Errorf("For(dark, 20/9/30) = %v, %v, %v", got.FontSize, got.SmallSize, got.MonoSize)
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
	CornerChoices()[0].ID = "mutated"
	ContrastChoices()[0].ID = "mutated"
	CustomColors()[0].Label = "mutated"
	if CornerChoices()[0].ID == "mutated" || ContrastChoices()[0].ID == "mutated" || CustomColors()[0].Label == "mutated" {
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

// Every text size of the range gives rows that hold its own text.
func TestEveryFontSizeGivesSaneSizes(t *testing.T) {
	for font := MinFontSize; font <= MaxFontSize; font++ {
		t.Run(strconv.Itoa(font), func(t *testing.T) {
			th := NewCatalog(Builtins()).For("dark", FontSizes{UI: font, Small: MaxSmallSize, Code: MaxCodeSize})
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
