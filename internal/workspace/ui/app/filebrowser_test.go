package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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
	a, _ := newTestApp(t)
	a.files = fakeDialog{} // no dialog program
	a.folders = folders.Disk{}
	return a, dir
}

// browserDir is the folder the open browser shows, with links resolved, as
// the temporary folder of macOS is behind one.
func browserDir(t *testing.T, a *App) string {
	t.Helper()
	if a.browser == nil {
		t.Fatal("the file browser is not open")
	}
	return evalLinks(t, a.browser.model.Dir)
}

func evalLinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func pressEdit(a *App, edit core.Edit, keyDown ebiten.Key, alt bool) {
	a.step(core.Input{Edits: []core.Edit{edit}, Keys: []ebiten.Key{keyDown}, Alt: alt})
}

func TestWithoutADialogProgramTheFileBrowserOpensInTheHomeFolder(t *testing.T) {
	a, dir := browsingApp(t)
	a.pickFile("Choose the file to send", false, "", func(string) {})
	if got := browserDir(t, a); got != evalLinks(t, dir) {
		t.Fatalf("dir %q, want %q", got, dir)
	}
	a.step(core.Input{})
	dump := view.Dump(a.fileBrowserView())
	for _, want := range []string{`"Choose the file to send"`, `"3 items"`, `"Hidden files"`, `"sub"`, `"api.json"`, `"notes.txt"`, `"Cancel"`, `"Open"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the browser does not show %s:\n%s", want, dump)
		}
	}
	if !a.overlayOpen() {
		t.Fatal("the browser holds the keyboard like a dialog")
	}
}

func TestTheBrowserDoesNotRememberTheLastFolder(t *testing.T) {
	a, dir := browsingApp(t)
	a.pickFile("Pick", false, "", func(string) {})
	typePath(a, filepath.Join(dir, "sub")+string(filepath.Separator))
	a.browse(filebrowser.Entered{})
	if browserDir(t, a) != filepath.Join(evalLinks(t, dir), "sub") {
		t.Fatalf("Enter on a typed folder opens it: %q", a.browser.model.Dir)
	}
	a.closeBrowser()
	a.pickFile("Pick again", false, "", func(string) {})
	if got := browserDir(t, a); got != evalLinks(t, dir) {
		t.Fatalf("the next browser opens in %q, want home %q", got, dir)
	}
}

func TestTheBrowserIsDrivenByTheKeyboard(t *testing.T) {
	a, dir := browsingApp(t)
	var got string
	a.pickFile("Pick", false, "", func(path string) { got = path })
	a.step(core.Input{})
	pressEdit(a, core.EditDown, ebiten.KeyArrowDown, false) // sub
	a.step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if browserDir(t, a) != filepath.Join(evalLinks(t, dir), "sub") {
		t.Fatalf("Down and Enter open the folder: %q", a.browser.model.Dir)
	}
	pressEdit(a, core.EditBackspace, ebiten.KeyBackspace, false)
	if browserDir(t, a) != evalLinks(t, dir) {
		t.Fatalf("Backspace with nothing typed goes up: %q", a.browser.model.Dir)
	}
	if !strings.HasSuffix(a.browser.path.Text, string(filepath.Separator)) {
		t.Fatalf("the Backspace that went up must not also delete: %q", a.browser.path.Text)
	}
	// A selected text is the field's to delete, even when it is the whole
	// folder: Backspace must not also go up.
	a.browser.path.SelectAll()
	pressEdit(a, core.EditBackspace, ebiten.KeyBackspace, false)
	if browserDir(t, a) != evalLinks(t, dir) || a.browser.path.Text != "" {
		t.Fatalf("Backspace over a selection deletes it: dir %q text %q", a.browser.model.Dir, a.browser.path.Text)
	}
	typePath(a, a.browser.model.Dir+string(filepath.Separator))
	a.typeText("note")
	if visible := a.browser.model.Visible(); len(visible) != 1 || visible[0].Name != "notes.txt" {
		t.Fatalf("typing filters the list: %+v", visible)
	}
	pressEdit(a, core.EditUp, ebiten.KeyArrowUp, true)
	if browserDir(t, a) != filepath.Dir(evalLinks(t, dir)) {
		t.Fatalf("Alt+Up goes up: %q", a.browser.model.Dir)
	}
	typePath(a, filepath.Join(dir, "notes.txt"))
	a.step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if a.browser != nil || evalLinks(t, got) != filepath.Join(evalLinks(t, dir), "notes.txt") {
		t.Fatalf("Enter on a typed path chooses it: got %q, browser %v", got, a.browser != nil)
	}
}

func TestEscapeClosesTheBrowserWithoutAnAnswer(t *testing.T) {
	a, _ := browsingApp(t)
	called := false
	a.pickFile("Pick", false, "", func(string) { called = true })
	a.step(core.Input{Escape: true})
	if a.browser != nil || called {
		t.Fatalf("browser %v called %v", a.browser != nil, called)
	}
}

func TestSavingOverAFileAsksFirst(t *testing.T) {
	a, dir := browsingApp(t)
	var got string
	a.pickFile("Save response body", true, "api.json", func(path string) { got = path })
	if !strings.HasSuffix(a.browser.path.Text, "api.json") {
		t.Fatalf("the save name follows the folder: %q", a.browser.path.Text)
	}
	a.step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if a.dialog == nil || a.dialog.Title != "Replace file" || got != "" {
		t.Fatalf("dialog %+v got %q", a.dialog, got)
	}
	a.step(core.Input{Escape: true})
	if a.dialog != nil || a.browser == nil {
		t.Fatal("Escape takes back the question first and leaves the browser open")
	}
	a.step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	a.dialogDefault()
	if a.browser != nil || evalLinks(t, got) != filepath.Join(evalLinks(t, dir), "api.json") {
		t.Fatalf("Replace takes the path: %q, browser %v", got, a.browser != nil)
	}
}

func TestADoubleClickOpensARow(t *testing.T) {
	a, dir := browsingApp(t)
	var got string
	a.pickFile("Pick", false, "", func(path string) { got = path })
	x, y := shownCenter(t, a, "api.json")
	a.step(core.Input{X: x, Y: y, Click: true, Held: true, Clicks: 1})
	a.step(core.Input{X: x, Y: y})
	if got != "" || a.browser.model.Selected != 1 {
		t.Fatalf("a click selects: got %q selected %d", got, a.browser.model.Selected)
	}
	a.step(core.Input{X: x, Y: y, Click: true, Held: true, Clicks: 2})
	if evalLinks(t, got) != filepath.Join(evalLinks(t, dir), "api.json") {
		t.Fatalf("a double click opens: %q", got)
	}
}

func TestTheHiddenFilesToggleShowsDotNames(t *testing.T) {
	a, dir := browsingApp(t)
	if err := os.WriteFile(filepath.Join(dir, ".env"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	a.pickFile("Pick", false, "", func(string) {})
	if strings.Contains(view.Dump(a.fileBrowserView()), `".env"`) {
		t.Fatal("hidden names are left out at first")
	}
	x, y := shownCenter(t, a, "Hidden files")
	click(a, x, y)
	if !strings.Contains(view.Dump(a.fileBrowserView()), `".env"`) {
		t.Fatal("the toggle shows them")
	}
}

func TestAFolderThatCannotBeReadIsSaidInTheFooter(t *testing.T) {
	a, dir := browsingApp(t)
	a.pickFile("Pick", false, "", func(string) {})
	typePath(a, filepath.Join(dir, "gone")+string(filepath.Separator))
	a.browse(filebrowser.Entered{})
	if !strings.Contains(a.browser.model.Problem, "Cannot open") {
		t.Fatalf("problem %q", a.browser.model.Problem)
	}
	if !strings.Contains(view.Dump(a.fileBrowserView()), "Cannot open") {
		t.Fatal("the footer shows the problem")
	}
}

// shownCenter is the middle of the element showing text, once a frame has
// laid the browser out.
func shownCenter(t *testing.T, a *App, text string) (int, int) {
	t.Helper()
	a.step(core.Input{X: 5000, Y: 5000})
	return textCenter(t, a, text)
}

// typePath puts path in the browser's field as if it was typed.
func typePath(a *App, path string) {
	a.browser.path = widget.NewTextField(path)
	a.browse(filebrowser.Typed{Text: path})
}
