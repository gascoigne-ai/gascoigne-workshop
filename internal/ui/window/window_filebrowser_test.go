package window

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
)

// browsingApp is a test app without a dialog program whose file browser
// lists a fake folder tree; its home holds a folder and two files.
func browsingApp(t *testing.T) (*Component, string) {
	t.Helper()
	home := "/home/user"
	a := testApp(t)
	a.files = fakeDialog{suggest: "zenity"} // no dialog program, zenity would give one
	fake := newFakeFolders()
	fake.listing = map[string][]workspace.FolderEntry{home: {
		{Name: "sub", Dir: true}, {Name: "api.json"}, {Name: "notes.txt"},
	}}
	for _, name := range []string{"api.json", "notes.txt"} {
		fake.files[filepath.Join(home, name)] = []byte("{}")
	}
	a.folders = fake
	return a, home
}

// typePath puts path in the open browser's field as the user would: the field
// is selected whole and the path typed over it.
func typePath(a *Component, path string) {
	a.Step(core.Input{})
	a.Step(core.Input{SelectAll: true})
	a.typeText(path)
}

// pressEnter is the Enter key of one frame.
func pressEnter(a *Component) { a.Step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}}) }

func TestTheBrowserHoldsTheKeyboardLikeADialog(t *testing.T) {
	a, _ := browsingApp(t)
	a.pickFile("Choose the file to send", filedialog.OpenFile, "", func(string) {})
	if !a.over.HoldsKeyboard() {
		t.Fatal("the browser holds the keyboard like a dialog")
	}
}

func TestEscapeClosesTheBrowserWithoutAnAnswer(t *testing.T) {
	a, _ := browsingApp(t)
	called := false
	a.pickFile("Pick", filedialog.OpenFile, "", func(string) { called = true })
	a.Step(core.Input{Escape: true})
	if a.over.Browser.Shown() || called {
		t.Fatalf("browser %v called %v", a.over.Browser.Shown(), called)
	}
}

func TestSavingOverAFileAsksFirst(t *testing.T) {
	a, dir := browsingApp(t)
	var got string
	a.pickFile("Save response body", filedialog.SaveFile, "api.json", func(path string) { got = path })
	if !strings.HasSuffix(a.over.Browser.Path(), "api.json") {
		t.Fatalf("the save name follows the folder: %q", a.over.Browser.Path())
	}
	a.Step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if a.over.Dialog() == nil || a.over.Dialog().Title != "Replace file" || got != "" {
		t.Fatalf("dialog %+v got %q", a.over.Dialog(), got)
	}
	a.Step(core.Input{Escape: true})
	if a.over.Dialog() != nil || !a.over.Browser.Shown() {
		t.Fatal("Escape takes back the question first and leaves the browser open")
	}
	a.Step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	a.dialogDefault()
	if a.over.Browser.Shown() || got != filepath.Join(dir, "api.json") {
		t.Fatalf("Replace takes the path: %q, browser %v", got, a.over.Browser.Shown())
	}
}
