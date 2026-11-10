package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/environment/controller/envs"
	"runtime"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Environments are named sets of variables (Dev, Prod...) that override
// the collection's variables while active. a.env is the index of the
// active one in a.col.Envs, -1 for none.

// activeEnv is the active environment, or nil.
func (a *App) activeEnv() *model.Environment { return a.env.Active(&a.col) }

// envName is the name of the active environment, "" for none.
func (a *App) envName() string { return a.env.Name(&a.col) }

// vars are the variables requests are sent with: the collection's,
// overridden by the active environment's.
func (a *App) vars() []model.KV {
	if e := a.activeEnv(); e != nil {
		return model.MergeVars(a.col.Vars, e.Vars)
	}
	return a.col.Vars
}

// varsFor are the variables req is sent with: the collection's, the active
// environment's and those of the folders above it (model.VarsFor).
func (a *App) varsFor(req *model.Request) []model.KV { return a.col.VarsFor(req, a.activeEnv()) }

// varsRows are the rows the Vars pane edits.
func (a *App) varsRows() *[]model.KV {
	if f := a.varsFolder; f != nil {
		return &f.Vars
	}
	if e := a.activeEnv(); e != nil && !a.varsCollection {
		return &e.Vars
	}
	return &a.col.Vars
}

func (a *App) varsCaption() string {
	if f := a.varsFolder; f != nil {
		return "Folder: " + f.Name
	}
	if e := a.activeEnv(); e != nil && !a.varsCollection {
		return "Environment: " + e.Name
	}
	return "Collection variables"
}

// selectEnv makes environment i the active one (-1 for none) and
// remembers the choice.
func (a *App) selectEnv(i int) {
	a.commitEdit()
	name := a.env.Select(&a.col, i)
	a.varsFolder, a.varsCollection = nil, false
	a.rememberEnv(name)
}

func (a *App) rememberEnv(name string) {
	a.saveSettings(func(s *settings.Settings) { s.Environment = name })
}

// pickEnv sets the active environment from the remembered name.
func (a *App) pickEnv() { a.env.Pick(&a.col, a.settings.Environment) }

// addEnv creates an environment and makes it active.
func (a *App) addEnv(name string) {
	a.commitEdit()
	remembered := a.env.Add(&a.col, name)
	a.varsFolder, a.varsCollection = nil, false
	a.rememberEnv(remembered)
}

func (a *App) renameEnv(i int, name string) {
	if _, changed := a.env.Rename(&a.col, i, name); changed && i == a.env.Index {
		a.selectEnv(i) // the remembered name follows
	}
}

func (a *App) deleteEnv(i int) {
	a.commitEdit()
	if a.env.Delete(&a.col, i) {
		a.selectEnv(a.env.Index)
	}
}

// toggleSecret marks row i of the active environment's variables as a
// secret, whose value is kept out of the collection files, or clears it.
func (a *App) toggleSecret(i int) {
	e := a.activeEnv()
	if e == nil || i < 0 || i >= len(e.Vars) {
		return
	}
	a.commitEdit()
	e.Vars[i].Secret = !e.Vars[i].Secret
}

// envMenu is the menu of the environment chip: envs.Menu decides the rows,
// this says what each one does.
func (a *App) envMenu() []menuItem {
	rows := envs.Menu(a.env, &a.col)
	items := make([]menuItem, len(rows))
	for i, row := range rows {
		items[i] = menuItem{label: row.Label, disabled: row.Disabled, run: a.envAction(row)}
		if row.Action == envs.Separator {
			items[i] = menuSeparator
		}
	}
	return items
}

// envAction is what a row of the chip's menu does.
func (a *App) envAction(row envs.Row) func() {
	switch row.Action {
	case envs.Select:
		return func() { a.selectEnv(row.Env) }
	case envs.New:
		return func() { a.prompt("New environment", "", "Create", a.addEnv) }
	case envs.Rename:
		return func() {
			a.prompt("Rename environment", a.envName(), "Rename", func(text string) { a.renameEnv(row.Env, text) })
		}
	case envs.Delete:
		return func() {
			a.confirm("Delete environment", "Delete \""+a.envName()+"\" and its variables?", "Delete", func() { a.deleteEnv(row.Env) })
		}
	}
	return nil
}

// envChip is the environment switcher in the header.
func (a *App) envChip() Node {
	label := "No environment"
	if n := a.envName(); n != "" {
		label = n
	}
	return Row(PadX(10), H(w.CompactH), Bg(w.HeroText), Control("pill"), AlignItems(AlignCenter),
		OnClickAt(func(x, y int) { a.openMenu(x, y, a.envMenu()) }),
		Tooltip("Environment: variables that override the collection's"),
		Text(label, Small(), Bold(), Color(w.Hero)),
	)
}

// urlTip is the tooltip of the URL field: what the URL resolves to with the
// active variables, and the names that do not resolve. Empty when the URL
// uses no variables.
func (a *App) urlTip(t *reqTab) string {
	names := model.VarNames(t.Req.URL)
	if len(names) == 0 {
		return ""
	}
	vars := a.varsFor(t.Req)
	tip := "Resolves to " + model.Expand(t.Req.URL, vars)
	if missing := model.Unresolved(t.Req.URL, vars); len(missing) > 0 {
		tip += "\nNot defined: " + strings.Join(missing, ", ")
	}
	return tip
}

// editVars points the Vars pane at the variables of dir (the collection's
// for the root) and shows it.
func (a *App) editVars(dir *model.Folder) {
	a.commitEdit()
	if dir == &a.col.Folder {
		a.varsFolder, a.varsCollection = nil, true
	} else {
		a.varsFolder, a.varsCollection = dir, false
	}
	for i, p := range requestPanesFor(a.cur()) {
		if p.label == "Vars" {
			a.cur().paneTab = i
		}
	}
}

// variableCompletion is what Ctrl+Space in the URL field offers (see
// model.VarCompletion).
func (a *App) variableCompletion(t *reqTab) (names []string, from int) {
	return model.VarCompletion(t.url.Text, t.url.Cursor, a.varsFor(t.Req))
}

// completeVariable opens a menu of the variables that fit at the cursor of
// the URL field; picking one writes {{name}} there, replacing a partly
// typed name.
func (a *App) completeVariable() {
	if a.prefs.RawFocused() {
		a.prefs.CompleteRaw(a)
		return
	}
	t := a.cur()
	if !t.url.Focused {
		a.toast(w.PlatformKeyText(runtime.GOOS, "Ctrl+Space")+" completes a {{variable}} in the URL field", false)
		return
	}
	names, from := a.variableCompletion(t)
	if len(names) == 0 {
		a.toast("No variable fits here", false)
		return
	}
	items := make([]menuItem, len(names))
	for i, name := range names {
		items[i] = menuItem{label: name, run: func() { a.insertVariable(t, name, from) }}
	}
	a.openMenu(sidebarW+120, headerH+tabsH+urlH-8, items)
}

// insertVariable writes {{name}} at the cursor of the URL field, replacing
// the partly typed name that starts at rune from (-1 for none).
func (a *App) insertVariable(t *reqTab, name string, from int) {
	runes := []rune(t.url.Text)
	cur := min(t.url.Cursor, len(runes))
	if from < 0 || from > cur {
		from = cur
	}
	text := string(runes[:from]) + "{{" + name + "}}" + string(runes[cur:])
	t.url.Text = text
	t.url.Cursor = from + len([]rune("{{"+name+"}}"))
	t.url.HasSel = false
	t.url.Focused = true
	a.urlEdited(t)
}
