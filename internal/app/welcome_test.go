package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestWelcomeOpensTheFirstTimeAVersionRuns(t *testing.T) {
	a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: filepath.Join(t.TempDir(), "state.json"), Welcome: true}))
	if a.dialog == nil || !strings.Contains(a.dialog.Title, "0.1.0 alpha") {
		t.Fatalf("dialog = %+v, want the welcome dialog of 0.1.0 alpha", a.dialog)
	}
}

func TestWelcomeStaysClosedWhenNotAskedFor(t *testing.T) {
	a := testApp(t)
	if a.dialog != nil {
		t.Fatalf("dialog = %+v, want none", a.dialog)
	}
}

func TestWelcomeIsRememberedOnceClosed(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true}))
	a.step(core.Input{Escape: true})
	if a.dialog != nil {
		t.Fatal("Escape left the welcome dialog open")
	}
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
	again := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true}))
	if again.dialog != nil {
		t.Fatalf("welcome shown again for the same version: %+v", again.dialog)
	}
}

func TestWelcomeShowsAgainForANewVersion(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := (statefile.Files{}).Save(statePath, state.State{Welcomed: "v0.0.9"}); err != nil {
		t.Fatal(err)
	}
	a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true}))
	if a.dialog == nil {
		t.Fatal("welcome stayed closed for a version that was not welcomed yet")
	}
}

func TestClickingTheVersionOpensWelcome(t *testing.T) {
	a := testApp(t)
	click(a, a.w-30, a.h-statusH/2)
	if a.dialog == nil || !strings.Contains(a.dialog.Title, "0.1.0 alpha") {
		t.Fatalf("dialog = %+v, want the welcome dialog", a.dialog)
	}
}

func TestWelcomeNotesNameEveryProtocol(t *testing.T) {
	var all strings.Builder
	for _, section := range welcomeNotes {
		all.WriteString(section.Title + "\n" + strings.Join(section.Items, "\n") + "\n")
	}
	for _, name := range []string{"REST", "WebSocket", "SSE", "MCP", "gRPC", "cURL", "OpenAPI", "Postman"} {
		if !strings.Contains(all.String(), name) {
			t.Errorf("the welcome notes do not mention %s", name)
		}
	}
}

// The notes are prose, and a Lines column does not wrap, so the dialog
// wraps each point itself at a word boundary.
func TestWrapBreaksAtWordsWithinTheLimit(t *testing.T) {
	tests := []struct {
		name, text string
		limit      int
		want       string
	}{
		{"short text stays", "one two", 20, "one two"},
		{"breaks at the last space that fits", "one two three four", 9, "one two\nthree\nfour"},
		{"a word longer than the limit stands alone", "tiny enormousword end", 6, "tiny\nenormousword\nend"},
		{"empty text", "", 10, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := wrap(test.text, test.limit); got != test.want {
				t.Fatalf("wrap(%q, %d) = %q, want %q", test.text, test.limit, got, test.want)
			}
		})
	}
}

// A dev build has no first run to wait for: WelcomeEvery opens the dialog
// at every start, even for a version that was welcomed before.
func TestWelcomeEveryOpensEvenWhenSeen(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := (statefile.Files{}).Save(statePath, state.State{Welcomed: version}); err != nil {
		t.Fatal(err)
	}
	if a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true})); a.dialog != nil {
		t.Fatal("welcome opened for a version already seen")
	}
	if a := testApp(t, untouchedSettings(), withOptions(Options{StatePath: statePath, Welcome: true, WelcomeEvery: true})); a.dialog == nil {
		t.Fatal("WelcomeEvery left the dialog closed")
	}
}
