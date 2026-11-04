package ui

import (
	"strconv"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Environments are named sets of variables (Dev, Prod...) that override
// the collection's variables while active. a.env is the index of the
// active one in a.col.Envs, -1 for none.

// activeEnv is the active environment, or nil.
func (a *App) activeEnv() *model.Environment {
	if a.env < 0 || a.env >= len(a.col.Envs) {
		return nil
	}
	return &a.col.Envs[a.env]
}

// envName is the name of the active environment, "" for none.
func (a *App) envName() string {
	if e := a.activeEnv(); e != nil {
		return e.Name
	}
	return ""
}

// vars are the variables requests are sent with: the collection's,
// overridden by the active environment's.
func (a *App) vars() []model.KV {
	if e := a.activeEnv(); e != nil {
		return model.MergeVars(a.col.Vars, e.Vars)
	}
	return a.col.Vars
}

// varsRows are the rows the Vars pane edits.
func (a *App) varsRows() *[]model.KV {
	if e := a.activeEnv(); e != nil {
		return &e.Vars
	}
	return &a.col.Vars
}

func (a *App) varsCaption() string {
	if e := a.activeEnv(); e != nil {
		return "Environment: " + e.Name
	}
	return "Collection variables"
}

// selectEnv makes environment i the active one (-1 for none) and
// remembers the choice.
func (a *App) selectEnv(i int) {
	a.commitEdit()
	if i < 0 || i >= len(a.col.Envs) {
		i = -1
	}
	a.env = i
	name := a.envName()
	a.saveSettings(func(s *settings.Settings) { s.Environment = name })
}

// pickEnv sets the active environment from the remembered name.
func (a *App) pickEnv() {
	a.env = -1
	for i, e := range a.col.Envs {
		if e.Name == a.settings.Environment {
			a.env = i
		}
	}
}

func (a *App) envNameTaken(name string, except int) bool {
	for i, e := range a.col.Envs {
		if i != except && e.Name == name {
			return true
		}
	}
	return false
}

func (a *App) uniqueEnvName(base string, except int) string {
	name := base
	for n := 2; a.envNameTaken(name, except); n++ {
		name = base + " " + strconv.Itoa(n)
	}
	return name
}

// addEnv creates an environment and makes it active.
func (a *App) addEnv(name string) {
	if name == "" {
		name = "Environment"
	}
	a.col.Envs = append(a.col.Envs, model.Environment{Name: a.uniqueEnvName(name, -1)})
	a.selectEnv(len(a.col.Envs) - 1)
}

func (a *App) renameEnv(i int, name string) {
	if i < 0 || i >= len(a.col.Envs) || name == "" {
		return
	}
	a.col.Envs[i].Name = a.uniqueEnvName(name, i)
	if i == a.env {
		a.selectEnv(i) // the remembered name follows
	}
}

func (a *App) deleteEnv(i int) {
	if i < 0 || i >= len(a.col.Envs) {
		return
	}
	a.commitEdit()
	a.col.Envs = append(a.col.Envs[:i], a.col.Envs[i+1:]...)
	switch {
	case a.env == i:
		a.selectEnv(-1)
	case a.env > i:
		a.selectEnv(a.env - 1)
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

// envMenu is the menu of the environment chip.
func (a *App) envMenu() []menuItem {
	mark := func(on bool, label string) string {
		if on {
			return "✓ " + label
		}
		return "   " + label
	}
	items := []menuItem{{label: mark(a.env < 0, "No environment"), run: func() { a.selectEnv(-1) }}}
	for i, e := range a.col.Envs {
		items = append(items, menuItem{label: mark(i == a.env, e.Name), run: func() { a.selectEnv(i) }})
	}
	cur := a.env
	return append(items, menuSeparator,
		menuItem{label: "New environment...", run: func() { a.prompt("New environment", "", "Create", a.addEnv) }},
		menuItem{label: "Rename...", disabled: cur < 0, run: func() {
			a.prompt("Rename environment", a.envName(), "Rename", func(t string) { a.renameEnv(cur, t) })
		}},
		menuItem{label: "Delete", disabled: cur < 0, run: func() {
			a.confirm("Delete environment", "Delete \""+a.envName()+"\" and its variables?", "Delete", func() { a.deleteEnv(cur) })
		}},
	)
}

// envChip is the environment switcher in the header.
func (a *App) envChip() Node {
	label := "No environment"
	if n := a.envName(); n != "" {
		label = n
	}
	return Row(PadXY(10, 0), H(w.CompactH), Bg(w.HeroText), Control("pill"), AlignItems(AlignCenter),
		OnClick(func() { a.openMenu(a.ctx.In.X, a.ctx.In.Y, a.envMenu()) }),
		Tooltip("Environment: variables that override the collection's"),
		Text(label, Small(), Bold(), Color(w.Hero)),
	)
}
