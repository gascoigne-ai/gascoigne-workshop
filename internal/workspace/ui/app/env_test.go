package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestActiveEnvironmentOverridesCollectionVariables(t *testing.T) {
	a, _ := newTestApp(t)
	a.col.Vars = []model.KV{{Name: "host", Value: "prod.test", Enabled: true}, {Name: "k", Value: "1", Enabled: true}}
	a.col.Envs = []model.Environment{{Name: "Dev", Vars: []model.KV{{Name: "host", Value: "dev.test", Enabled: true}}}}
	if got := model.Expand("{{host}} {{k}}", a.vars()); got != "prod.test 1" {
		t.Fatalf("no environment: %q", got)
	}
	a.selectEnv(0)
	if got := model.Expand("{{host}} {{k}}", a.vars()); got != "dev.test 1" {
		t.Fatalf("Dev: %q", got)
	}
	a.selectEnv(-1)
	if got := model.Expand("{{host}}", a.vars()); got != "prod.test" {
		t.Fatalf("none again: %q", got)
	}
}

func TestSelectedEnvironmentIsRemembered(t *testing.T) {
	a, _ := newTestApp(t)
	a.col.Envs = []model.Environment{{Name: "Dev"}, {Name: "Prod"}}
	a.selectEnv(1)
	st, err := settingsfile.Load(a.settingsPath)
	if err != nil || st.Environment != "Prod" {
		t.Fatalf("saved %q %v", st.Environment, err)
	}
	if a.envName() != "Prod" {
		t.Fatalf("name %q", a.envName())
	}
}

func TestEnvironmentLifecycle(t *testing.T) {
	a, _ := newTestApp(t)
	a.addEnv("Dev")
	a.addEnv("Dev") // a clashing name gets a suffix
	if len(a.col.Envs) != 2 || a.col.Envs[1].Name == "Dev" || a.env.Index != 1 {
		t.Fatalf("%+v active %d", a.col.Envs, a.env.Index)
	}
	a.renameEnv(1, "Staging")
	if a.col.Envs[1].Name != "Staging" || a.envName() != "Staging" {
		t.Fatalf("%+v", a.col.Envs)
	}
	a.deleteEnv(1)
	if len(a.col.Envs) != 1 || a.env.Index != -1 {
		t.Fatalf("%+v active %d", a.col.Envs, a.env.Index)
	}
	a.selectEnv(0)
	a.deleteEnv(0)
	if len(a.col.Envs) != 0 || a.env.Index != -1 {
		t.Fatal("last environment not removed")
	}
}

func TestVarsPaneEditsTheActiveEnvironment(t *testing.T) {
	a, _ := newTestApp(t)
	pane := requestPanes[4]
	if pane.label != "Vars" {
		t.Fatal("Vars pane moved")
	}
	if rows := pane.rows(a, a.cur()); rows != &a.col.Vars {
		t.Fatal("without an environment the pane edits the collection variables")
	}
	a.addEnv("Dev")
	if rows := pane.rows(a, a.cur()); rows != &a.col.Envs[0].Vars {
		t.Fatal("with an environment the pane edits its variables")
	}
	if got := pane.captionOf(a); got != "Environment: Dev" {
		t.Fatalf("caption %q", got)
	}
}

