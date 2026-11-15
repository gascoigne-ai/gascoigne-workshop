package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/environment/controller/envs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Environments are named sets of variables (Dev, Prod...) that override
// a collection's variables while active. Each open collection has its own
// active one (collections.Entry.Env); the chip and the Vars pane show the
// focused collection's.

// activeEnv is the focused collection's active environment, or nil.
func (a *App) activeEnv() *model.Environment {
	entry, ok := a.focus()
	if !ok {
		return nil
	}
	return entry.Environment()
}

// envName is the name of the active environment, "" for none.
func (a *App) envName() string {
	if env := a.activeEnv(); env != nil {
		return env.Name
	}
	return ""
}

// vars are the variables of the focused collection, overridden by its
// active environment's.
func (a *App) vars() []model.KV {
	entry, ok := a.focus()
	if !ok {
		return nil
	}
	if env := entry.Environment(); env != nil {
		return model.MergeVars(entry.Collection.Vars, env.Vars)
	}
	return entry.Collection.Vars
}

// varsFor are the variables req is sent with: those of its own collection,
// its active environment and the folders above it (model.VarsFor), never
// another collection's.
func (a *App) varsFor(req *model.Request) []model.KV {
	entry, ok := a.entryFor(req)
	if !ok {
		return nil
	}
	return entry.VarsFor(req)
}

// varsRows are the rows the Vars pane edits. With nothing open they are a
// scratch list that is kept nowhere.
func (a *App) varsRows() *[]model.KV {
	if f := a.varsFolder; f != nil {
		return &f.Vars
	}
	if col := a.varsCollection; col != nil {
		return &col.Vars
	}
	entry, ok := a.focus()
	if !ok {
		return &[]model.KV{}
	}
	if env := entry.Environment(); env != nil {
		return &env.Vars
	}
	return &entry.Collection.Vars
}

func (a *App) varsCaption() string {
	if f := a.varsFolder; f != nil {
		return "Folder: " + f.Name
	}
	if e := a.activeEnv(); e != nil && a.varsCollection == nil {
		return "Environment: " + e.Name
	}
	return "Collection variables"
}

// chooseEnv makes environment index the active one of col, without
// remembering it.
func (a *App) chooseEnv(col *model.Collection, index int) {
	a.open = collections.Update(a.open, collections.ChooseEnv{Collection: col, Index: index})
}

// focusedSelection is the focused collection and its environment choice
// for the envs rules; ok is false when nothing is open.
func (a *App) focusedSelection() (col *model.Collection, selection envs.Selection, ok bool) {
	entry, ok := a.focus()
	return entry.Collection, envs.Selection{Index: entry.Env}, ok
}

// selectEnv makes environment i of the focused collection the active one
// (-1 for none) and remembers the choice.
func (a *App) selectEnv(i int) {
	a.commitEdit()
	col, selection, ok := a.focusedSelection()
	if !ok {
		return
	}
	selection.Select(col, i)
	a.chooseEnv(col, selection.Index)
	a.varsFolder, a.varsCollection = nil, nil
	a.rememberCollections()
}

// addEnv creates an environment in the focused collection and makes it
// active.
func (a *App) addEnv(name string) {
	a.commitEdit()
	col, selection, ok := a.focusedSelection()
	if !ok {
		return
	}
	selection.Add(col, name)
	a.chooseEnv(col, selection.Index)
	a.varsFolder, a.varsCollection = nil, nil
	a.rememberCollections()
}

func (a *App) renameEnv(i int, name string) {
	col, selection, ok := a.focusedSelection()
	if !ok {
		return
	}
	if _, changed := selection.Rename(col, i, name); changed && i == selection.Index {
		a.selectEnv(i) // the remembered name follows
	}
}

func (a *App) deleteEnv(i int) {
	a.commitEdit()
	col, selection, ok := a.focusedSelection()
	if !ok {
		return
	}
	moved := selection.Delete(col, i)
	a.chooseEnv(col, selection.Index) // a delete before the active one shifts its index
	if moved {
		a.selectEnv(selection.Index)
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
func (a *App) envMenu() []MenuItem {
	col, selection, ok := a.focusedSelection()
	if !ok {
		return nil
	}
	rows := envs.Menu(selection, col)
	items := make([]MenuItem, len(rows))
	for i, row := range rows {
		items[i] = MenuItem{Label: row.Label, Disabled: row.Disabled, Run: a.envAction(row)}
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

var envChipTemplate = markup.Must("envchip.vml", envchipSource)

// envChip is the environment switcher in the header.
func (a *App) envChip() Node {
	label := "No environment"
	if n := a.envName(); n != "" {
		label = n
	}
	return envChipTemplate.MustBuild(markup.Scope{
		"label":    label,
		"compactH": w.CompactH,
		"openMenu": func(x, y int) { a.openMenu(x, y, a.envMenu()) },
	})
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
	if a.isRoot(dir) {
		owner, _ := a.open.Owner(dir)
		a.varsFolder, a.varsCollection = nil, owner.Collection
	} else {
		a.varsFolder, a.varsCollection = dir, nil
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
		a.toast(w.PlatformKeyText(w.HostOS, "Ctrl+Space")+" completes a {{variable}} in the URL field", false)
		return
	}
	names, from := a.variableCompletion(t)
	if len(names) == 0 {
		a.toast("No variable fits here", false)
		return
	}
	items := make([]MenuItem, len(names))
	for i, name := range names {
		items[i] = MenuItem{Label: name, Run: func() { a.insertVariable(t, name, from) }}
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

// Templates, parsed once above.
const (
	envchipSource = `<!-- The chip stops at 200 px and cuts a long name, so the settings button
     beside it stays in a 400 px window; "No environment" (111 px) fits.
     Narrower, it shrinks after the title, down to its padding and a few
     pixels: a larger min-w would widen the chip of a short name. -->
<row pad-x="10" h="{{compactH}}" max-w="200" min-w="24" shrink bg="hero-text" control="pill" align-items="center" on-click-at="openMenu" tooltip="Environment: variables that override the collection's">
  <text small bold ellipsis grow="1" color="hero">{{label}}</text>
</row>
`
)
