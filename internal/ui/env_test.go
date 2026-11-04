package ui

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
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
	st, err := settings.Load(a.settingsPath)
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
	if len(a.col.Envs) != 2 || a.col.Envs[1].Name == "Dev" || a.env != 1 {
		t.Fatalf("%+v active %d", a.col.Envs, a.env)
	}
	a.renameEnv(1, "Staging")
	if a.col.Envs[1].Name != "Staging" || a.envName() != "Staging" {
		t.Fatalf("%+v", a.col.Envs)
	}
	a.deleteEnv(1)
	if len(a.col.Envs) != 1 || a.env != -1 {
		t.Fatalf("%+v active %d", a.col.Envs, a.env)
	}
	a.selectEnv(0)
	a.deleteEnv(0)
	if len(a.col.Envs) != 0 || a.env != -1 {
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
