//go:build !release

package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/userdir"
)

// A screenshot run that changes settings (-set, -record) and is given no
// -settings keeps them in memory. It saved them to the default settings
// file, so a screenshot taken by hand overwrote the user's real settings.
func TestAScreenshotWithoutSettingsFlagLeavesTheDefaultSettingsFileAlone(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"set", []string{"-set", "theme=light"}},
		// a combination no other action has, so it is saved and not left pending
		{"record", []string{"-page", "shortcuts", "-record", "close-tab=Ctrl+Alt+K"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sandbox := t.TempDir()
			flags := append([]string{"-screenshot", filepath.Join(sandbox, "shot.png"), "-state", "", "-themes", ""}, tc.args...)
			runProgram(t, sandbox, shotWidth, shotHeight, flags...)
			// softwareEnv puts XDG_CONFIG_HOME at sandbox/config.
			defaultSettings := filepath.Join(sandbox, "config", userdir.Name, "settings.json")
			if _, err := os.Stat(defaultSettings); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("the default settings file was touched (stat: %v)", err)
			}
		})
	}
}

// Given -settings, a screenshot run saves its changes to that file.
func TestAScreenshotWithSettingsFlagSavesToThatFile(t *testing.T) {
	sandbox := t.TempDir()
	settingsPath := filepath.Join(sandbox, "chosen.json")
	runProgram(t, sandbox, shotWidth, shotHeight,
		"-screenshot", filepath.Join(sandbox, "shot.png"), "-state", "", "-themes", "",
		"-settings", settingsPath, "-set", "theme=light")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("the chosen settings file was not written: %v", err)
	}
}
