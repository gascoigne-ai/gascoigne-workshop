package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestActiveEnvironmentOverridesCollectionVariables(t *testing.T) {
	a := testApp(t)
	a.col().Vars = []request.KV{{Name: "host", Value: "prod.test", Enabled: true}, {Name: "k", Value: "1", Enabled: true}}
	a.col().Envs = []request.Environment{{Name: "Dev", Vars: []request.KV{{Name: "host", Value: "dev.test", Enabled: true}}}}
	if got := request.Expand("{{host}} {{k}}", a.vars()); got != "prod.test 1" {
		t.Fatalf("no environment: %q", got)
	}
	a.selectEnv(0)
	if got := request.Expand("{{host}} {{k}}", a.vars()); got != "dev.test 1" {
		t.Fatalf("Dev: %q", got)
	}
	a.selectEnv(-1)
	if got := request.Expand("{{host}}", a.vars()); got != "prod.test" {
		t.Fatalf("none again: %q", got)
	}
}

func TestSelectedEnvironmentIsRemembered(t *testing.T) {
	a := testApp(t)
	a.col().Envs = []request.Environment{{Name: "Dev"}, {Name: "Prod"}}
	a.selectEnv(1)
	st, err := settingsDisk.Load(a.SettingsPath())
	if got := st.RememberedCollections(); err != nil || len(got) != 1 || got[0].Environment != "Prod" {
		t.Fatalf("saved %+v %v", got, err)
	}
	if a.envName() != "Prod" {
		t.Fatalf("name %q", a.envName())
	}
}

func TestEnvironmentLifecycle(t *testing.T) {
	a := testApp(t)
	a.addEnv("Dev")
	a.addEnv("Dev") // a clashing name gets a suffix
	if len(a.col().Envs) != 2 || a.col().Envs[1].Name == "Dev" || a.envIndex() != 1 {
		t.Fatalf("%+v active %d", a.col().Envs, a.envIndex())
	}
	a.renameEnv(1, "Staging")
	if a.col().Envs[1].Name != "Staging" || a.envName() != "Staging" {
		t.Fatalf("%+v", a.col().Envs)
	}
	a.deleteEnv(1)
	if len(a.col().Envs) != 1 || a.envIndex() != -1 {
		t.Fatalf("%+v active %d", a.col().Envs, a.envIndex())
	}
	a.selectEnv(0)
	a.deleteEnv(0)
	if len(a.col().Envs) != 0 || a.envIndex() != -1 {
		t.Fatal("last environment not removed")
	}
}

func TestVarsPaneEditsTheActiveEnvironment(t *testing.T) {
	a := testApp(t)
	pane := requestPanes[4]
	if pane.label != "Vars" {
		t.Fatal("Vars pane moved")
	}
	if rows := pane.rows(a, a.cur()); rows != &a.col().Vars {
		t.Fatal("without an environment the pane edits the collection variables")
	}
	a.addEnv("Dev")
	if rows := pane.rows(a, a.cur()); rows != &a.col().Envs[0].Vars {
		t.Fatal("with an environment the pane edits its variables")
	}
	if got := pane.captionOf(a); got != "Environment: Dev" {
		t.Fatalf("caption %q", got)
	}
}

func TestUnresolvedVariablesAreFlagged(t *testing.T) {
	a := testApp(t)
	a.col().Vars = []request.KV{{Name: "known", Value: "1", Enabled: true}}
	segs := a.urlSegments(a.cur().Req, "{{known}}/{{missing}}/x")
	want := []w.Token{w.OK, w.Text, w.Error, w.Text}
	if len(segs) != 4 {
		t.Fatalf("%+v", segs)
	}
	for i, s := range segs {
		if s.Style != want[i] && !(i == 1 || i == 3) {
			t.Fatalf("segment %d (%q) style %s", i, s.Text, s.Style)
		}
	}
	if segs[0].Style != w.OK || segs[2].Style != w.Error {
		t.Fatalf("%+v", segs)
	}
}

func TestToggleSecretOnAnEnvironmentRow(t *testing.T) {
	a := testApp(t)
	a.addEnv("Dev")
	a.col().Envs[0].Vars = []request.KV{{Name: "token", Value: "x", Enabled: true}}
	a.toggleSecret(0)
	if !a.col().Envs[0].Vars[0].Secret {
		t.Fatal("not marked")
	}
	a.toggleSecret(0)
	if a.col().Envs[0].Vars[0].Secret {
		t.Fatal("not unmarked")
	}
}

