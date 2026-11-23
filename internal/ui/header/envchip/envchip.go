// Package envchip is the environment switcher in the header: a pill with
// the active environment's name that opens a menu to pick, add, rename or
// delete one. envs.Menu decides the rows; this package says what each one
// does through Deps, so the chip holds no state of its own.
package envchip

import (
	envs "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/environments"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// noEnvironment is the label of the chip with no environment active.
const noEnvironment = "No environment"

// Component is the chip.
type Component struct{ deps Deps }

// New returns a chip asking deps.
func New(deps Deps) *Component { return &Component{deps: deps} }

// label is the active environment's name, noEnvironment for none.
func (c *Component) label() string {
	col, selection, ok := c.deps.Envs.Focused()
	if !ok {
		return noEnvironment
	}
	if name := selection.Name(col); name != "" {
		return name
	}
	return noEnvironment
}

// menu is the menu of the chip, nil when no collection is open.
func (c *Component) menu() []widget.MenuItem {
	col, selection, ok := c.deps.Envs.Focused()
	if !ok {
		return nil
	}
	rows := envs.Menu(selection, col)
	items := make([]widget.MenuItem, len(rows))
	for i, row := range rows {
		items[i] = widget.MenuItem{Label: row.Label, Disabled: row.Disabled, Run: c.action(row)}
		if row.Checked {
			items[i].Icon = "check"
		}
		if row.Action == envs.Separator {
			items[i] = widget.MenuItem{Separator: true}
		}
	}
	return items
}

// action is what a row of the menu does. The name a dialog quotes is read
// when the row is chosen, not when the menu opened.
func (c *Component) action(row envs.Row) func() {
	switch row.Action {
	case envs.Select:
		return func() { c.deps.Envs.Select(row.Env) }
	case envs.New:
		return func() { c.deps.Dialogs.Prompt("New environment", "", "Create", c.deps.Envs.Add) }
	case envs.Rename:
		return func() {
			c.deps.Dialogs.Prompt("Rename environment", c.activeName(), "Rename", func(text string) { c.deps.Envs.Rename(row.Env, text) })
		}
	case envs.Delete:
		return func() {
			c.deps.Dialogs.Confirm("Delete environment", "Delete \""+c.activeName()+"\" and its variables?", "Delete", func() { c.deps.Envs.Delete(row.Env) })
		}
	}
	return nil
}

// activeName is the active environment's name, "" for none.
func (c *Component) activeName() string {
	col, selection, ok := c.deps.Envs.Focused()
	if !ok {
		return ""
	}
	return selection.Name(col)
}
