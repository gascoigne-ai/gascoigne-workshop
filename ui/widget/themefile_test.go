package widget

import (
	"image/color"
	"strings"
	"testing"
)

func TestParseThemeResolvesVariablesAlphaAndControls(t *testing.T) {
	th, err := ParseTheme([]byte(`{
		"name": "t", "fontSize": 16,
		"variables": {"rose": "#e88aa6", "deep": "#10203040"},
		"colors": {"accent": "$rose", "shadow": "$deep", "border": "#445566"},
		"controls": {
			"button": {"radius": 6, "gradient": 0.1, "border": "border"},
			"hero":   {"gradient": ["$rose", "#000000"]},
			"flat":   {}
		}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if th.Name != "t" || th.FontSize != 16 || th.RadiusScale != 1 {
		t.Fatalf("header: %+v", th)
	}
	if th.Color(Accent) != RGB(0xe88aa6) || th.Color(Shadow) != (color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0x40}) {
		t.Fatalf("colors: %+v %+v", th.Color(Accent), th.Color(Shadow))
	}
	b := th.Controls["button"]
	if b.Radius != 6 || b.Shade != 0.1 || b.Border != Border || b.Top != nil {
		t.Fatalf("button: %+v", b)
	}
	h := th.Controls["hero"]
	if h.Top == nil || *h.Top != RGB(0xe88aa6) || *h.Bottom != RGB(0) || !h.HasGradient() {
		t.Fatalf("hero: %+v", h)
	}
	if th.Controls["flat"].HasGradient() {
		t.Fatal("an empty control is flat")
	}
}

func TestParseThemeMergesOverABase(t *testing.T) {
	base := Dark()
	th, err := ParseTheme([]byte(`{"colors": {"accent": "#112233"}, "controls": {"button": {"radius": 2}}}`), &base)
	if err != nil {
		t.Fatal(err)
	}
	if th.Color(Accent) != RGB(0x112233) || th.Color(Bg) != base.Color(Bg) {
		t.Fatal("a color is replaced, the rest kept")
	}
	if th.Controls["button"].Radius != 2 || th.Controls["primary"].Radius != base.Controls["primary"].Radius {
		t.Fatal("a control is replaced, the rest kept")
	}
	if base.Color(Accent) == RGB(0x112233) || base.Controls["button"].Radius == 2 {
		t.Fatal("the base theme must not change")
	}
}

func TestParseThemeRejectsMistakes(t *testing.T) {
	for name, src := range map[string]string{
		"broken json":          `{`,
		"unknown field":        `{"colour": {}}`,
		"unknown variable":     `{"colors": {"bg": "$nope"}}`,
		"bad color":            `{"colors": {"bg": "red"}}`,
		"gradient too big":     `{"controls": {"b": {"gradient": 0.9}}}`,
		"gradient shape":       `{"controls": {"b": {"gradient": ["#000000"]}}}`,
		"negative radius":      `{"controls": {"b": {"radius": -1}}}`,
		"border without color": `{"controls": {"b": {"border": "nothing"}}}`,
	} {
		if _, err := ParseTheme([]byte(src), nil); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestStyleAppliesRoundingScaleAndTheGradientSwitch(t *testing.T) {
	th := Dark()
	base := th.Style("button")
	if base.Radius <= 0 || !th.Style("hero").HasGradient() {
		t.Fatalf("the dashboard look: %+v", base)
	}
	th.RadiusScale = 0
	if th.Style("button").Radius != 0 {
		t.Fatal("scale 0 is square")
	}
	th.RadiusScale = 1.5
	if got := th.Style("button").Radius; got != base.Radius*1.5 {
		t.Fatalf("scaled radius %v", got)
	}
	th.NoGradients = true
	if th.Style("button").HasGradient() || th.Style("hero").HasGradient() {
		t.Fatal("gradients off must flatten every control")
	}
	if (Dark()).Style("not-a-control") != (ControlStyle{}) {
		t.Fatal("an unlisted control is plain")
	}
}

func TestBuiltinThemesListTheSameControlsAndTokens(t *testing.T) {
	d, l := Dark(), Light()
	for name := range d.Controls {
		if _, ok := l.Controls[name]; !ok {
			t.Errorf("light lacks control %q", name)
		}
	}
	for tok := range d.Colors {
		if _, ok := l.Colors[tok]; !ok {
			t.Errorf("light lacks color %q", tok)
		}
	}
	for _, name := range []string{"button", "primary", "icon-button", "field", "table", "menu", "tooltip", "tabstrip", "statusbar", "hero"} {
		if _, ok := d.Controls[name]; !ok {
			t.Errorf("no style for %q", name)
		}
	}
	if !strings.Contains(d.Name+l.Name, "dark") {
		t.Fatal("themes carry their names")
	}
}
