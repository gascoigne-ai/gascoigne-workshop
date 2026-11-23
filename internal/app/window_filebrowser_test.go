package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
)

// browsingApp is a test app without a dialog program whose file browser
// lists the disk; its home is a temporary folder holding a folder and two
// files, and its working directory another one.
func browsingApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"api.json", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir)
	t.Chdir(t.TempDir())
	a := testApp(t)
	a.files = fakeDialog{suggest: "zenity"} // no dialog program, zenity would give one
	a.folders = folders.Disk{}
	return a, dir
}

func evalLinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// typePath puts path in the open browser's field as the user would: the field
// is selected whole and the path typed over it.
func typePath(a *App, path string) {
	a.Step(core.Input{})
	a.Step(core.Input{SelectAll: true})
	a.typeText(path)
}

// pressEnter is the Enter key of one frame.
func pressEnter(a *App) { a.Step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}}) }

func TestTheBrowserHoldsTheKeyboardLikeADialog(t *testing.T) {
	a, _ := browsingApp(t)
	a.pickFile("Choose the file to send", filedialog.OpenFile, "", func(string) {})
	if !a.overlayOpen() {
		t.Fatal("the browser holds the keyboard like a dialog")
	}
}

func TestEscapeClosesTheBrowserWithoutAnAnswer(t *testing.T) {
	a, _ := browsingApp(t)
	called := false
	a.pickFile("Pick", filedialog.OpenFile, "", func(string) { called = true })
	a.Step(core.Input{Escape: true})
	if a.browser.Shown() || called {
		t.Fatalf("browser %v called %v", a.browser.Shown(), called)
	}
}

func TestSavingOverAFileAsksFirst(t *testing.T) {
	a, dir := browsingApp(t)
	var got string
	a.pickFile("Save response body", filedialog.SaveFile, "api.json", func(path string) { got = path })
	if !strings.HasSuffix(a.browser.Path(), "api.json") {
		t.Fatalf("the save name follows the folder: %q", a.browser.Path())
	}
	a.Step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if a.dialog == nil || a.dialog.Title != "Replace file" || got != "" {
		t.Fatalf("dialog %+v got %q", a.dialog, got)
	}
	a.Step(core.Input{Escape: true})
	if a.dialog != nil || !a.browser.Shown() {
		t.Fatal("Escape takes back the question first and leaves the browser open")
	}
	a.Step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	a.dialogDefault()
	if a.browser.Shown() || evalLinks(t, got) != filepath.Join(evalLinks(t, dir), "api.json") {
		t.Fatalf("Replace takes the path: %q, browser %v", got, a.browser.Shown())
	}
}
