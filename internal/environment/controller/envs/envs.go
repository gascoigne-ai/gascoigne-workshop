// Package envs is the state and the decisions behind the environment
// chip: which environment of a collection is active, and how adding,
// renaming and deleting move that choice. Drawing the chip and asking for
// names is the application's job.
package envs

import (
	"strconv"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// DefaultName names an environment created without a name.
const DefaultName = "Environment"

// Selection is the active environment of a collection: an index into its
// Envs, -1 for none. The zero value selects the first one, so start with
// Selection{Index: -1}.
type Selection struct{ Index int }

// None is the selection of no environment.
func None() Selection { return Selection{Index: -1} }

// Active is the selected environment, or nil.
func (sel Selection) Active(col *model.Collection) *model.Environment {
	if sel.Index < 0 || sel.Index >= len(col.Envs) {
		return nil
	}
	return &col.Envs[sel.Index]
}

// Name is the name of the selected environment, "" for none.
func (sel Selection) Name(col *model.Collection) string {
	if env := sel.Active(col); env != nil {
		return env.Name
	}
	return ""
}

// Select makes environment i the active one, none when i is out of range,
// and returns the name to remember.
func (sel *Selection) Select(col *model.Collection, i int) string {
	if i < 0 || i >= len(col.Envs) {
		i = -1
	}
	sel.Index = i
	return sel.Name(col)
}

// Pick selects the environment called name, none when there is no such one.
func (sel *Selection) Pick(col *model.Collection, name string) {
	sel.Index = -1
	for i, env := range col.Envs {
		if env.Name == name {
			sel.Index = i
		}
	}
}

// UniqueName is base, or base with " 2", " 3"... when another environment
// than the one at except already has it.
func UniqueName(envs []model.Environment, base string, except int) string {
	name := base
	for n := 2; taken(envs, name, except); n++ {
		name = base + " " + strconv.Itoa(n)
	}
	return name
}

func taken(envs []model.Environment, name string, except int) bool {
	for i, env := range envs {
		if i != except && env.Name == name {
			return true
		}
	}
	return false
}

// Add creates an environment, selects it and returns the name to remember.
func (sel *Selection) Add(col *model.Collection, name string) string {
	if name == "" {
		name = DefaultName
	}
	col.Envs = append(col.Envs, model.Environment{Name: UniqueName(col.Envs, name, -1)})
	return sel.Select(col, len(col.Envs)-1)
}

// Rename gives environment i a unique name based on name and reports it;
// changed is false for an empty name or an unknown environment.
func (sel Selection) Rename(col *model.Collection, i int, name string) (renamed string, changed bool) {
	if i < 0 || i >= len(col.Envs) || name == "" {
		return "", false
	}
	col.Envs[i].Name = UniqueName(col.Envs, name, i)
	return col.Envs[i].Name, true
}

// Delete removes environment i and keeps the selection on the same
// environment, or on none when it was the one removed. selectionMoved
// reports that the remembered name has to be updated.
func (sel *Selection) Delete(col *model.Collection, i int) (selectionMoved bool) {
	if i < 0 || i >= len(col.Envs) {
		return false
	}
	col.Envs = append(col.Envs[:i], col.Envs[i+1:]...)
	switch {
	case sel.Index == i:
		sel.Index = -1
		return true
	case sel.Index > i:
		sel.Index--
		return true
	}
	return false
}
