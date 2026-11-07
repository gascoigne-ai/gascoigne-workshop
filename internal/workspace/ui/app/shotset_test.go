package ui

import "testing"

func TestAScreenshotCanChangeSettingsLiveThroughTheSamePathAsTheDialog(t *testing.T) {
	a, _ := newTestApp(t)
	a.shot.path = "unused.png"
	a.shot.set = []string{"flat=true", "contrast=high", "uiScale=120"}
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
