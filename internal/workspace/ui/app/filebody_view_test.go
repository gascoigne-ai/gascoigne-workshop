package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fileBodyStates are the looks of the file body editor: no path yet, a
// path with no file behind it (the note turns red) and a path to a file.
var fileBodyStates = []struct {
	name string
	path func(t *testing.T) string
}{
	{"empty", func(t *testing.T) string { return "" }},
	{"missing", func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.bin") }},
	{"found", func(t *testing.T) string {
		path := filepath.Join(t.TempDir(), "found.bin")
		if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}},
}

// The file body editor is pinned as text (element kinds, options, rects)
// for each state at the request pane's width and at the narrowest main
// area, so moving it to a template cannot change it unseen.
func TestTheFileBodyLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", settingsWideW},
		{"narrow", settingsNarrowW},
	}
	for _, state := range fileBodyStates {
		for _, size := range sizes {
			name := "filebody-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				tab := a.cur()
				tab.file = widget.NewTextField(state.path(t))
				root := a.fileBody(tab)
				Layout(a.ctx, root, size.width, settingsPaneH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// What Dump does not show of the path field: it edits the tab's own file
// field, shows its placeholder and does nothing on Enter.
func TestTheFileBodyFieldEditsTheTabsPath(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	tab := a.cur()
	fields := fieldStates(a.fileBody(tab))
	if len(fields) != 1 {
		t.Fatalf("%d fields, want 1: %q", len(fields), fields)
	}
	for _, want := range []string{"field:" + fmt.Sprintf("%p", &tab.file), "placeholder:Path of the file to send", "onEnter:<nil>"} {
		if !strings.Contains(fields[0], want) {
			t.Errorf("path field lacks %q: %s", want, fields[0])
		}
	}
}

// The file body in the test window's Body pane: the path field and the
// Choose... button share the row under the type chips.
const (
	fileBodyRowY   = 244
	filePathFieldX = 400
	fileChooseX    = 712
)

// openFileBody shows the Body pane of the open request with a File body.
func openFileBody(t *testing.T) *App {
	a := openBodyPane(t)
	t.Cleanup(a.waitWrites)
	a.setBodyType(a.cur(), model.BodyFile)
	a.step(core.Input{})
	return a
}

// Typing into the path field makes the typed path the request's body.
func TestTypingAFilePathSetsTheBody(t *testing.T) {
	a := openFileBody(t)
	click(a, filePathFieldX, fileBodyRowY)
	a.typeText("data.bin")
	tab := a.cur()
	if tab.file.Text != "data.bin" || tab.Req.Body != "data.bin" {
		t.Fatalf("field %q body %q, want data.bin in both", tab.file.Text, tab.Req.Body)
	}
}

// Without a desktop file chooser Choose... opens the in-app file browser,
// and the chosen file becomes the field's text and the request's body.
func TestChooseAsksForTheFileAndSetsThePath(t *testing.T) {
	a := openFileBody(t)
	a.folders = folders.Disk{}
	picked := filepath.Join(t.TempDir(), "picked.bin")
	if err := os.WriteFile(picked, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	click(a, fileChooseX, fileBodyRowY)
	if a.browser == nil || a.browser.title != "Choose the file to send" {
		t.Fatalf("browser = %+v, want the file browser", a.browser)
	}
	typePath(a, picked)
	a.step(core.Input{Enter: true})
	tab := a.cur()
	if tab.file.Text != picked || tab.Req.Body != picked {
		t.Fatalf("field %q body %q, want %q in both", tab.file.Text, tab.Req.Body, picked)
	}
}

// Right-clicking the path field opens the text menu.
func TestTheFilePathFieldHasTheTextMenu(t *testing.T) {
	a := openFileBody(t)
	rightClick(a, filePathFieldX, fileBodyRowY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}
