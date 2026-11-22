//go:build !release

package main

import "testing"

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

func TestToolFormNeedsSendAtStart(t *testing.T) {
	step := toolForm("search")
	err := step.apply(&scene{sendOnStart: false})
	want := `tool "search" needs -send, which connects to the server that announces it`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}
