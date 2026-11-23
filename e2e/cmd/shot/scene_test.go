//go:build !release

package main

import (
	"flag"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// sceneOf parses args as the command line would and names the type of each
// step the scene gets, in order.
func sceneOf(t *testing.T, args ...string) []string {
	t.Helper()
	set := flag.NewFlagSet("workshop", flag.ContinueOnError)
	scene := newSceneFlags(set)
	if err := set.Parse(args); err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, step := range scene.steps() {
		kinds = append(kinds, strings.TrimPrefix(strings.TrimPrefix(fmt.Sprintf("%T", step), "*"), "main."))
	}
	return kinds
}

func TestSceneFlagsMakeOneStepPerFlagInTheFixedOrder(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"no flags give no steps", nil, nil},
		{"one flag", []string{"-request", "Echo"}, []string{"openRequestStep"}},
		{"a repeated flag gives a step each, in the given order",
			[]string{"-toast", "a", "-toast", "b", "-record", "x", "-record", "y", "-add-collection", "d1", "-add-collection", "d2"},
			[]string{"addCollectionStep", "addCollectionStep", "toastStep", "toastStep", "recordStep", "recordStep"}},
		{"the order of the flags does not matter",
			[]string{"-sending", "-find", "x", "-filter", "f", "-wrap", "-request", "Echo", "-set", "flat=true"},
			[]string{"setSettingStep", "openRequestStep", "wrapStep", "filterStep", "sendOpenStep", "findStep"}},
		{"a boolean flag turned off gives no step", []string{"-wrap=false", "-log"}, []string{"callLogStep"}},
		{"an empty text gives no step", []string{"-request", "", "-pane", "Settings"}, []string{"showPaneStep"}},
		{"every flag in the order a scene needs",
			[]string{"-tool", "t", "-find", "f", "-sending", "-type-url", "u", "-filter", "f", "-close-all", "-browse", "d", "-log", "-sent", "-raw-settings",
				"-record", "r", "-font-list", "ui", "-page", "p", "-toast", "t", "-dialog", "rename", "-wrap", "-type-body", "b", "-body", "json", "-response-pane", "Tests", "-pane", "Params",
				"-request", "Echo", "-open-all", "-add-collection", "d", "-set", "a=b"},
			[]string{"setSettingStep", "addCollectionStep", "openAllStep", "openRequestStep", "showPaneStep", "showResponsePaneStep", "chooseBodyStep", "typeBodyStep", "wrapStep",
				"openDialogStep", "toastStep", "settingsPageStep", "fontListStep", "recordStep", "rawSettingsStep", "sentLogStep", "callLogStep", "browseStep", "closeAllStep", "filterStep", "typeURLStep",
				"sendOpenStep", "findStep", "toolFormStep"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sceneOf(t, tc.args...); !slices.Equal(got, tc.want) {
				t.Errorf("steps %v, want %v", got, tc.want)
			}
		})
	}
}