func TestURLTooltipShowsTheResolvedURLAndWhatIsMissing(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.URL = "https://plain.test/x"
	if a.urlTip(tab) != "" {
		t.Fatal("a URL without variables needs no tooltip")
	}
	a.col().Vars = []request.KV{{Name: "host", Value: "api.test", Enabled: true}}
	tab.Req.URL = "https://{{host}}/{{path}}"
	tip := a.urlTip(tab)
	if !strings.Contains(tip, "Resolves to https://api.test/{{path}}") || !strings.Contains(tip, "Not defined: path") {
		t.Fatalf("tip %q", tip)
	}
}

func TestFolderVariablesAreEditedInTheVarsPaneAndResolveForTheirRequests(t *testing.T) {
	a := testApp(t)
	folder := a.col().Folders[1] // holds the open request
	req := a.cur().Req
	a.col().Vars = []request.KV{{Name: "host", Value: "collection.test", Enabled: true}}
	a.editVars(folder)
	pane := requestPanes[a.cur().paneTab]
	if pane.label != "Vars" || pane.rows(a, a.cur()) != &folder.Vars || pane.captionOf(a) != "Folder: "+folder.Name {
		t.Fatalf("pane %q caption %q", pane.label, pane.captionOf(a))
	}
	folder.Vars = []request.KV{{Name: "host", Value: "folder.test", Enabled: true}}
	if got := request.Expand("{{host}}", a.varsFor(req)); got != "folder.test" {
		t.Fatalf("folder variable not used: %q", got)
	}
	a.editVars(&a.col().Folder)
	if requestPanes[a.cur().paneTab].rows(a, a.cur()) != &a.col().Vars {
		t.Fatal("the root edits the collection's variables")
	}
	a.addEnv("Dev") // choosing an environment goes back to the usual pane
	if requestPanes[a.cur().paneTab].rows(a, a.cur()) != &a.col().Envs[0].Vars {
		t.Fatal("selecting an environment resets the Vars pane")
	}
}

func TestCtrlSpaceCompletesVariablesInTheURLField(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	a.col().Vars = []request.KV{
		{Name: "host", Value: "h", Enabled: true}, {Name: "hostname", Value: "n", Enabled: true},
		{Name: "token", Value: "t", Enabled: true}, {Name: "off", Value: "x"},
	}
	*a.editor.URLBar.Field(tab.Tab) = widget.NewTextField("https://{{ho")
	a.editor.URLBar.Field(tab.Tab).Focused = true
	names, from := a.variableCompletion(tab)
	if from != len("https://") || len(names) != 2 || names[0] != "host" || names[1] != "hostname" {
		t.Fatalf("names %v from %d", names, from)
	}
	a.completeVariable()
	if widget.PopupMenu(a.ctx) == nil || len(widget.PopupMenu(a.ctx).Items) != 2 {
		t.Fatal("no completion menu")
	}
	widget.PopupMenu(a.ctx).Items[1].Run() // hostname
	if a.editor.URLBar.Field(tab.Tab).Text != "https://{{hostname}}" || tab.Req.URL != "https://{{hostname}}" || a.editor.URLBar.Field(tab.Tab).Cursor != len("https://{{hostname}}") {
		t.Fatalf("url %q req %q cursor %d", a.editor.URLBar.Field(tab.Tab).Text, tab.Req.URL, a.editor.URLBar.Field(tab.Tab).Cursor)
	}

	*a.editor.URLBar.Field(tab.Tab) = widget.NewTextField("https://x/")
	a.editor.URLBar.Field(tab.Tab).Focused = true
	names, from = a.variableCompletion(tab)
	if from != -1 || len(names) != 3 { // outside braces every enabled variable fits, and only enabled ones
		t.Fatalf("names %v from %d", names, from)
	}
	a.insertVariable(tab, "token", from)
	if a.editor.URLBar.Field(tab.Tab).Text != "https://x/{{token}}" {
		t.Fatalf("url %q", a.editor.URLBar.Field(tab.Tab).Text)
	}
	*a.editor.URLBar.Field(tab.Tab) = widget.NewTextField("a")
	a.editor.URLBar.Field(tab.Tab).Focused = false
	a.closeMenu()
	a.completeVariable()
	if widget.PopupMenu(a.ctx) != nil {
		t.Fatal("completion outside the URL field must not open a menu")
	}
}
