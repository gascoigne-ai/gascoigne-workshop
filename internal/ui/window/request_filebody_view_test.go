package window

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The file body in the test window's Body pane: the path field and the
// Choose... button share the row under the type chips.
const (
	fileBodyRowY   = 244
	filePathFieldX = 400
	fileChooseX    = 712
)

// openFileBody shows the Body pane of the open request with a File body.
func openFileBody(t *testing.T) *Component {
	a := openBodyPane(t)
	t.Cleanup(a.WaitWrites)
	a.editor.Body.ChooseType(a.cur().Tab, request.BodyFile)
	a.Step(core.Input{})
	return a
}

// Typing into the path field makes the typed path the request's body.
func TestTypingAFilePathSetsTheBody(t *testing.T) {
	a := openFileBody(t)
	click(a, filePathFieldX, fileBodyRowY)
	a.typeText("data.bin")
	tab := a.cur()
	if a.editor.Body.File(tab.Tab).Text != "data.bin" || tab.Req.Body != "data.bin" {
		t.Fatalf("field %q body %q, want data.bin in both", a.editor.Body.File(tab.Tab).Text, tab.Req.Body)
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
	if !a.over.Browser.Shown() || !strings.Contains(layoutDump(a), `"Choose the file to send"`) {
		t.Fatalf("browser shown %v, want the file browser", a.over.Browser.Shown())
	}
	typePath(a, picked)
	a.Step(core.Input{Enter: true})
	tab := a.cur()
	if a.editor.Body.File(tab.Tab).Text != picked || tab.Req.Body != picked {
		t.Fatalf("field %q body %q, want %q in both", a.editor.Body.File(tab.Tab).Text, tab.Req.Body, picked)
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
