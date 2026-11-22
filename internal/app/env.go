package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/envs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Environments are named sets of variables (Dev, Prod...) that override
// a collection's variables while active. Each open collection has its own
// active one (collections.Entry.Env); the chip and the Vars pane show the
// focused collection's.

// activeEnv is the focused collection's active environment, or nil.
func (a *App) activeEnv() *request.Environment {
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
func (a *App) vars() []request.KV {
	entry, ok := a.focus()
	if !ok {
		return nil
	}
	if env := entry.Environment(); env != nil {
		return request.MergeVars(entry.Collection.Vars, env.Vars)
	}
	return entry.Collection.Vars
}

// varsFor are the variables req is sent with: those of its own collection,
// its active environment and the folders above it (request.VarsFor), never
// another collection's.
func (a *App) varsFor(req *request.Request) []request.KV {
	entry, ok := a.entryFor(req)
	if !ok {
		return nil
	}
	return entry.VarsFor(req)
}

// varsRows are the rows the Vars pane edits. With nothing open they are a
// scratch list that is kept nowhere.
func (a *App) varsRows() *[]request.KV {
	entry, _ := a.focus() // not found is the zero Entry, which has no collection
	return a.varsTarget.rows(entry)
}

func (a *App) varsCaption() string { return a.varsTarget.caption(a.activeEnv()) }

// chooseEnv makes environment index the active one of col, without
// remembering it.
func (a *App) chooseEnv(col *request.Collection, index int) {
	a.open = collections.Update(a.open, collections.ChooseEnv{Collection: col, Index: index})
}

// focusedSelection is the focused collection and its environment choice
// for the envs rules; ok is false when nothing is open.
func (a *App) focusedSelection() (col *request.Collection, selection envs.Selection, ok bool) {
	entry, ok := a.focus()
	return entry.Collection, envs.Selection{Index: entry.Env}, ok
}

// selectEnv makes environment i of the focused collection the active one
// (-1 for none) and remembers the choice.
func (a *App) selectEnv(index int) {
	a.commitEdit()
	col, selection, ok := a.focusedSelection()
	if !ok {
		return
	}
	selection.Select(col, index)
	a.chooseEnv(col, selection.Index)
	a.varsTarget = varsTarget{}
	a.rememberEnvironments()
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
	a.varsTarget = varsTarget{}
	a.rememberEnvironments()
}

func (a *App) renameEnv(index int, name string) {
	col, selection, ok := a.focusedSelection()
	if !ok {
		return
	}
	if _, changed := selection.Rename(col, index, name); changed && index == selection.Index {
		a.selectEnv(index) // the remembered name follows
	}
}

func (a *App) deleteEnv(index int) {
	a.commitEdit()
	col, selection, ok := a.focusedSelection()
	if !ok {
		return
	}
	moved := selection.Delete(col, index)
	a.chooseEnv(col, selection.Index) // a delete before the active one shifts its index
	if moved {
		a.selectEnv(selection.Index)
	}
}

// toggleSecret marks row i of the active environment's variables as a
// secret, whose value is kept out of the collection files, or clears it.
func (a *App) toggleSecret(index int) {
	e := a.activeEnv()
	if e == nil || index < 0 || index >= len(e.Vars) {
		return
	}
	a.commitEdit()
	e.Vars[index].Secret = !e.Vars[index].Secret
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
		if row.Checked {
			items[i].Icon = "check"
		}
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
	return envChipTemplate.MustBuild(a.scope(markup.Scope{
		"label":    label,
		"openMenu": func(posX, posY int) { a.openMenu(posX, posY, a.envMenu()) },
	}))
}

// urlTip is the tooltip of the URL field: what the URL resolves to with the
// active variables, and the names that do not resolve. Empty when the URL
// uses no variables.
func (a *App) urlTip(tab *reqTab) string {
	names := request.VarNames(tab.Req.URL)
	if len(names) == 0 {
		return ""
	}
	vars := a.varsFor(tab.Req)
	tip := "Resolves to " + request.Expand(tab.Req.URL, vars)
	if missing := request.Unresolved(tab.Req.URL, vars); len(missing) > 0 {
		tip += "\nNot defined: " + strings.Join(missing, ", ")
	}
	return tip
}

// editVars points the Vars pane at the variables of dir (the collection's
// for the root) and shows it.
func (a *App) editVars(dir *request.Folder) {
	a.commitEdit()
	if a.isRoot(dir) {
		owner, _ := a.open.Owner(request.Node{Dir: dir})
		a.varsTarget = collectionVars(owner.Collection)
	} else {
		a.varsTarget = folderVars(dir)
	}
	for i, p := range requestPanesFor(a.cur()) {
		if p.label == "Vars" {
			a.cur().paneTab = i
		}
	}
}

// variableCompletion is what Ctrl+Space in the URL field offers (see
// request.VarCompletion).
func (a *App) variableCompletion(tab *reqTab) (names []string, from int) {
	return request.VarCompletion(tab.url.Text, tab.url.Cursor, a.varsFor(tab.Req))
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
		a.toast(w.PlatformKeyText(a.ctx.HostOS, "Ctrl+Space")+" completes a {{variable}} in the URL field", workspace.Info)
		return
	}
	names, from := a.variableCompletion(t)
	if len(names) == 0 {
		a.toast("No variable fits here", workspace.Info)
		return
	}
	items := make([]MenuItem, len(names))
	for i, name := range names {
		items[i] = MenuItem{Label: name, Run: func() { a.insertVariable(t, name, from) }}
	}
	a.openMenu(sidebarW+120, headerH+tabsH+urlH-w.LooseGap, items)
}

// insertVariable writes {{name}} at the cursor of the URL field, replacing
// the partly typed name that starts at rune from (-1 for none).
func (a *App) insertVariable(tab *reqTab, name string, from int) {
	runes := []rune(tab.url.Text)
	cur := min(tab.url.Cursor, len(runes))
	if from < 0 || from > cur {
		from = cur
	}
	text := string(runes[:from]) + "{{" + name + "}}" + string(runes[cur:])
	tab.url.Text = text
	tab.url.Cursor = from + len([]rune("{{"+name+"}}"))
	tab.url.HasSel = false
	tab.url.Focused = true
	a.urlEdited(tab)
}

// Templates, parsed once above.
const (
	envchipSource = `<!-- The chip stops at 200 px and cuts a long name, so the settings button
     beside it stays in a 400 px window; "No environment" (111 px) fits.
     Narrower, it shrinks after the title, down to its padding and a few
     pixels: a larger min-w would widen the chip of a short name. -->
<row pad-x="{{labelPad}}" h="{{compactH}}" max-w="{{envMaxW}}" min-w="{{envMinW}}" shrink bg="hero-text" control="pill" align-items="center" on-click-at="openMenu" tooltip="Environment: variables that override the collection's">
  <text small bold ellipsis grow="1" color="hero">{{label}}</text>
</row>
`
)