func TestUnresolvedVariablesAreFlagged(t *testing.T) {
	a, _ := newTestApp(t)
	a.col.Vars = []model.KV{{Name: "known", Value: "1", Enabled: true}}
	segs := a.varSegments("{{known}}/{{missing}}/x")
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

func TestSecretValuesAreMaskedInTheTable(t *testing.T) {
	rows := kvRows([]model.KV{{Name: "token", Value: "hunter2", Secret: true, Enabled: true}, {Name: "a", Value: "visible"}}, true, false)
	if rows[0].Cells[1] == "hunter2" || rows[0].Cells[1] == "" {
		t.Fatalf("secret shown as %q", rows[0].Cells[1])
	}
	if rows[1].Cells[1] != "visible" {
		t.Fatalf("plain value %q", rows[1].Cells[1])
	}
}

func TestToggleSecretOnAnEnvironmentRow(t *testing.T) {
	a, _ := newTestApp(t)
	a.addEnv("Dev")
	a.col.Envs[0].Vars = []model.KV{{Name: "token", Value: "x", Enabled: true}}
	a.toggleSecret(0)
	if !a.col.Envs[0].Vars[0].Secret {
		t.Fatal("not marked")
	}
	a.toggleSecret(0)
	if a.col.Envs[0].Vars[0].Secret {
		t.Fatal("not unmarked")
	}
}

func TestURLTooltipShowsTheResolvedURLAndWhatIsMissing(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.URL = "https://plain.test/x"
	if a.urlTip(tab) != "" {
		t.Fatal("a URL without variables needs no tooltip")
	}
	a.col.Vars = []model.KV{{Name: "host", Value: "api.test", Enabled: true}}
	tab.Req.URL = "https://{{host}}/{{path}}"
	tip := a.urlTip(tab)
	if !strings.Contains(tip, "Resolves to https://api.test/{{path}}") || !strings.Contains(tip, "Not defined: path") {
		t.Fatalf("tip %q", tip)
	}
}

func TestFolderVariablesAreEditedInTheVarsPaneAndResolveForTheirRequests(t *testing.T) {
	a, _ := newTestApp(t)
	folder := a.col.Folders[1] // holds the open request
	req := a.cur().Req
	a.col.Vars = []model.KV{{Name: "host", Value: "collection.test", Enabled: true}}
	a.editVars(folder)
	pane := requestPanes[a.cur().paneTab]
	if pane.label != "Vars" || pane.rows(a, a.cur()) != &folder.Vars || pane.captionOf(a) != "Folder: "+folder.Name {
		t.Fatalf("pane %q caption %q", pane.label, pane.captionOf(a))
	}
	folder.Vars = []model.KV{{Name: "host", Value: "folder.test", Enabled: true}}
	if got := model.Expand("{{host}}", a.varsFor(req)); got != "folder.test" {
		t.Fatalf("folder variable not used: %q", got)
	}
	a.editVars(&a.col.Folder)
	if requestPanes[a.cur().paneTab].rows(a, a.cur()) != &a.col.Vars {
		t.Fatal("the root edits the collection's variables")
	}
	a.addEnv("Dev") // choosing an environment goes back to the usual pane
	if requestPanes[a.cur().paneTab].rows(a, a.cur()) != &a.col.Envs[0].Vars {
		t.Fatal("selecting an environment resets the Vars pane")
	}
}

func TestCtrlSpaceCompletesVariablesInTheURLField(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	a.col.Vars = []model.KV{
		{Name: "host", Value: "h", Enabled: true}, {Name: "hostname", Value: "n", Enabled: true},
		{Name: "token", Value: "t", Enabled: true}, {Name: "off", Value: "x"},
	}
	tab.url = view.NewTextField("https://{{ho")
	tab.url.Focused = true
	names, from := a.variableCompletion(tab)
	if from != len("https://") || len(names) != 2 || names[0] != "host" || names[1] != "hostname" {
		t.Fatalf("names %v from %d", names, from)
	}
	a.completeVariable()
	if widget.PopupMenu(a.ctx) == nil || len(widget.PopupMenu(a.ctx).Items) != 2 {
		t.Fatal("no completion menu")
	}
	widget.PopupMenu(a.ctx).Items[1].Run() // hostname
	if tab.url.Text != "https://{{hostname}}" || tab.Req.URL != "https://{{hostname}}" || tab.url.Cursor != len("https://{{hostname}}") {
		t.Fatalf("url %q req %q cursor %d", tab.url.Text, tab.Req.URL, tab.url.Cursor)
	}

	tab.url = view.NewTextField("https://x/")
	tab.url.Focused = true
	names, from = a.variableCompletion(tab)
	if from != -1 || len(names) != 3 { // outside braces every enabled variable fits, and only enabled ones
		t.Fatalf("names %v from %d", names, from)
	}
	a.insertVariable(tab, "token", from)
	if tab.url.Text != "https://x/{{token}}" {
		t.Fatalf("url %q", tab.url.Text)
	}
	tab.url = view.NewTextField("a")
	tab.url.Focused = false
	a.closeMenu()
	a.completeVariable()
	if widget.PopupMenu(a.ctx) != nil {
		t.Fatal("completion outside the URL field must not open a menu")
	}
}
