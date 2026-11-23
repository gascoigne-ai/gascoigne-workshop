package filebrowser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/folders"
	browsing "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// browsingWorld is a browser without a dialog program that lists the disk;
// its home is a temporary folder holding a folder and two files, and its
// working directory another one.
func browsingWorld(t *testing.T) (*world, string) {
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
	browsingWorld := newWorld(t, folders.Disk{})
	browsingWorld.desktop.suggest = "zenity" // no dialog program, zenity would give one
	return browsingWorld, dir
}

// browserDir is the folder the open browser shows, with links resolved, as
// the temporary folder of macOS is behind one.
func browserDir(t *testing.T, world *world) string {
	t.Helper()
	if !world.c.Shown() {
		t.Fatal("the file browser is not open")
	}
	return evalLinks(t, world.c.session.model.Dir)
}

func evalLinks(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func pressEdit(world *world, edit core.Edit, keyDown ebiten.Key, alt bool) {
	world.step(core.Input{Edits: []core.Edit{edit}, Keys: []ebiten.Key{keyDown}, Alt: alt})
}

// typePath puts path in the browser's field as if it was typed.
func typePath(world *world, path string) {
	world.c.session.path = widget.NewTextField(path)
	world.c.update(browsing.Typed{Text: path})
}

func TestWithoutADialogProgramTheFileBrowserOpensInTheHomeFolder(t *testing.T) {
	world, dir := browsingWorld(t)
	world.c.Pick("Choose the file to send", filedialog.OpenFile, "", func(string) {})
	if got := browserDir(t, world); got != evalLinks(t, dir) {
		t.Fatalf("dir %q, want %q", got, dir)
	}
	world.step(core.Input{})
	dump := world.layoutDump()
	for _, want := range []string{`"Choose the file to send"`, `"3 items"`, `"Hidden files"`, `"sub"`, `"api.json"`, `"notes.txt"`, `"Cancel"`, `"Open"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the browser does not show %s:\n%s", want, dump)
		}
	}
}

func TestTheBrowserDoesNotRememberTheLastFolder(t *testing.T) {
	world, dir := browsingWorld(t)
	world.c.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	typePath(world, filepath.Join(dir, "sub")+string(filepath.Separator))
	world.c.update(browsing.Entered{})
	if browserDir(t, world) != filepath.Join(evalLinks(t, dir), "sub") {
		t.Fatalf("Enter on a typed folder opens it: %q", world.c.session.model.Dir)
	}
	world.c.Close()
	world.c.Pick("Pick again", filedialog.OpenFile, "", func(string) {})
	if got := browserDir(t, world); got != evalLinks(t, dir) {
		t.Fatalf("the next browser opens in %q, want home %q", got, dir)
	}
}

func TestTheBrowserIsDrivenByTheKeyboard(t *testing.T) {
	world, dir := browsingWorld(t)
	var got string
	world.c.Pick("Pick", filedialog.OpenFile, "", func(path string) { got = path })
	world.step(core.Input{})
	pressEdit(world, core.EditDown, ebiten.KeyArrowDown, false) // sub
	world.step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if browserDir(t, world) != filepath.Join(evalLinks(t, dir), "sub") {
		t.Fatalf("Down and Enter open the folder: %q", world.c.session.model.Dir)
	}
	pressEdit(world, core.EditBackspace, ebiten.KeyBackspace, false)
	if browserDir(t, world) != evalLinks(t, dir) {
		t.Fatalf("Backspace with nothing typed goes up: %q", world.c.session.model.Dir)
	}
	if !strings.HasSuffix(world.c.session.path.Text, string(filepath.Separator)) {
		t.Fatalf("the Backspace that went up must not also delete: %q", world.c.session.path.Text)
	}
	// A selected text is the field's to delete, even when it is the whole
	// folder: Backspace must not also go up.
	world.c.session.path.SelectAll()
	pressEdit(world, core.EditBackspace, ebiten.KeyBackspace, false)
	if browserDir(t, world) != evalLinks(t, dir) || world.c.session.path.Text != "" {
		t.Fatalf("Backspace over a selection deletes it: dir %q text %q", world.c.session.model.Dir, world.c.session.path.Text)
	}
	typePath(world, world.c.session.model.Dir+string(filepath.Separator))
	world.step(core.Input{Typed: []rune("note")})
	if visible := world.c.session.model.Visible(); len(visible) != 1 || visible[0].Name != "notes.txt" {
		t.Fatalf("typing filters the list: %+v", visible)
	}
	pressEdit(world, core.EditUp, ebiten.KeyArrowUp, true)
	if browserDir(t, world) != filepath.Dir(evalLinks(t, dir)) {
		t.Fatalf("Alt+Up goes up: %q", world.c.session.model.Dir)
	}
	typePath(world, filepath.Join(dir, "notes.txt"))
	world.step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if world.c.Shown() || evalLinks(t, got) != filepath.Join(evalLinks(t, dir), "notes.txt") {
		t.Fatalf("Enter on a typed path chooses it: got %q, browser %v", got, world.c.Shown())
	}
}

func TestADoubleClickOpensARow(t *testing.T) {
	world, dir := browsingWorld(t)
	var got string
	world.c.Pick("Pick", filedialog.OpenFile, "", func(path string) { got = path })
	x, y := world.textCenter("api.json")
	world.step(core.Input{X: x, Y: y, Click: true, Held: true, Clicks: 1})
	world.step(core.Input{X: x, Y: y})
	if got != "" || world.c.session.model.Selected != 1 {
		t.Fatalf("a click selects: got %q selected %d", got, world.c.session.model.Selected)
	}
	world.step(core.Input{X: x, Y: y, Click: true, Held: true, Clicks: 2})
	if evalLinks(t, got) != filepath.Join(evalLinks(t, dir), "api.json") {
		t.Fatalf("a double click opens: %q", got)
	}
}

func TestTheHiddenFilesToggleShowsDotNames(t *testing.T) {
	world, dir := browsingWorld(t)
	if err := os.WriteFile(filepath.Join(dir, ".env"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	world.c.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	if strings.Contains(world.layoutDump(), `".env"`) {
		t.Fatal("hidden names are left out at first")
	}
	x, y := world.textCenter("Hidden files")
	world.click(x, y)
	if !strings.Contains(world.layoutDump(), `".env"`) {
		t.Fatal("the toggle shows them")
	}
}

func TestAFolderThatCannotBeReadIsSaidInTheFooter(t *testing.T) {
	world, dir := browsingWorld(t)
	world.c.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	typePath(world, filepath.Join(dir, "gone")+string(filepath.Separator))
	world.c.update(browsing.Entered{})
	if !strings.Contains(world.c.session.model.Problem, "Cannot open") {
		t.Fatalf("problem %q", world.c.session.model.Problem)
	}
	dump := world.layoutDump()
	if !strings.Contains(dump, "Cannot open") {
		t.Fatal("the footer shows the problem")
	}
	if strings.Contains(dump, "For native dialogs") {
		t.Fatalf("the hint gives way to the problem:\n%s", dump)
	}
}

func TestTheFooterSaysWhichProgramGivesNativeDialogs(t *testing.T) {
	world, _ := browsingWorld(t)
	world.c.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	if dump := world.layoutDump(); !strings.Contains(dump, `"For native dialogs install zenity"`) {
		t.Fatalf("no hint in the footer:\n%s", dump)
	}
}

func TestNoHintWhenNoProgramIsSuggested(t *testing.T) {
	world, _ := browsingWorld(t)
	world.desktop.suggest = ""
	world.c.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	if dump := world.layoutDump(); strings.Contains(dump, "For native dialogs") {
		t.Fatalf("a hint with no program to name:\n%s", dump)
	}
}
