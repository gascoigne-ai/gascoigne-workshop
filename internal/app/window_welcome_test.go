package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestWelcomeOpensTheFirstTimeAVersionRuns(t *testing.T) {
	a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: filepath.Join(t.TempDir(), "state.json"), Welcome: true}))
	if a.over.Dialog() == nil || !strings.Contains(a.over.Dialog().Title, "0.1.0 alpha") {
		t.Fatalf("dialog = %+v, want the welcome dialog of 0.1.0 alpha", a.over.Dialog())
	}
}

func TestWelcomeStaysClosedWhenNotAskedFor(t *testing.T) {
	a := testApp(t)
	if a.over.Dialog() != nil {
		t.Fatalf("dialog = %+v, want none", a.over.Dialog())
	}
}

func TestWelcomeIsRememberedOnceClosed(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true}))
	a.Step(core.Input{Escape: true})
	if a.over.Dialog() != nil {
		t.Fatal("Escape left the welcome dialog open")
	}
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
	again := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true}))
	if again.over.Dialog() != nil {
		t.Fatalf("welcome shown again for the same version: %+v", again.over.Dialog())
	}
}

func TestWelcomeShowsAgainForANewVersion(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := (statefile.Files{}).Save(statePath, state.State{Welcomed: "v0.0.9"}); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true}))
	if a.over.Dialog() == nil {
		t.Fatal("welcome stayed closed for a version that was not welcomed yet")
	}
}

func TestClickingTheVersionOpensWelcome(t *testing.T) {
	a := testApp(t)
	click(a, a.w-30, a.h-sizes.StatusH/2)
	if a.over.Dialog() == nil || !strings.Contains(a.over.Dialog().Title, "0.1.0 alpha") {
		t.Fatalf("dialog = %+v, want the welcome dialog", a.over.Dialog())
	}
}

// A dev build has no first run to wait for: WelcomeEvery opens the dialog
// at every start, even for a version that was welcomed before.
func TestWelcomeEveryOpensEvenWhenSeen(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := (statefile.Files{}).Save(statePath, state.State{Welcomed: version}); err != nil {
		t.Fatal(err)
	}
	if a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true})); a.over.Dialog() != nil {
		t.Fatal("welcome opened for a version already seen")
	}
	if a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true, WelcomeEvery: true})); a.over.Dialog() == nil {
		t.Fatal("WelcomeEvery left the dialog closed")
	}
}
