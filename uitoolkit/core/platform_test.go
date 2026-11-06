package core

import "testing"

func TestThePrimaryModifierIsCommandOnMacAndControlElsewhere(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		ctrl, meta bool
		want       bool
	}{
		{"linux control", "linux", true, false, true},
		{"linux meta is not primary", "linux", false, true, false},
		{"windows control", "windows", true, false, true},
		{"windows meta (the Windows key) is not primary", "windows", false, true, false},
		{"mac command", "darwin", false, true, true},
		{"mac control still works", "darwin", true, false, true},
		{"mac nothing", "darwin", false, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := PrimaryDown(test.goos, test.ctrl, test.meta); got != test.want {
				t.Fatalf("PrimaryDown = %v, want %v", got, test.want)
			}
		})
	}
}

func TestKeyTextNamesThePrimaryModifierForThePlatform(t *testing.T) {
	tests := []struct {
		goos, in, want string
	}{
		{"linux", "Ctrl+Z", "Ctrl+Z"},
		{"windows", "Ctrl+Shift+Tab", "Ctrl+Shift+Tab"},
		{"darwin", "Ctrl+Z", "Cmd+Z"},
		{"darwin", "Ctrl+Y or Ctrl+Shift+Z", "Cmd+Y or Cmd+Shift+Z"},
		{"darwin", "Alt+Enter", "Alt+Enter"},
		{"darwin", "", ""},
	}
	for _, test := range tests {
		if got := PlatformKeyText(test.goos, test.in); got != test.want {
			t.Errorf("%s: PlatformKeyText(%q) = %q, want %q", test.goos, test.in, got, test.want)
		}
	}
}
