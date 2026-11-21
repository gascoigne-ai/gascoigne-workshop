package app

import "testing"

func TestAScreenshotCanChangeSettingsLiveThroughTheSamePathAsTheDialog(t *testing.T) {
	a := testApp(t)
	a.shot.path = "unused.png"
	a.shot.steps = []Step{Set("flat=true"), Set("contrast=high"), Set("uiScale=120")}
	for range settleFrames {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if !a.settings.Flat || a.settings.Contrast != "high" || a.settings.UIScalePct != 120 {
		t.Fatalf("flat %v contrast %q scale %d", a.settings.Flat, a.settings.Contrast, a.settings.UIScalePct)
	}
	if !a.ctx.Th.NoGradients {
		t.Fatal("the theme did not follow the live change")
	}
}

func TestParseSettingTakesBooleansNumbersAndText(t *testing.T) {
	for spec, want := range map[string]any{"flat=true": true, "uiScale=130": 130, "layout=stacked": "stacked"} {
		key, value, err := parseSetting(spec)
		if err != nil || value != want || key == "" {
			t.Errorf("%q: %q %v %v", spec, key, value, err)
		}
	}
	if _, _, err := parseSetting("nonsense"); err == nil {
		t.Error("a spec without = was accepted")
	}
}

// Each face has its own size key, and -set reaches the live theme for all three.
func TestAScreenshotCanSetTheSizeOfEachFace(t *testing.T) {
	a := testApp(t)
	a.shot.path = "unused.png"
	a.shot.steps = []Step{Set("fontSize=16"), Set("smallFontSize=8"), Set("codeFontSize=32")}
	for range settleFrames {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	th := a.ctx.Th
	if th.FontSize != 16 || th.SmallSize != 8 || th.MonoSize != 32 {
		t.Fatalf("theme sizes = %v, %v, %v; want 16, 8, 32", th.FontSize, th.SmallSize, th.MonoSize)
	}
}
