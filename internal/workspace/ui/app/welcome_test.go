package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// welcomeApp is a test app that shows the welcome dialog when this
// version has not been welcomed in the state file at statePath.
func welcomeApp(t *testing.T, statePath string) *App {
	t.Helper()
	return welcomeAppWith(t, Options{StatePath: statePath, Welcome: true})
}

func welcomeAppWith(t *testing.T, opts Options) *App {
	t.Helper()
	opts.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	a, err := NewApp(withAdapters(opts))
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(core.Input{})
	return a
}

func TestWelcomeOpensTheFirstTimeAVersionRuns(t *testing.T) {
	a := welcomeApp(t, filepath.Join(t.TempDir(), "state.json"))
	if a.dialog == nil || !strings.Contains(a.dialog.Title, "0.1.0 alpha") {
		t.Fatalf("dialog = %+v, want the welcome dialog of 0.1.0 alpha", a.dialog)
	}
}

func TestWelcomeStaysClosedWhenNotAskedFor(t *testing.T) {
	a, _ := newTestApp(t)
	if a.dialog != nil {
		t.Fatalf("dialog = %+v, want none", a.dialog)
	}
}

func TestWelcomeIsRememberedOnceClosed(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	a := welcomeApp(t, statePath)
	a.step(core.Input{Escape: true})
	if a.dialog != nil {
		t.Fatal("Escape left the welcome dialog open")
	}
	if err := a.SaveState(); err != nil {
		t.Fatal(err)
	}
	again := welcomeApp(t, statePath)
	if again.dialog != nil {
		t.Fatalf("welcome shown again for the same version: %+v", again.dialog)
	}
}

func TestWelcomeShowsAgainForANewVersion(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := (statefile.Files{}).Save(statePath, state.State{Welcomed: "v0.0.9"}); err != nil {
		t.Fatal(err)
	}
	a := welcomeApp(t, statePath)
	if a.dialog == nil {
		t.Fatal("welcome stayed closed for a version that was not welcomed yet")
	}
}

func TestClickingTheVersionOpensWelcome(t *testing.T) {
	a, _ := newTestApp(t)
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
	if a := welcomeAppWith(t, Options{StatePath: statePath, Welcome: true}); a.dialog != nil {
		t.Fatal("welcome opened for a version already seen")
	}
	if a := welcomeAppWith(t, Options{StatePath: statePath, Welcome: true, WelcomeEvery: true}); a.dialog == nil {
		t.Fatal("WelcomeEvery left the dialog closed")
	}
}
